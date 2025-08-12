package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/middleware"
	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/storage"
)

type Server struct {
	storage storage.Storage
	db      *sql.DB
}

func NewServer(storage storage.Storage, db *sql.DB) *Server {
	return &Server{
		storage: storage,
		db:      db,
	}
}

func (s *Server) PingHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	if err := s.db.PingContext(r.Context()); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	key := middleware.GetKeyFromContext(r.Context())
	switch {
	case strings.Contains(contentType, "application/json"):
		s.updateJSONMetrics(r.Context(), w, r, key)
	case strings.Contains(contentType, "text/plain"), contentType == "":
		s.updateTextMetrics(w, r, key)
	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
	}
}

func (s *Server) GetValue(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	var response interface{}
	key := middleware.GetKeyFromContext(r.Context())

	switch {
	case strings.Contains(contentType, "application/json"):
		metrics := s.getJSONValue(r.Context(), w, r)
		if metrics.ID == "" {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		jsonData, err := json.Marshal(metrics)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.setResponseHash(w, jsonData, key)
		w.Write(jsonData)

		return
	default:
		metrics := s.getTextValue(r.Context(), w, r)
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
	s.setResponseHash(w, []byte(fmt.Sprint(response)), key)

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, response)
}

func (s *Server) UpdateBatchMetrics(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metric
	key := middleware.GetKeyFromContext(r.Context())

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
			http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
			return
		}
	}

	if s.db != nil {
		if err := s.execInTransactionWithRetry(r.Context(), func(tx *sql.Tx) error {
			for _, metric := range metrics {
				if err := updateMetricInTx(tx, metric); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			if s.storage.IsUniqueViolationError(err) {
				http.Error(w, "Conflict: unique violation", http.StatusConflict)
			} else {
				http.Error(w, "Error Internal Server Error (IsUniqueViolation)", http.StatusInternalServerError)
			}
			return
		}
	}

	if s.db == nil {
		for _, metric := range metrics {
			if err := s.updateMetric(r.Context(), metric); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	updatedMetrics, err := s.getUpdatedMetrics(r.Context(), metrics)
	if err != nil {
		http.Error(w, "Internal Server Error (getUpdatedMetrics )", http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(updatedMetrics)
	if err != nil {
		http.Error(w, "Internal Server Error (getUpdatedMetrics )", http.StatusInternalServerError)
		return
	}

	s.setResponseHash(w, jsonData, key)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (s *Server) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	var response strings.Builder

	response.WriteString("<html><body><h1>Metrics</h1><ul>")

	gauges, err := s.storage.Gauges(r.Context())
	if err == nil {
		for name, value := range gauges {
			response.WriteString(fmt.Sprintf("<li>%s: %s</li>", name, helpers.FormatFloat(value)))
		}
	}

	counters, err := s.storage.Counters(r.Context())
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
