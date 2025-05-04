package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

func TestMethodCheck(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := chi.NewRouter()
	server := &Server{MemStorage: memStorage}

	handler.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(server.UpdateMetrics))
	handler.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(server.GetValue))
	handler.Get("/", GetMetricsList(memStorage))

	tests := []struct {
		name         string
		method       string
		url          string
		expectedCode int
	}{
		{"POST Update Gauge", http.MethodPost, "/update/gauge/metric1/123.45", http.StatusOK},
		{"POST Update Counter", http.MethodPost, "/update/counter/metric2/100", http.StatusOK},
		{"GET Value Gauge", http.MethodGet, "/value/gauge/metric1", http.StatusOK},
		{"GET Value Counter", http.MethodGet, "/value/counter/metric2", http.StatusOK},
		{"GET Metrics List", http.MethodGet, "/", http.StatusOK},
		{"PUT Method Not Allowed", http.MethodPut, "/update/gauge/metric1/123.45", http.StatusMethodNotAllowed},
		{"DELETE Method Not Allowed", http.MethodDelete, "/update/gauge/metric1/123.45", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			if tt.method == http.MethodPost {
				req.Header.Set("Content-Type", "text/plain")
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func Test_updateMetrics(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		url          string
		body         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Valid Gauge Update",
			method:       http.MethodPost,
			url:          "/update/gauge/metric1/123.45",
			body:         "",
			expectedCode: http.StatusOK,
			expectedBody: "Metrics updated",
		},
		{
			name:         "Valid Counter Update",
			method:       http.MethodPost,
			url:          "/update/counter/metric2/100",
			body:         "",
			expectedCode: http.StatusOK,
			expectedBody: "Metrics updated",
		},
		{
			name:         "Invalid Metric Type",
			method:       http.MethodPost,
			url:          "/update/invalid/metric1/123.45",
			body:         "",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid metric type\n",
		},
		{
			name:         "Invalid Gauge Value",
			method:       http.MethodPost,
			url:          "/update/gauge/metric1/invalid",
			body:         "",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid gauge value\n",
		},
		{
			name:         "Invalid Counter Value",
			method:       http.MethodPost,
			url:          "/update/counter/metric2/invalid",
			body:         "",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid counter value\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memStorage := storage.NewMemStorage()
			handler := chi.NewRouter()
			server := &Server{MemStorage: memStorage}

			handler.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(server.UpdateMetrics))
			req := httptest.NewRequest(tt.method, tt.url, nil)
			if tt.method == http.MethodPost {
				req.Header.Set("Content-Type", "text/plain")
				if tt.expectedCode == http.StatusUnsupportedMediaType {
					req.Header.Set("Content-Type", "application/json")
				}
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}
func Test_getValue(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		url          string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Valid Gauge Get",
			method:       http.MethodGet,
			url:          "/value/gauge/metric1",
			expectedCode: http.StatusOK,
			expectedBody: "123.45",
		},
		{
			name:         "Valid Counter Get",
			method:       http.MethodGet,
			url:          "/value/counter/metric2",
			expectedCode: http.StatusOK,
			expectedBody: "100",
		},
		{
			name:         "Invalid Metric Type",
			method:       http.MethodGet,
			url:          "/value/invalid/metric1",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid metric type\n",
		},
		{
			name:         "Gauge Not Found",
			method:       http.MethodGet,
			url:          "/value/gauge/nonexistent",
			expectedCode: http.StatusNotFound,
			expectedBody: "Not Found\n",
		},
		{
			name:         "Counter Not Found",
			method:       http.MethodGet,
			url:          "/value/counter/nonexistent",
			expectedCode: http.StatusNotFound,
			expectedBody: "Not Found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memStorage := storage.NewMemStorage()
			memStorage.SetGauge("metric1", 123.45)
			memStorage.AddCounter("metric2", 100)

			server := &Server{MemStorage: memStorage}
			handler := chi.NewRouter()
			handler.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(server.GetValue))

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}
