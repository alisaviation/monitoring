package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/cmd/server/helpers"
	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/storage"
)

func updateMetrics(memStorage *storage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//if r.Header.Get("Content-Type") != "text/plain" {
		//	http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		//	return
		//}

		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		metricValue := chi.URLParam(r, "value")

		switch models.MetricType(metricType) {
		case models.Gauge:
			value, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(w, "Bad Request: invalid gauge value", http.StatusBadRequest)
				return
			}
			memStorage.SetGauge(metricName, value)
		case models.Counter:
			value, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(w, "Bad Request: invalid counter value", http.StatusBadRequest)
				return
			}
			memStorage.AddCounter(metricName, value)
		default:
			http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Metrics updated")
	}
}

func getValue(memStorage *storage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")

		switch models.MetricType(metricType) {
		case models.Gauge:
			value, exists := memStorage.GetGauge(metricName)
			if !exists {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			helpers.WriteResponse(w, http.StatusOK, value)
		case models.Counter:
			value, exists := memStorage.GetCounter(metricName)
			if !exists {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			helpers.WriteResponse(w, http.StatusOK, value)
		default:
			http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
			return
		}
	}
}

func getMetricsList(memStorage *storage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		var response strings.Builder
		response.WriteString("<html><body><h1>Metrics</h1><ul>")
		for name, value := range memStorage.Gauges() {
			response.WriteString(fmt.Sprintf("<li>%s: %s</li>", name, helpers.FormatFloat(value)))
		}
		for name, value := range memStorage.Counters() {
			response.WriteString(fmt.Sprintf("<li>%s: %d</li>", name, value))
		}
		response.WriteString("</ul></body></html>")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response.String()))
	}
}
