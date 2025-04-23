package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alisaviation/monitoring/internal/storage"
)

func TestMethodCheck(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		expectedCode int
	}{
		{"POST", http.MethodPost, http.StatusOK},
		{"GET", http.MethodGet, http.StatusMethodNotAllowed},
		{"PUT", http.MethodPut, http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/update/gauge/metric1/123.45", nil)
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			memStorage := storage.NewMemStorage()
			handler := methodCheck([]string{http.MethodPost})(updateMetrics(memStorage))
			handler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func TestUpdateMetrics(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		contentType  string
		expectedCode int
		expectedBody string
	}{
		{
			"Valid Gauge",
			"/update/gauge/metric1/123.45",
			"text/plain",
			http.StatusOK,
			"Metrics updated",
		},
		{
			"Valid Counter",
			"/update/counter/metric2/100",
			"text/plain",
			http.StatusOK,
			"Metrics updated",
		},
		{
			"Invalid Metric Type",
			"/update/invalid/metric3/100",
			"text/plain",
			http.StatusBadRequest,
			"Bad Request: invalid metric type\n",
		},
		{
			"Invalid Gauge Value",
			"/update/gauge/metric4/invalid",
			"text/plain",
			http.StatusBadRequest,
			"Bad Request: invalid gauge value\n",
		},
		{
			"Invalid Counter Value",
			"/update/counter/metric5/invalid",
			"text/plain",
			http.StatusBadRequest,
			"Bad Request: invalid counter value\n",
		},
		{
			"Missing Metric Name",
			"/update/gauge/",
			"text/plain",
			http.StatusNotFound,
			"Not Found\n",
		},
		{
			"Unsupported Media Type",
			"/update/gauge/metric6/123.45",
			"application/json",
			http.StatusUnsupportedMediaType,
			"Unsupported Media Type\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			memStorage := storage.NewMemStorage()
			handler := updateMetrics(memStorage)
			handler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}
