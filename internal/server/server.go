package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

type Server struct {
	MemStorage *storage.MemStorage
}

func (s *Server) UpdateMetrics(w http.ResponseWriter, r *http.Request) {

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
		s.MemStorage.SetGauge(metricName, value)
	case models.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid counter value", http.StatusBadRequest)
			return
		}
		s.MemStorage.AddCounter(metricName, value)
	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Metrics updated")

}

func (s *Server) GetValue(w http.ResponseWriter, r *http.Request) {

	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	switch models.MetricType(metricType) {
	case models.Gauge:
		value, exists := s.MemStorage.GetGauge(metricName)
		if !exists {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		helpers.WriteResponse(w, http.StatusOK, value)
	case models.Counter:
		value, exists := s.MemStorage.GetCounter(metricName)
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

func GetMetricsList(memStorage *storage.MemStorage) http.HandlerFunc {
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
