package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/middleware"
	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/storage"
)

func Test_methodCheck(t *testing.T) {
	memStorage := storage.NewMemStorage("")
	handler := chi.NewRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %s", err)
	}
	defer db.Close()

	server := &Server{
		Storage: memStorage,
		DB:      db,
	}

	handler.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(server.UpdateMetrics))
	handler.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(server.GetValue))
	handler.Post("/update/", helpers.MethodCheck([]string{http.MethodPost})(server.UpdateMetrics))
	handler.Post("/value/", helpers.MethodCheck([]string{http.MethodPost})(server.GetValue))
	handler.Get("/value/", helpers.MethodCheck([]string{http.MethodGet})(server.GetValue))
	handler.Get("/", helpers.MethodCheck([]string{http.MethodGet})(server.GetMetricsList))
	handler.Get("/ping", helpers.MethodCheck([]string{http.MethodGet})(server.PingHandler))
	handler.Post("/updates/", helpers.MethodCheck([]string{http.MethodPost})(server.UpdateBatchMetrics))

	tests := []struct {
		name         string
		method       string
		url          string
		body         string
		expectedCode int
		expectedBody string
		setupMock    func()
	}{
		{
			"POST Update Gauge JSON",
			http.MethodPost,
			"/update/gauge/metric1/123.45",
			"",
			http.StatusOK,
			`{"id":"metric1","type":"gauge","value":123.45}`,
			func() {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO gauges").WithArgs("metric1", 123.45).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			"POST Update Counter JSON",
			http.MethodPost,
			"/update/counter/metric2/100",
			"",
			http.StatusOK,
			`{"id":"metric2","type":"counter","delta":100}`,
			func() {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO counters").WithArgs("metric2", int64(100)).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			"GET Value Gauge",
			http.MethodGet,
			"/value/gauge/metric1",
			"",
			http.StatusOK,
			"123.45",
			func() {
				mock.ExpectQuery("SELECT value FROM gauges WHERE name = ?").WithArgs("metric1").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(123.45))
			},
		},
		{
			"GET Value Counter",
			http.MethodGet,
			"/value/counter/metric2",
			"",
			http.StatusOK,
			"100",
			func() {
				mock.ExpectQuery("SELECT value FROM counters WHERE name = ?").WithArgs("metric2").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(int64(100)))
			},
		},
		{"GET Metrics List",
			http.MethodGet,
			"/",
			"",
			http.StatusOK,
			"",
			nil,
		},
		{
			"GET Value Empty",
			http.MethodGet,
			"/value/",
			"",
			http.StatusBadRequest,
			"",
			nil,
		},
		{
			"POST Value Empty",
			http.MethodPost,
			"/value/",
			"",
			http.StatusBadRequest,
			"",
			nil,
		},
		{
			"PUT Method Not Allowed",
			http.MethodPut,
			"/update/gauge/metric1/123.45",
			"",
			http.StatusMethodNotAllowed,
			"",
			nil,
		},
		{
			"DELETE Method Not Allowed",
			http.MethodDelete,
			"/update/gauge/metric1/123.45",
			"",
			http.StatusMethodNotAllowed,
			"",
			nil,
		},
		{
			"GET Ping Success",
			http.MethodGet,
			"/ping",
			"",
			http.StatusOK,
			"",
			nil,
		},
		{
			"POST Update Gauge Text",
			http.MethodPost,
			"/update/gauge/metric1/123.45",
			"", http.StatusOK,
			"",
			nil},
		{
			"POST Update Counter Text",
			http.MethodPost,
			"/update/counter/metric2/100",
			"",
			http.StatusOK,
			"",
			nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			var req *http.Request
			if tt.method == http.MethodPost && tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d for %s %s", tt.expectedCode, w.Code, tt.method, tt.url)
			}

			if tt.expectedBody != "" {
				body := strings.TrimSpace(w.Body.String())
				if body != tt.expectedBody {
					t.Errorf("Expected body %q, got %q for %s %s", tt.expectedBody, body, tt.method, tt.url)
				}
			}
		})
	}
}

