package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/storage"
)

type Server struct {
	Storage storage.Storage
	DB      *sql.DB
}

func NewServer(storage storage.Storage, db *sql.DB) *Server {
	return &Server{
		Storage: storage,
		DB:      db,
	}
}

func (s *Server) PingHandler(w http.ResponseWriter, r *http.Request) {

	if err := s.DB.PingContext(r.Context()); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		s.UpdateJSONMetrics(r.Context(), w, r)
	case strings.Contains(contentType, "text/plain"), contentType == "":
		s.UpdateTextMetrics(w, r)
	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
	}
}

func (s *Server) UpdateJSONMetrics(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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
	s.respondWithMetric(ctx, w, metrics)
}

func (s *Server) UpdateTextMetrics(w http.ResponseWriter, r *http.Request) {
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

	s.respondWithMetric(r.Context(), w, metric)
}

func (s *Server) GetValue(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	var response interface{}

	switch {
	case strings.Contains(contentType, "application/json"):
		metrics := s.GetJSONValue(r.Context(), w, r)
		if metrics.ID == "" {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		jsonData, err := json.Marshal(metrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonData)
		return
	default:
		metrics := s.GetTextValue(r.Context(), w, r)
		if metrics.ID == "" {
			return
		}
		switch metrics.MType {
		case models.Gauge:
			if metrics.Value != nil {
				response = *metrics.Value
			}
		case models.Counter:
			if metrics.Delta != nil {
				response = *metrics.Delta
			}
		}
	}

	if response == nil {
		http.Error(w, "Not Found in GetValue", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, response)
}

func (s *Server) GetJSONValue(ctx context.Context, w http.ResponseWriter, r *http.Request) models.Metric {
	var metrics models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return models.Metric{}
	}

	switch metrics.MType {
	case models.Gauge:
		value, err := s.Storage.GetGauge(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found  gauge in GetJSONValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, err := s.Storage.GetCounter(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found  coutner in GetJSONValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Delta = delta
	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
	}
	return metrics
}

func (s *Server) GetTextValue(ctx context.Context, w http.ResponseWriter, r *http.Request) models.Metric {
	var metrics models.Metric

	metrics.ID = chi.URLParam(r, "name")
	metrics.MType = chi.URLParam(r, "type")

	switch metrics.MType {
	case models.Gauge:
		value, err := s.Storage.GetGauge(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found gauge value in GetTextValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, err := s.Storage.GetCounter(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found conter value in GetTextValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Delta = delta
	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
		return models.Metric{}
	}
	return metrics
}

func (s *Server) UpdateBatchMetrics(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		http.Error(w, "Bad Request: empty metrics batch", http.StatusBadRequest)
		return
	}
	for _, metric := range metrics {
		if err := validateMetric(metric); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if s.DB != nil {
		if err := s.execInTransactionWithRetry(r.Context(), func(tx *sql.Tx) error {
			for _, metric := range metrics {
				if err := updateMetricInTx(tx, metric); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			if s.Storage.IsUniqueViolationError(err) {
				http.Error(w, "Conflict: unique violation", http.StatusConflict)
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
	}
	if s.DB == nil {
		for _, metric := range metrics {
			if err := s.updateMetric(r.Context(), metric); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	updatedMetrics, err := s.getUpdatedMetrics(r.Context(), metrics)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedMetrics); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	var response strings.Builder

	response.WriteString("<html><body><h1>Metrics</h1><ul>")

	gauges, err := s.Storage.Gauges(r.Context())
	if err == nil {
		for name, value := range gauges {
			response.WriteString(fmt.Sprintf("<li>%s: %s</li>", name, helpers.FormatFloat(value)))
		}
	}

	counters, err := s.Storage.Counters(r.Context())
	if err == nil {
		for name, value := range counters {
			response.WriteString(fmt.Sprintf("<li>%s: %d</li>", name, value))
		}
	}

	response.WriteString("</ul></body></html>")

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response.String()))

}

func (s *Server) respondWithMetric(ctx context.Context, w http.ResponseWriter, metric models.Metric) {
	switch metric.MType {
	case models.Gauge:
		value, err := s.Storage.GetGauge(ctx, metric.ID)
		if err != nil {
			metric.Value = value
		}
	case models.Counter:
		delta, err := s.Storage.GetCounter(ctx, metric.ID)
		if err != nil {
			metric.Delta = delta
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
func (s *Server) updateMetric(ctx context.Context, metric models.Metric) error {
	switch metric.MType {
	case models.Gauge:
		return s.Storage.SetGauge(ctx, metric.ID, *metric.Value)
	case models.Counter:
		return s.Storage.AddCounter(ctx, metric.ID, *metric.Delta)
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
			return errors.New("Bad Request: value is required for gauge")
		}
	case models.Counter:
		if metric.Delta == nil {
			return errors.New("Bad Request: delta is required for counter")
		}
	default:
		return errors.New("Bad Request: invalid metric type")
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
			value, err := s.Storage.GetGauge(ctx, metric.ID)
			if err != nil {
				return nil, err
			}
			updatedMetric.Value = value
		case models.Counter:
			delta, err := s.Storage.GetCounter(ctx, metric.ID)
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
