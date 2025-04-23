package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/storage"
)

func methodCheck(methods []string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			for _, method := range methods {
				if r.Method == method {
					next(w, r)
					return
				}
			}
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func updateMetrics(memStorage *storage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
			return
		}

		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/update/"), "/")
		if len(parts) != 3 {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		metricType := models.MetricType(parts[0])
		metricName := parts[1]
		metricValue := parts[2]

		switch metricType {
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
