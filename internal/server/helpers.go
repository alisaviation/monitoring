package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/models"
)

func (p *Server) updateMetric(ctx context.Context, metric models.Metric) error {
	switch metric.MType {
	case models.Gauge:
		return p.Storage.SetGauge(ctx, metric.ID, *metric.Value)
	case models.Counter:
		return p.Storage.AddCounter(ctx, metric.ID, *metric.Delta)
	default:
		return &helpers.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "Bad Request: invalid metric type",
		}
	}
}

func validateMetric(metric models.Metric) error {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return errors.New("bad Request: value is required for gauge")
		}
	case models.Counter:
		if metric.Delta == nil {
			return errors.New("bad Request: delta is required for counter")
		}
	default:
		return errors.New("bad Request: invalid metric type")
	}
	return nil
}

func (p *Server) getUpdatedMetrics(ctx context.Context, metrics []models.Metric) ([]models.Metric, error) {
	var updatedMetrics []models.Metric
	for _, metric := range metrics {
		var updatedMetric models.Metric
		updatedMetric.ID = metric.ID
		updatedMetric.MType = metric.MType

		switch metric.MType {
		case models.Gauge:
			value, err := p.Storage.GetGauge(ctx, metric.ID)
			if err != nil {
				return nil, err
			}
			updatedMetric.Value = value
		case models.Counter:
			delta, err := p.Storage.GetCounter(ctx, metric.ID)
			if err != nil {
				return nil, err
			}
			updatedMetric.Delta = delta
		default:
			return nil, fmt.Errorf("invalid metric type")
		}

		updatedMetrics = append(updatedMetrics, updatedMetric)
	}
	return updatedMetrics, nil
}

func (p *Server) respondWithMetric(ctx context.Context, w http.ResponseWriter, metric models.Metric, key string) {
	switch metric.MType {
	case models.Gauge:
		value, err := p.Storage.GetGauge(ctx, metric.ID)
		if err != nil {
			metric.Value = value
		}
	case models.Counter:
		delta, err := p.Storage.GetCounter(ctx, metric.ID)
		if err != nil {
			metric.Delta = delta
		}
	}

	jsonData, err := json.Marshal(metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.setResponseHash(w, jsonData, key)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (p *Server) setResponseHash(w http.ResponseWriter, data []byte, key string) {
	if key == "" {
		return
	}
	hash := helpers.CalculateHash(data, key)
	w.Header().Set("HashSHA256", hash)
}

func (p *Server) getKeyFromContext(ctx context.Context) string {
	if val, ok := ctx.Value("key").(string); ok {
		return val
	}
	return ""
}
