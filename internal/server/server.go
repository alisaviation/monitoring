package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

type Server struct {
	MemStorage *storage.MemStorage
}

func (s *Server) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	var metrics models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch metrics.MType {
	case models.Gauge:
		if metrics.Value == nil {
			http.Error(w, "Bad Request: value is required for gauge", http.StatusBadRequest)
			return
		}
		s.MemStorage.SetGauge(metrics.ID, *metrics.Value)
		metrics.Value, _ = s.MemStorage.GetGauge(metrics.ID)

	case models.Counter:
		if metrics.Delta == nil {
			http.Error(w, "Bad Request: delta is required for counter", http.StatusBadRequest)
			return
		}
		s.MemStorage.AddCounter(metrics.ID, *metrics.Delta)
		metrics.Delta, _ = s.MemStorage.GetCounter(metrics.ID)

	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)

}

func (s *Server) GetValue(w http.ResponseWriter, r *http.Request) {
	var metrics models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch metrics.MType {
	case models.Gauge:
		value, exists := s.MemStorage.GetGauge(metrics.ID)
		if !exists {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		metrics.Value = value

	case models.Counter:
		value, exists := s.MemStorage.GetCounter(metrics.ID)
		if !exists {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		metrics.Delta = value

	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)

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
