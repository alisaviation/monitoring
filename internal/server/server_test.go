package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/alisaviation/monitoring/internal/middleware"
	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

func TestMethodCheck(t *testing.T) {
	memStorage := storage.NewMemStorage()
	handler := chi.NewRouter()
	server := &Server{MemStorage: memStorage}

	handler.Post("/update/", helpers.MethodCheck([]string{http.MethodPost})(server.UpdateMetrics))
	handler.Post("/value/", helpers.MethodCheck([]string{http.MethodPost})(server.GetValue))
	handler.Get("/", helpers.MethodCheck([]string{http.MethodGet})(server.GetMetricsList))

	tests := []struct {
		name         string
		method       string
		url          string
		body         string
		expectedCode int
	}{
		{"POST Update Gauge", http.MethodPost, "/update/", `{"id": "metric1", "type": "gauge", "value": 123.45}`, http.StatusOK},
		{"POST Update Counter", http.MethodPost, "/update/", `{"id": "metric2", "type": "counter", "delta": 100}`, http.StatusOK},
		{"POST Value Gauge", http.MethodPost, "/value/", `{"id": "metric1", "type": "gauge"}`, http.StatusOK},
		{"POST Value Counter", http.MethodPost, "/value/", `{"id": "metric2", "type": "counter"}`, http.StatusOK},
		{"GET Metrics List", http.MethodGet, "/", "", http.StatusOK},
		{"PUT Method Not Allowed", http.MethodPut, "/update/", "", http.StatusMethodNotAllowed},
		{"DELETE Method Not Allowed", http.MethodDelete, "/update/", "", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == http.MethodPost {
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
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
			url:          "/update/",
			body:         `{"id": "metric1", "type": "gauge", "value": 123.45}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric1","type":"gauge","value":123.45}`,
		},
		{
			name:         "Valid Counter Update",
			method:       http.MethodPost,
			url:          "/update/",
			body:         `{"id": "metric2", "type": "counter", "delta": 100}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric2","type":"counter","delta":100}`,
		},
		{
			name:         "Invalid Metric Type",
			method:       http.MethodPost,
			url:          "/update/",
			body:         `{"id": "metric1", "type": "invalid", "value": 123.45}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid metric type\n",
		},
		{
			name:         "Invalid Gauge Value",
			method:       http.MethodPost,
			url:          "/update/",
			body:         `{"id": "metric1", "type": "gauge", "value": null}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: value is required for gauge\n",
		},
		{
			name:         "Invalid Counter Value",
			method:       http.MethodPost,
			url:          "/update/",
			body:         `{"id": "metric2", "type": "counter", "delta": null}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: delta is required for counter\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memStorage := storage.NewMemStorage()
			handler := chi.NewRouter()
			server := &Server{MemStorage: memStorage}

			handler.Post("/update/", helpers.MethodCheck([]string{http.MethodPost})(server.UpdateMetrics))
			req := httptest.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
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
		body         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Valid Gauge Get",
			method:       http.MethodPost,
			url:          "/value/",
			body:         `{"id": "metric1", "type": "gauge"}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric1","type":"gauge","value":123.45}`,
		},
		{
			name:         "Valid Counter Get",
			method:       http.MethodPost,
			url:          "/value/",
			body:         `{"id": "metric2", "type": "counter"}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric2","type":"counter","delta":100}`,
		},
		{
			name:         "Invalid Metric Type",
			method:       http.MethodPost,
			url:          "/value/",
			body:         `{"id": "metric1", "type": "invalid"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid metric type\n",
		},
		{
			name:         "Gauge Not Found",
			method:       http.MethodPost,
			url:          "/value/",
			body:         `{"id": "nonexistent", "type": "gauge"}`,
			expectedCode: http.StatusNotFound,
			expectedBody: "Not Found\n",
		},
		{
			name:         "Counter Not Found",
			method:       http.MethodPost,
			url:          "/value/",
			body:         `{"id": "nonexistent", "type": "counter"}`,
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
			handler.Post("/value/", helpers.MethodCheck([]string{http.MethodPost})(server.GetValue))

			req := httptest.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
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

func TestGzipSupport(t *testing.T) {

	memStorage := storage.NewMemStorage()
	srv := &Server{MemStorage: memStorage}
	memStorage.SetGauge("test_gauge", 123.45)
	memStorage.AddCounter("test_counter", 42)

	testCases := []struct {
		name        string
		method      string
		path        string
		contentType string
		body        interface{}
	}{
		{
			name:        "Update Gauge",
			method:      "POST",
			path:        "/update/",
			contentType: "application/json",
			body: models.Metric{
				ID:    "new_gauge",
				MType: models.Gauge,
				Value: pointer(56.78),
			},
		},
		{
			name:        "Update Counter",
			method:      "POST",
			path:        "/update/",
			contentType: "application/json",
			body: models.Metric{
				ID:    "new_counter",
				MType: models.Counter,
				Delta: pointer(int64(10)),
			},
		},
		{
			name:        "Get Value (Gauge)",
			method:      "POST",
			path:        "/value/",
			contentType: "application/json",
			body: models.Metric{
				ID:    "test_gauge",
				MType: models.Gauge,
			},
		},
		{
			name:        "Get Value (Counter)",
			method:      "POST",
			path:        "/value/",
			contentType: "application/json",
			body: models.Metric{
				ID:    "test_counter",
				MType: models.Counter,
			},
		},
		{
			name:   "Get Metrics List",
			method: "GET",
			path:   "/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" with gzip request", func(t *testing.T) {
			testGzipRequest(t, srv, tc.method, tc.path, tc.contentType, tc.body)
		})

		t.Run(tc.name+" with gzip response", func(t *testing.T) {
			testGzipResponse(t, srv, tc.method, tc.path, tc.contentType, tc.body)
		})
	}
}

func testGzipRequest(t *testing.T, srv *Server, method, path, contentType string, body interface{}) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	var bodyData []byte
	if body != nil {
		var err error
		bodyData, err = json.Marshal(body)
		require.NoError(t, err)
	}

	_, err := gz.Write(bodyData)
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()

	handler := middleware.GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path {
		case "/update/":
			srv.UpdateMetrics(w, r)
		case "/value/":
			srv.GetValue(w, r)
		case "/":
			srv.GetMetricsList(w, r)
		}
	}))

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func testGzipResponse(t *testing.T, srv *Server, method, path, contentType string, body interface{}) {
	var bodyData []byte
	if body != nil {
		var err error
		bodyData, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(bodyData))
	req.Header.Set("Accept-Encoding", "gzip")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	w := httptest.NewRecorder()

	handler := middleware.GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path {
		case "/update/":
			srv.UpdateMetrics(w, r)
		case "/value/":
			srv.GetValue(w, r)
		case "/":
			srv.GetMetricsList(w, r)
		}
	}))

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	gz, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	defer gz.Close()

	_, err = io.ReadAll(gz)
	require.NoError(t, err)
}

func pointer[T any](v T) *T {
	return &v
}
