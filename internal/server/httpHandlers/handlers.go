// Package grpc_handlers implements http server handlers for the monitoring service.

package httpHandlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/service"
)

type HTTPHandlers struct {
	metricsService *service.MetricsService
	key            string
}

func NewHTTPHandlers(metricsService *service.MetricsService, key string) *HTTPHandlers {
	return &HTTPHandlers{
		metricsService: metricsService,
		key:            key,
	}
}

// UpdateMetrics handles metric updates in either JSON or text format.
// Supports both single metric updates and batch updates.
func (h *HTTPHandlers) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):
		h.updateJSONMetrics(w, r)
	case strings.Contains(contentType, "text/plain"), contentType == "":
		h.updateTextMetrics(w, r)
	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
	}
}

// GetValue retrieves a metric value in either JSON or text format.
func (h *HTTPHandlers) GetValue(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):
		h.getJSONValue(w, r)
	default:
		h.getTextValue(w, r)
	}
}

// UpdateBatchMetrics handles batch updates of multiple metrics in JSON format.
func (h *HTTPHandlers) UpdateBatchMetrics(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.metricsService.UpdateMetricsBatch(r.Context(), metrics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedMetrics, err := h.metricsService.GetAllMetrics(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(updatedMetrics)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.setResponseHash(w, jsonData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// PingHandler handles requests to check database connectivity.
// Responds with 200 OK if the database is reachable, 500 otherwise.
func (h *HTTPHandlers) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.metricsService.Ping(r.Context()); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetMetricsList returns an HTML page listing all stored metrics.
func (h *HTTPHandlers) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.metricsService.GetAllMetrics(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var response strings.Builder
	response.WriteString("<html><body><h1>Metrics</h1><ul>")

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value != nil {
				response.WriteString(fmt.Sprintf("<li>%s: %s</li>", metric.ID, helpers.FormatFloat(*metric.Value)))
			}
		case models.Counter:
			if metric.Delta != nil {
				response.WriteString(fmt.Sprintf("<li>%s: %d</li>", metric.ID, *metric.Delta))
			}
		}
	}

	response.WriteString("</ul></body></html>")
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(response.String()))
}
