package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/models"
)

func (s *Server) updateMetric(ctx context.Context, metric models.Metric) error {
	switch metric.MType {
	case models.Gauge:
		return s.storage.SetGauge(ctx, metric.ID, *metric.Value)
	case models.Counter:
		return s.storage.AddCounter(ctx, metric.ID, *metric.Delta)
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

func (s *Server) getUpdatedMetrics(ctx context.Context, metrics []models.Metric) ([]models.Metric, error) {
	var updatedMetrics []models.Metric
	for _, metric := range metrics {
		var updatedMetric models.Metric
		updatedMetric.ID = metric.ID
		updatedMetric.MType = metric.MType

		switch metric.MType {
		case models.Gauge:
			value, err := s.storage.GetGauge(ctx, metric.ID)
			if err != nil {
				return nil, err
			}
			updatedMetric.Value = value
		case models.Counter:
			delta, err := s.storage.GetCounter(ctx, metric.ID)
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

func (s *Server) respondWithMetric(ctx context.Context, w http.ResponseWriter, metric models.Metric, key string) {
	switch metric.MType {
	case models.Gauge:
		value, err := s.storage.GetGauge(ctx, metric.ID)
		if err != nil {
			metric.Value = value
		}
	case models.Counter:
		delta, err := s.storage.GetCounter(ctx, metric.ID)
		if err != nil {
			metric.Delta = delta
		}
	}

	jsonData, err := json.Marshal(metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.setResponseHash(w, jsonData, key)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) setResponseHash(w http.ResponseWriter, data []byte, key string) {
	if key == "" {
		return
	}
	hash := helpers.CalculateHash(data, key)
	w.Header().Set("HashSHA256", hash)
}

func (s *Server) updateJSONMetrics(ctx context.Context, w http.ResponseWriter, r *http.Request, key string) {
	var metrics models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}
	if err := validateMetric(metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.updateMetric(ctx, metrics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.respondWithMetric(ctx, w, metrics, key)
}

func (s *Server) updateTextMetrics(w http.ResponseWriter, r *http.Request, key string) {
	metric := models.Metric{
		ID:    chi.URLParam(r, "name"),
		MType: chi.URLParam(r, "type"),
	}

	switch metric.MType {
	case models.Gauge:
		valueStr := chi.URLParam(r, "value")
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid gauge value", http.StatusBadRequest)
			return
		}
		metric.Value = &value
	case models.Counter:
		valueStr := chi.URLParam(r, "value")
		delta, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid counter value", http.StatusBadRequest)
			return
		}
		metric.Delta = &delta
	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
		return
	}

	if err := s.updateMetric(r.Context(), metric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.respondWithMetric(r.Context(), w, metric, key)
}

func (s *Server) getJSONValue(ctx context.Context, w http.ResponseWriter, r *http.Request) models.Metric {
	var metrics models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return models.Metric{}
	}

	switch metrics.MType {
	case models.Gauge:
		value, err := s.storage.GetGauge(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found  gauge in getJSONValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, err := s.storage.GetCounter(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found  coutner in getJSONValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Delta = delta
	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
	}
	return metrics
}

func (s *Server) getTextValue(ctx context.Context, w http.ResponseWriter, r *http.Request) models.Metric {
	var metrics models.Metric

	metrics.ID = chi.URLParam(r, "name")
	metrics.MType = chi.URLParam(r, "type")

	switch metrics.MType {
	case models.Gauge:
		value, err := s.storage.GetGauge(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found gauge value in getTextValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, err := s.storage.GetCounter(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found conter value in getTextValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Delta = delta
	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
		return models.Metric{}
	}
	return metrics
}

func (s *Server) handleRetry(ctx context.Context, attempt int, retryDelays [helpers.MaxRetries]time.Duration, lastErr error) error {
	if attempt < helpers.MaxRetries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelays[attempt]):
			return nil
		}
	}
	return fmt.Errorf("after %d attempts: %w", helpers.MaxRetries, lastErr)
}
