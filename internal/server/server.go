package server

import (
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
	psqlStorage storage.Storage
	DB          *sql.DB
	MemStorage  *storage.MemStorage
}

func NewServer(memStorage *storage.MemStorage, db *sql.DB) *Server {
	var s storage.Storage = memStorage

	if db != nil {
		pgStorage, err := storage.NewPostgresStorageFromDB(db)
		if err == nil {
			s = pgStorage
		}
	}

	return &Server{
		psqlStorage: s,
		DB:          db,
		MemStorage:  memStorage,
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
		if err := s.psqlStorage.SetGauge(metrics.ID, *metrics.Value); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		metrics.Value, _ = s.psqlStorage.GetGauge(metrics.ID)

	case models.Counter:
		if metrics.Delta == nil {
			http.Error(w, "Bad Request: delta is required for counter", http.StatusBadRequest)
			return
		}
		if err := s.psqlStorage.AddCounter(metrics.ID, *metrics.Delta); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		metrics.Delta, _ = s.psqlStorage.GetCounter(metrics.ID)

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
	valueStr := chi.URLParam(r, "value")

	var metrics models.Metric
	metrics.ID = chi.URLParam(r, "name")
	metrics.MType = chi.URLParam(r, "type")

	switch metrics.MType {
	case models.Gauge:
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid gauge value", http.StatusBadRequest)
			return
		}
		if err := s.psqlStorage.SetGauge(metrics.ID, value); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		metrics.Value, _ = s.psqlStorage.GetGauge(metrics.ID)
		//metrics.Value = &value
		//s.MemStorage.SetGauge(metrics.ID, value)

	case models.Counter:
		delta, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid counter value", http.StatusBadRequest)
			return
		}
		if err := s.psqlStorage.AddCounter(metrics.ID, delta); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		metrics.Delta, _ = s.psqlStorage.GetCounter(metrics.ID)
		//metrics.Delta = &delta
		//s.MemStorage.AddCounter(metrics.ID, delta)

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
		http.Error(w, "Not Found", http.StatusNotFound)
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
		value, exists := s.psqlStorage.GetGauge(metrics.ID)
		if !exists {
			http.Error(w, "Not Found", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, exists := s.psqlStorage.GetCounter(metrics.ID)
		if !exists {
			http.Error(w, "Not Found", http.StatusNotFound)
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
		value, exists := s.psqlStorage.GetGauge(metrics.ID)
		if !exists {
			http.Error(w, "Not Found", http.StatusNotFound)
			return models.Metric{}
		}
		metrics.Value = value
	case models.Counter:
		delta, exists := s.psqlStorage.GetCounter(metrics.ID)
		if !exists {
			http.Error(w, "Not Found", http.StatusNotFound)
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

	gauges, err := s.psqlStorage.Gauges()
	if err == nil {
		for name, value := range gauges {
			response.WriteString(fmt.Sprintf("<li>%s: %s</li>", name, helpers.FormatFloat(value)))
		}
	}
	//for name, value := range s.MemStorage.Gauges() {
	//	response.WriteString(fmt.Sprintf("<li>%s: %s</li>", name, helpers.FormatFloat(value)))
	//}
	counters, err := s.psqlStorage.Counters()
	if err == nil {
		for name, value := range counters {
			response.WriteString(fmt.Sprintf("<li>%s: %d</li>", name, value))
		}
	}

	//for name, value := range s.MemStorage.Counters() {
	//	response.WriteString(fmt.Sprintf("<li>%s: %d</li>", name, value))
	//}

	response.WriteString("</ul></body></html>")

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response.String()))

}
