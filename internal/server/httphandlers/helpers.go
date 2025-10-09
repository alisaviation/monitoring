package httphandlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/models"
)

func (h *HTTPHandlers) updateJSONMetrics(w http.ResponseWriter, r *http.Request) {
	var metric models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.metricsService.UpdateMetric(r.Context(), metric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.respondWithMetric(w, r, metric)
}

func (h *HTTPHandlers) updateTextMetrics(w http.ResponseWriter, r *http.Request) {
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

	if err := h.metricsService.UpdateMetric(r.Context(), metric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.respondWithMetric(w, r, metric)
}

func (h *HTTPHandlers) getJSONValue(w http.ResponseWriter, r *http.Request) {
	var metric models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	foundMetric, err := h.metricsService.GetMetric(r.Context(), metric.MType, metric.ID)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	h.respondWithMetric(w, r, *foundMetric)
}

func (h *HTTPHandlers) getTextValue(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	metric, err := h.metricsService.GetMetric(r.Context(), metricType, metricName)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	var response string
	switch metric.MType {
	case models.Gauge:
		if metric.Value != nil {
			response = fmt.Sprint(*metric.Value)
		}
	case models.Counter:
		if metric.Delta != nil {
			response = fmt.Sprint(*metric.Delta)
		}
	}

	if response == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	h.setResponseHash(w, []byte(response))
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, response)
}

func (h *HTTPHandlers) respondWithMetric(w http.ResponseWriter, r *http.Request, metric models.Metric) {
	jsonData, err := json.Marshal(metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.setResponseHash(w, jsonData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *HTTPHandlers) setResponseHash(w http.ResponseWriter, data []byte) {
	if h.key == "" {
		return
	}
	hash := helpers.CalculateHash(data, h.key)
	w.Header().Set("HashSHA256", hash)
}
