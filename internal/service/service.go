// Package service contains the business logic for metrics operations.
// It provides a layer between storage and handlers, handling metric validation,
// batch processing, transactions, and data conversion.
package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/storage"
)

// MetricsService encapsulates business logic for metric operations including
// validation, batch updates, and storage coordination. It supports both
// in-memory and database storage with transactional integrity.
type MetricsService struct {
	storage storage.Storage
	db      *sql.DB
}

// NewMetricsService creates a new instance of MetricsService with the given storage and database.
func NewMetricsService(storage storage.Storage, db *sql.DB) *MetricsService {
	return &MetricsService{
		storage: storage,
		db:      db,
	}
}

// UpdateMetric updates a single metric with validation based on metric type.
func (s *MetricsService) UpdateMetric(ctx context.Context, metric models.Metric) error {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return fmt.Errorf("value is required for gauge metric")
		}
		return s.storage.SetGauge(ctx, metric.ID, *metric.Value)
	case models.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("delta is required for counter metric")
		}
		return s.storage.AddCounter(ctx, metric.ID, *metric.Delta)
	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}
}

// UpdateMetricsBatch updates multiple metrics in a single operation.
func (s *MetricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metric) error {
	if len(metrics) == 0 {
		return fmt.Errorf("empty metrics batch")
	}

	if s.db != nil {
		return s.updateMetricsBatchInTransaction(ctx, metrics)
	}

	return s.updateMetricsBatchInMemory(ctx, metrics)
}

func (s *MetricsService) updateMetricsBatchInTransaction(ctx context.Context, metrics []models.Metric) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, metric := range metrics {
		if err := s.updateMetricInTx(tx, metric); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *MetricsService) updateMetricsBatchInMemory(ctx context.Context, metrics []models.Metric) error {
	for _, metric := range metrics {
		if err := s.UpdateMetric(ctx, metric); err != nil {
			return err
		}
	}
	return nil
}

func (s *MetricsService) updateMetricInTx(tx *sql.Tx, metric models.Metric) error {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return fmt.Errorf("gauge value is required")
		}
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO gauges (name, value)
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
		`, metric.ID, *metric.Value)
		return err
	case models.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("counter delta is required")
		}
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO counters (name, value)
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
		`, metric.ID, *metric.Delta)
		return err
	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}
}

// GetMetric retrieves a specific metric by type and name.
func (s *MetricsService) GetMetric(ctx context.Context, metricType, metricName string) (*models.Metric, error) {
	metric := &models.Metric{
		ID:    metricName,
		MType: metricType,
	}

	switch metricType {
	case models.Gauge:
		value, err := s.storage.GetGauge(ctx, metricName)
		if err != nil {
			return nil, err
		}
		metric.Value = value
	case models.Counter:
		delta, err := s.storage.GetCounter(ctx, metricName)
		if err != nil {
			return nil, err
		}
		metric.Delta = delta
	default:
		return nil, fmt.Errorf("unknown metric type: %s", metricType)
	}

	return metric, nil
}

// GetAllMetrics retrieves all stored metrics including both gauges and counters.
func (s *MetricsService) GetAllMetrics(ctx context.Context) ([]models.Metric, error) {
	var metrics []models.Metric

	gauges, err := s.storage.Gauges(ctx)
	if err != nil {
		return nil, fmt.Errorf("get gauges: %w", err)
	}

	for name, value := range gauges {
		val := value
		metrics = append(metrics, models.Metric{
			ID:    name,
			MType: models.Gauge,
			Value: &val,
		})
	}

	counters, err := s.storage.Counters(ctx)
	if err != nil {
		return nil, fmt.Errorf("get counters: %w", err)
	}

	for name, value := range counters {
		val := value
		metrics = append(metrics, models.Metric{
			ID:    name,
			MType: models.Counter,
			Delta: &val,
		})
	}

	return metrics, nil
}

// Ping checks the availability of the underlying storage.
func (s *MetricsService) Ping(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	return s.db.PingContext(ctx)
}