func Test_updateMetrics(t *testing.T) {
	memStorage := storage.NewMemStorage("")
	handler := chi.NewRouter()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %s", err)
	}
	defer db.Close()
	server := &Server{
		Storage: memStorage,
		DB:      db,
	}

	handler.Post("/update/", server.UpdateMetrics)
	handler.Post("/update/{type}/{name}/{value}", server.UpdateMetrics)

	tests := []struct {
		name         string
		method       string
		url          string
		contentType  string
		body         string
		expectedCode int
		expectedBody string
		setupMock    func()
	}{
		{
			name:         "JSON Valid Gauge Update",
			method:       http.MethodPost,
			url:          "/update/",
			contentType:  "application/json",
			body:         `{"id": "metric1", "type": "gauge", "value": 123.45}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric1","type":"gauge","value":123.45}`,
			setupMock:    nil,
		},
		{
			name:         "JSON Valid Counter Update",
			method:       http.MethodPost,
			url:          "/update/",
			contentType:  "application/json",
			body:         `{"id": "metric2", "type": "counter", "delta": 100}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric2","type":"counter","delta":100}`,
			setupMock:    nil,
		},
		{
			name:         "JSON Invalid Metric Type",
			method:       http.MethodPost,
			url:          "/update/",
			contentType:  "application/json",
			body:         `{"id": "metric1", "type": "invalid", "value": 123.45}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "bad Request: invalid metric type",
			setupMock:    nil,
		},
		{
			name:         "JSON Invalid Gauge Value",
			method:       http.MethodPost,
			url:          "/update/",
			contentType:  "application/json",
			body:         `{"id": "metric1", "type": "gauge"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "bad Request: value is required for gauge",
			setupMock:    nil,
		},
		{
			name:         "JSON Invalid Counter Value",
			method:       http.MethodPost,
			url:          "/update/",
			contentType:  "application/json",
			body:         `{"id": "metric2", "type": "counter", "delta": null}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `bad Request: delta is required for counter`,
			setupMock:    nil,
		},
		{
			name:         "Text Valid Gauge Update",
			method:       http.MethodPost,
			url:          "/update/gauge/metric3/456.78",
			contentType:  "text/plain",
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric3","type":"gauge","value":456.78}`,
			setupMock:    nil,
		},
		{
			name:         "Text Valid Counter Update",
			method:       http.MethodPost,
			url:          "/update/counter/metric4/200",
			contentType:  "text/plain",
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric4","type":"counter","delta":200}`,
			setupMock:    nil,
		},
		{
			name:         "Text Invalid Gauge Value",
			method:       http.MethodPost,
			url:          "/update/gauge/metric5/invalid",
			contentType:  "text/plain",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid gauge value",
			setupMock:    nil,
		},
		{
			name:         "Text Invalid Counter Value",
			method:       http.MethodPost,
			url:          "/update/counter/metric6/invalid",
			contentType:  "text/plain",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid counter value",
			setupMock:    nil,
		},
		{
			name:         "Text Invalid Metric Type",
			method:       http.MethodPost,
			url:          "/update/invalid/metric7/123",
			contentType:  "text/plain",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid metric type",
			setupMock:    nil,
		},
		{
			name:         "Unsupported Content-Type",
			method:       http.MethodPost,
			url:          "/update/",
			contentType:  "application/xml",
			body:         "<metric><id>test</id><type>gauge</type><value>1.0</value></metric>",
			expectedCode: http.StatusUnsupportedMediaType,
			expectedBody: "Unsupported Content-Type",
			setupMock:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func Test_getValue(t *testing.T) {
	memStorage := storage.NewMemStorage("")
	memStorage.SetGauge(context.Background(), "metric1", 123.45)
	memStorage.AddCounter(context.Background(), "metric2", 100)
	server := NewServer(memStorage, nil)

	tests := []struct {
		name         string
		method       string
		url          string
		contentType  string
		body         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "JSON Valid Gauge Get",
			method:       http.MethodPost,
			url:          "/value/",
			contentType:  "application/json",
			body:         `{"id": "metric1", "type": "gauge"}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric1","type":"gauge","value":123.45}`,
		},
		{
			name:         "JSON Valid Counter Get",
			method:       http.MethodPost,
			url:          "/value/",
			contentType:  "application/json",
			body:         `{"id": "metric2", "type": "counter"}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"metric2","type":"counter","delta":100}`,
		},
		{
			name:         "JSON Invalid Metric Type",
			method:       http.MethodPost,
			url:          "/value/",
			contentType:  "application/json",
			body:         `{"id": "metric1", "type": "invalid"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `Bad Request: invalid metric type`,
		},
		{
			name:         "JSON Gauge Not Found",
			method:       http.MethodPost,
			url:          "/value/",
			contentType:  "application/json",
			body:         `{"id": "nonexistent", "type": "gauge"}`,
			expectedCode: http.StatusNotFound,
			expectedBody: "Not Found",
		},
		{
			name:         "Counter Not Found",
			method:       http.MethodPost,
			url:          "/value/",
			contentType:  "application/json",
			body:         `{"id": "nonexistent", "type": "counter"}`,
			expectedCode: http.StatusNotFound,
			expectedBody: "Not Found",
		},
		{
			name:         "Text Valid Gauge Get",
			method:       http.MethodGet,
			url:          "/value/gauge/metric1",
			contentType:  "text/plain",
			expectedCode: http.StatusOK,
			expectedBody: "123.45",
		},
		{
			name:         "Text Valid Counter Get",
			method:       http.MethodGet,
			url:          "/value/counter/metric2",
			contentType:  "text/plain",
			expectedCode: http.StatusOK,
			expectedBody: "100",
		},
		{
			name:         "Text Invalid Metric Type",
			method:       http.MethodGet,
			url:          "/value/invalid/metric1",
			contentType:  "text/plain",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid metric type",
		},
		{
			name:         "Text Gauge Not Found",
			method:       http.MethodGet,
			url:          "/value/gauge/nonexistent",
			contentType:  "text/plain",
			expectedCode: http.StatusNotFound,
			expectedBody: "Not Found",
		},
		{
			name:         "Text Counter Not Found",
			method:       http.MethodGet,
			url:          "/value/counter/nonexistent",
			contentType:  "text/plain",
			expectedCode: http.StatusNotFound,
			expectedBody: "Not Found",
		},
		{
			name:         "Unsupported Content-Type",
			method:       http.MethodPost,
			url:          "/value/",
			contentType:  "application/xml",
			body:         "<metric><id>metric1</id><type>gauge</type></metric>",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Bad Request: invalid metric type\n",
		},
	}
	handler := chi.NewRouter()
	handler.Post("/value/", server.GetValue)
	handler.Get("/value/{type}/{name}", server.GetValue)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func Test_gzipSupport(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "-a=localhost:8080", "-i=300", "-f=metrics.json", "-r=true"}

	memStorage := storage.NewMemStorage("")
	memStorage.SetGauge(context.Background(), "test_gauge", 123.45)
	memStorage.AddCounter(context.Background(), "test_counter", 42)
	srv := &Server{Storage: memStorage, DB: nil}

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
