package server

import (
	"context"
	"database/sql"
	"encoding/json"
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
	if s.DB == nil {
		http.Error(w, "Database connection not initialized", http.StatusInternalServerError)
		return
	}

	if err := s.DB.Ping(); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		s.UpdateJSONMetrics(w, r)
	case strings.Contains(contentType, "text/plain"), contentType == "":
		s.UpdateTextMetrics(w, r)
	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
	}
}

func (s *Server) UpdateJSONMetrics(w http.ResponseWriter, r *http.Request) {
	var metrics models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}

	switch metrics.MType {
	case models.Gauge:
		if metrics.Value == nil {
			http.Error(w, "Bad Request: value is required for gauge", http.StatusBadRequest)
			return
		}
		if err := s.Storage.SetGauge(metrics.ID, *metrics.Value); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		metrics.Value, _ = s.Storage.GetGauge(metrics.ID)

	case models.Counter:
		if metrics.Delta == nil {
			http.Error(w, "Bad Request: delta is required for counter", http.StatusBadRequest)
			return
		}
		if err := s.Storage.AddCounter(metrics.ID, *metrics.Delta); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		metrics.Delta, _ = s.Storage.GetCounter(metrics.ID)

	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(jsonData)
}

func (s *Server) UpdateTextMetrics(w http.ResponseWriter, r *http.Request) {

	var metrics models.Metric
	valueStr := chi.URLParam(r, "value")
	metrics.ID = chi.URLParam(r, "name")
	metrics.MType = chi.URLParam(r, "type")

	switch metrics.MType {
	case models.Gauge:
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid gauge value", http.StatusBadRequest)
			return
		}
		if err := s.Storage.SetGauge(metrics.ID, value); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		metrics.Value, _ = s.Storage.GetGauge(metrics.ID)

	case models.Counter:
		delta, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid counter value", http.StatusBadRequest)
			return
		}
		if err := s.Storage.AddCounter(metrics.ID, delta); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		metrics.Delta, _ = s.Storage.GetCounter(metrics.ID)

	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

	}
	w.Write(jsonData)
}

func (s *Server) GetValue(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	var response interface{}

	switch {
	case strings.Contains(contentType, "application/json"):
		metrics := s.GetJSONValue(w, r)
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
		metrics := s.GetTextValue(w, r)
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

func (s *Server) GetJSONValue(w http.ResponseWriter, r *http.Request) models.Metric {
	var metrics models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return models.Metric{}
	}

	switch metrics.MType {
	case models.Gauge:
		value, exists := s.Storage.GetGauge(metrics.ID)
		if !exists {
			http.Error(w, "Not Found  gauge in GetJSONValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, exists := s.Storage.GetCounter(metrics.ID)
		if !exists {
			http.Error(w, "Not Found  couner in GetJSONValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Delta = delta
	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
	}
	return metrics
}

func (s *Server) GetTextValue(w http.ResponseWriter, r *http.Request) models.Metric {
	var metrics models.Metric

	metrics.ID = chi.URLParam(r, "name")
	metrics.MType = chi.URLParam(r, "type")

	switch metrics.MType {
	case models.Gauge:
		value, exists := s.Storage.GetGauge(metrics.ID)
		if !exists {
			http.Error(w, "Not Found gauge value in GetTextValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, exists := s.Storage.GetCounter(metrics.ID)
		if !exists {
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

func (s *Server) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	var response strings.Builder

	response.WriteString("<html><body><h1>Metrics</h1><ul>")

	gauges, err := s.Storage.Gauges()
	if err == nil {
		for name, value := range gauges {
			response.WriteString(fmt.Sprintf("<li>%s: %s</li>", name, helpers.FormatFloat(value)))
		}
	}

	counters, err := s.Storage.Counters()
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

	if s.DB != nil {
		ctx := context.Background()
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			http.Error(w, "Internal Server Error in tx updateBatch", http.StatusInternalServerError)
			return
		}
		defer func(tx *sql.Tx) {
			err := tx.Rollback()
			if err != nil {
				http.Error(w, "Internal Server Error in tx updateBatch", http.StatusInternalServerError)
			}
		}(tx)

		for _, metric := range metrics {
			if err := updateMetricInTx(tx, metric); err != nil {
				http.Error(w, "Internal Server Error in tx updateBatch", http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Error ", http.StatusInternalServerError)
			return
		}
	} else {
		for _, metric := range metrics {
			switch metric.MType {
			case models.Gauge:
				if metric.Value == nil {
					http.Error(w, "Bad Request: value is required for gauge", http.StatusBadRequest)
					return
				}
				if err := s.Storage.SetGauge(metric.ID, *metric.Value); err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			case models.Counter:
				if metric.Delta == nil {
					http.Error(w, "Bad Request: delta is required for counter", http.StatusBadRequest)
					return
				}
				if err := s.Storage.AddCounter(metric.ID, *metric.Delta); err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			default:
				http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
				return
			}
		}
	}
	var updatedMetrics []models.Metric
	for _, metric := range metrics {
		var updatedMetric models.Metric
		updatedMetric.ID = metric.ID
		updatedMetric.MType = metric.MType

		switch metric.MType {
		case models.Gauge:
			value, _ := s.Storage.GetGauge(metric.ID)
			updatedMetric.Value = value
		case models.Counter:
			delta, _ := s.Storage.GetCounter(metric.ID)
			updatedMetric.Delta = delta
		}

		updatedMetrics = append(updatedMetrics, updatedMetric)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedMetrics); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func updateMetricInTx(tx *sql.Tx, metric models.Metric) error {
	switch metric.MType {
	case models.Gauge:

		if metric.Value == nil {
			return fmt.Errorf("value is required for gauge")
		}
		_, err := tx.Exec(`
            INSERT INTO gauges (name, value)
            VALUES ($1, $2)
            ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
        `, metric.ID, *metric.Value)
		return err
	case models.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("delta is required for counter")
		}
		_, err := tx.Exec(`
            INSERT INTO counters (name, value)
            VALUES ($1, $2)
            ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
        `, metric.ID, *metric.Delta)
		return err
	default:
		return fmt.Errorf("invalid metric type")
	}
}
