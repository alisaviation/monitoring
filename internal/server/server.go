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

func (p *Server) PingHandler(w http.ResponseWriter, r *http.Request) {

	if err := p.DB.PingContext(r.Context()); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (p *Server) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		p.UpdateJSONMetrics(r.Context(), w, r)
	case strings.Contains(contentType, "text/plain"), contentType == "":
		p.UpdateTextMetrics(w, r)
	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
	}
}

func (p *Server) UpdateJSONMetrics(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var metrics models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}
	if err := validateMetric(metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := p.updateMetric(ctx, metrics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.respondWithMetric(ctx, w, metrics, p.getKeyFromContext(r.Context()))
}

func (p *Server) UpdateTextMetrics(w http.ResponseWriter, r *http.Request) {
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

	if err := p.updateMetric(r.Context(), metric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.respondWithMetric(r.Context(), w, metric, p.getKeyFromContext(r.Context()))
}

func (p *Server) GetValue(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	var response interface{}

	switch {
	case strings.Contains(contentType, "application/json"):
		metrics := p.GetJSONValue(r.Context(), w, r)
		if metrics.ID == "" {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		jsonData, err := json.Marshal(metrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if r.Method == http.MethodPost {
			p.setResponseHash(w, jsonData, p.getKeyFromContext(r.Context()))
		}
		if r.Method == http.MethodGet {
			key := p.getKeyFromContext(r.Context())
			p.setResponseHash(w, jsonData, key)
		}
		w.Write(jsonData)
		return
	default:
		metrics := p.GetTextValue(r.Context(), w, r)
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
	if r.Method == http.MethodPost {
		responseStr := fmt.Sprint(response)
		p.setResponseHash(w, []byte(responseStr), p.getKeyFromContext(r.Context()))
	}
	if r.Method == http.MethodGet {
		key := p.getKeyFromContext(r.Context())
		p.setResponseHash(w, []byte(fmt.Sprint(response)), key)
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, response)
}

func (p *Server) GetJSONValue(ctx context.Context, w http.ResponseWriter, r *http.Request) models.Metric {
	var metrics models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return models.Metric{}
	}

	switch metrics.MType {
	case models.Gauge:
		value, err := p.Storage.GetGauge(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found  gauge in GetJSONValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, err := p.Storage.GetCounter(ctx, metrics.ID)
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

func (p *Server) GetTextValue(ctx context.Context, w http.ResponseWriter, r *http.Request) models.Metric {
	var metrics models.Metric

	metrics.ID = chi.URLParam(r, "name")
	metrics.MType = chi.URLParam(r, "type")

	switch metrics.MType {
	case models.Gauge:
		value, err := p.Storage.GetGauge(ctx, metrics.ID)
		if err != nil {
			http.Error(w, "Not Found gauge value in GetTextValue", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, err := p.Storage.GetCounter(ctx, metrics.ID)
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

func (p *Server) UpdateBatchMetrics(w http.ResponseWriter, r *http.Request) {
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

	if p.DB != nil {
		if err := p.execInTransactionWithRetry(r.Context(), func(tx *sql.Tx) error {
			for _, metric := range metrics {
				if err := updateMetricInTx(tx, metric); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			if p.Storage.IsUniqueViolationError(err) {
				http.Error(w, "Conflict: unique violation", http.StatusConflict)
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
	}
	if p.DB == nil {
		for _, metric := range metrics {
			if err := p.updateMetric(r.Context(), metric); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	updatedMetrics, err := p.getUpdatedMetrics(r.Context(), metrics)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(updatedMetrics)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	p.setResponseHash(w, jsonData, p.getKeyFromContext(r.Context()))
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (p *Server) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	var response strings.Builder

	response.WriteString("<html><body><h1>Metrics</h1><ul>")

	gauges, err := p.Storage.Gauges(r.Context())
	if err == nil {
		for name, value := range gauges {
			response.WriteString(fmt.Sprintf("<li>%s: %s</li>", name, helpers.FormatFloat(value)))
		}
	}

	counters, err := p.Storage.Counters(r.Context())
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
