package httpHandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/service"
)

type mockMetricsService struct {
	updateMetricFunc       func(ctx context.Context, metric models.Metric) error
	updateMetricsBatchFunc func(ctx context.Context, metrics []models.Metric) error
	getMetricFunc          func(ctx context.Context, metricType, metricName string) (*models.Metric, error)
	getAllMetricsFunc      func(ctx context.Context) ([]models.Metric, error)
	pingFunc               func(ctx context.Context) error
}

func (m *mockMetricsService) UpdateMetric(ctx context.Context, metric models.Metric) error {
	if m.updateMetricFunc != nil {
		return m.updateMetricFunc(ctx, metric)
	}
	return nil
}

func (m *mockMetricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metric) error {
	if m.updateMetricsBatchFunc != nil {
		return m.updateMetricsBatchFunc(ctx, metrics)
	}
	return nil
}

func (m *mockMetricsService) GetMetric(ctx context.Context, metricType, metricName string) (*models.Metric, error) {
	if m.getMetricFunc != nil {
		return m.getMetricFunc(ctx, metricType, metricName)
	}
	return nil, nil
}

func (m *mockMetricsService) GetAllMetrics(ctx context.Context) ([]models.Metric, error) {
	if m.getAllMetricsFunc != nil {
		return m.getAllMetricsFunc(ctx)
	}
	return nil, nil
}

func (m *mockMetricsService) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

type TestHTTPHandlers struct {
	metricsService *mockMetricsService
	key            string
}

func (h *TestHTTPHandlers) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):
		h.updateJSONMetrics(w, r)
	case strings.Contains(contentType, "text/plain"), contentType == "":
		h.updateTextMetrics(w, r)
	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
	}
}

func (h *TestHTTPHandlers) GetValue(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):
		h.getJSONValue(w, r)
	default:
		h.getTextValue(w, r)
	}
}

func (h *TestHTTPHandlers) UpdateBatchMetrics(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.metricsService.UpdateMetricsBatch(r.Context(), metrics); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedMetrics, err := h.metricsService.GetAllMetrics(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(updatedMetrics)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.setResponseHash(w, jsonData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *TestHTTPHandlers) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.metricsService.Ping(r.Context()); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TestHTTPHandlers) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.metricsService.GetAllMetrics(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var response strings.Builder
	response.WriteString("<html><body><h1>Metrics</h1><ul>")

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value != nil {
				response.WriteString(fmt.Sprintf("<li>%s: %f</li>", metric.ID, *metric.Value))
			}
		case models.Counter:
			if metric.Delta != nil {
				response.WriteString(fmt.Sprintf("<li>%s: %d</li>", metric.ID, *metric.Delta))
			}
		}
	}

	response.WriteString("</ul></body></html>")
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(response.String()))
}

func (h *TestHTTPHandlers) updateJSONMetrics(w http.ResponseWriter, r *http.Request) {
	var metric models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.metricsService.UpdateMetric(r.Context(), metric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.respondWithMetric(w, r, metric)
}

func (h *TestHTTPHandlers) getJSONValue(w http.ResponseWriter, r *http.Request) {
	var metric models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	foundMetric, err := h.metricsService.GetMetric(r.Context(), metric.MType, metric.ID)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	h.respondWithMetric(w, r, *foundMetric)
}

func (h *TestHTTPHandlers) respondWithMetric(w http.ResponseWriter, r *http.Request, metric models.Metric) {
	jsonData, err := json.Marshal(metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.setResponseHash(w, jsonData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *TestHTTPHandlers) setResponseHash(w http.ResponseWriter, data []byte) {
	if h.key == "" {
		return
	}
	hash := fmt.Sprintf("hash-%s", h.key)
	w.Header().Set("HashSHA256", hash)
}

func TestHTTPHandlers_GetMetricsList(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
		key            string
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "successful get metrics list",
			fields: fields{
				key: "",
			},
			args: args{
				r: httptest.NewRequest("GET", "/", nil),
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "<html>",
		},
		{
			name: "get metrics list with service error",
			fields: fields{
				key: "",
			},
			args: args{
				r: httptest.NewRequest("GET", "/", nil),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockMetricsService{}

			if tt.name == "successful get metrics list" {
				mockService.getAllMetricsFunc = func(ctx context.Context) ([]models.Metric, error) {
					value := 25.5
					delta := int64(100)
					return []models.Metric{
						{ID: "temperature", MType: "gauge", Value: &value},
						{ID: "requests", MType: "counter", Delta: &delta},
					}, nil
				}
			} else if tt.name == "get metrics list with service error" {
				mockService.getAllMetricsFunc = func(ctx context.Context) ([]models.Metric, error) {
					return nil, errors.New("service error")
				}
			}

			h := &TestHTTPHandlers{
				metricsService: mockService,
				key:            tt.fields.key,
			}

			recorder := httptest.NewRecorder()
			h.GetMetricsList(recorder, tt.args.r)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("GetMetricsList() status code = %v, want %v", recorder.Code, tt.wantStatusCode)
			}

			if tt.wantBody != "" && !strings.Contains(recorder.Body.String(), tt.wantBody) {
				t.Errorf("GetMetricsList() body = %v, should contain %v", recorder.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHTTPHandlers_PingHandler(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
		key            string
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "ping successful",
			fields: fields{
				key: "",
			},
			args: args{
				r: httptest.NewRequest("GET", "/ping", nil),
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "ping with database error",
			fields: fields{
				key: "",
			},
			args: args{
				r: httptest.NewRequest("GET", "/ping", nil),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockMetricsService{}

			if tt.name == "ping successful" {
				mockService.pingFunc = func(ctx context.Context) error {
					return nil
				}
			} else if tt.name == "ping with database error" {
				mockService.pingFunc = func(ctx context.Context) error {
					return errors.New("database error")
				}
			}

			h := &TestHTTPHandlers{
				metricsService: mockService,
				key:            tt.fields.key,
			}

			recorder := httptest.NewRecorder()
			h.PingHandler(recorder, tt.args.r)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("PingHandler() status code = %v, want %v", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestHTTPHandlers_UpdateBatchMetrics(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
		key            string
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "successful batch update",
			fields: fields{
				key: "",
			},
			args: args{
				r: func() *http.Request {
					metrics := []models.Metric{
						{ID: "temp", MType: "gauge"},
						{ID: "req", MType: "counter"},
					}
					body, _ := json.Marshal(metrics)
					return httptest.NewRequest("POST", "/updates", bytes.NewReader(body))
				}(),
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "batch update with invalid JSON",
			fields: fields{
				key: "",
			},
			args: args{
				r: httptest.NewRequest("POST", "/updates", bytes.NewReader([]byte("invalid json"))),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "batch update with service error",
			fields: fields{
				key: "",
			},
			args: args{
				r: func() *http.Request {
					metrics := []models.Metric{{ID: "temp", MType: "gauge"}}
					body, _ := json.Marshal(metrics)
					return httptest.NewRequest("POST", "/updates", bytes.NewReader(body))
				}(),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockMetricsService{}

			switch tt.name {
			case "successful batch update":
				mockService.updateMetricsBatchFunc = func(ctx context.Context, metrics []models.Metric) error {
					return nil
				}
				mockService.getAllMetricsFunc = func(ctx context.Context) ([]models.Metric, error) {
					return []models.Metric{}, nil
				}
			case "batch update with service error":
				mockService.updateMetricsBatchFunc = func(ctx context.Context, metrics []models.Metric) error {
					return errors.New("service error")
				}
			}

			h := &TestHTTPHandlers{
				metricsService: mockService,
				key:            tt.fields.key,
			}

			recorder := httptest.NewRecorder()
			h.UpdateBatchMetrics(recorder, tt.args.r)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("UpdateBatchMetrics() status code = %v, want %v", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func createRequestWithChiParams(method, url string, params map[string]string, body []byte) *http.Request {
	r := httptest.NewRequest(method, url, bytes.NewReader(body))

	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}

	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r
}

func (h *TestHTTPHandlers) updateTextMetrics(w http.ResponseWriter, r *http.Request) {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		http.Error(w, "Bad Request: missing URL parameters", http.StatusBadRequest)
		return
	}

	metricType := rctx.URLParam("type")
	metricName := rctx.URLParam("name")
	valueStr := rctx.URLParam("value")

	if metricType == "" || metricName == "" || valueStr == "" {
		http.Error(w, "Bad Request: missing parameters", http.StatusBadRequest)
		return
	}

	metric := models.Metric{
		ID:    metricName,
		MType: metricType,
	}

	switch metric.MType {
	case models.Gauge:
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid gauge value", http.StatusBadRequest)
			return
		}
		metric.Value = &value
	case models.Counter:
		delta, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid counter value", http.StatusBadRequest)
			return
		}
		metric.Delta = &delta
	default:
		http.Error(w, "Bad Request: invalid metric type", http.StatusBadRequest)
		return
	}

	if err := h.metricsService.UpdateMetric(r.Context(), metric); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.respondWithMetric(w, r, metric)
}

func (h *TestHTTPHandlers) getTextValue(w http.ResponseWriter, r *http.Request) {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		http.Error(w, "Bad Request: missing URL parameters", http.StatusBadRequest)
		return
	}

	metricType := rctx.URLParam("type")
	metricName := rctx.URLParam("name")

	if metricType == "" || metricName == "" {
		http.Error(w, "Bad Request: missing parameters", http.StatusBadRequest)
		return
	}

	metric, err := h.metricsService.GetMetric(r.Context(), metricType, metricName)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	var response string
	switch metric.MType {
	case models.Gauge:
		if metric.Value != nil {
			response = fmt.Sprint(*metric.Value)
		}
	case models.Counter:
		if metric.Delta != nil {
			response = fmt.Sprint(*metric.Delta)
		}
	}

	if response == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	h.setResponseHash(w, []byte(response))
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, response)
}

func TestHTTPHandlers_GetValue(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
		key            string
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		contentType    string
	}{
		{
			name: "get JSON value successfully",
			fields: fields{
				key: "",
			},
			args: args{
				r: func() *http.Request {
					metric := models.Metric{ID: "temperature", MType: "gauge"}
					body, _ := json.Marshal(metric)
					req := httptest.NewRequest("POST", "/value", bytes.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					return req
				}(),
			},
			wantStatusCode: http.StatusOK,
			contentType:    "application/json",
		},
		{
			name: "get text value successfully",
			fields: fields{
				key: "",
			},
			args: args{
				r: createRequestWithChiParams("GET", "/value/gauge/temperature",
					map[string]string{"type": "gauge", "name": "temperature"}, nil),
			},
			wantStatusCode: http.StatusOK,
			contentType:    "text/plain",
		},
		{
			name: "get value not found",
			fields: fields{
				key: "",
			},
			args: args{
				r: func() *http.Request {
					metric := models.Metric{ID: "nonexistent", MType: "gauge"}
					body, _ := json.Marshal(metric)
					req := httptest.NewRequest("POST", "/value", bytes.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					return req
				}(),
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "get text value with missing parameters",
			fields: fields{
				key: "",
			},
			args: args{
				r: createRequestWithChiParams("GET", "/value/", map[string]string{}, nil),
			},
			wantStatusCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockMetricsService{}

			switch tt.name {
			case "get JSON value successfully", "get text value successfully":
				mockService.getMetricFunc = func(ctx context.Context, metricType, metricName string) (*models.Metric, error) {
					value := 25.5
					return &models.Metric{
						ID:    metricName,
						MType: metricType,
						Value: &value,
					}, nil
				}
			case "get value not found":
				mockService.getMetricFunc = func(ctx context.Context, metricType, metricName string) (*models.Metric, error) {
					return nil, errors.New("not found")
				}
			}

			h := &TestHTTPHandlers{
				metricsService: mockService,
				key:            tt.fields.key,
			}

			recorder := httptest.NewRecorder()
			h.GetValue(recorder, tt.args.r)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("GetValue() status code = %v, want %v", recorder.Code, tt.wantStatusCode)
			}

			if tt.contentType != "" {
				actualContentType := recorder.Header().Get("Content-Type")
				if !strings.Contains(actualContentType, tt.contentType) {
					t.Errorf("GetValue() content type = %v, should contain %v", actualContentType, tt.contentType)
				}
			}
		})
	}
}

func TestHTTPHandlers_UpdateMetrics(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
		key            string
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		contentType    string
	}{
		{
			name: "update JSON metrics successfully",
			fields: fields{
				key: "",
			},
			args: args{
				r: func() *http.Request {
					metric := models.Metric{ID: "temperature", MType: "gauge"}
					body, _ := json.Marshal(metric)
					req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					return req
				}(),
			},
			wantStatusCode: http.StatusOK,
			contentType:    "application/json",
		},
		{
			name: "update text metrics successfully - gauge",
			fields: fields{
				key: "",
			},
			args: args{
				r: createRequestWithChiParams("POST", "/update/gauge/temperature/25.5",
					map[string]string{"type": "gauge", "name": "temperature", "value": "25.5"}, nil),
			},
			wantStatusCode: http.StatusOK,
			contentType:    "application/json",
		},
		{
			name: "update text metrics successfully - counter",
			fields: fields{
				key: "",
			},
			args: args{
				r: createRequestWithChiParams("POST", "/update/counter/requests/10",
					map[string]string{"type": "counter", "name": "requests", "value": "10"}, nil),
			},
			wantStatusCode: http.StatusOK,
			contentType:    "application/json",
		},
		{
			name: "update text metrics with invalid gauge value",
			fields: fields{
				key: "",
			},
			args: args{
				r: createRequestWithChiParams("POST", "/update/gauge/temperature/invalid",
					map[string]string{"type": "gauge", "name": "temperature", "value": "invalid"}, nil),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "update text metrics with invalid counter value",
			fields: fields{
				key: "",
			},
			args: args{
				r: createRequestWithChiParams("POST", "/update/counter/requests/invalid",
					map[string]string{"type": "counter", "name": "requests", "value": "invalid"}, nil),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "update text metrics with invalid metric type",
			fields: fields{
				key: "",
			},
			args: args{
				r: createRequestWithChiParams("POST", "/update/invalid/temperature/25.5",
					map[string]string{"type": "invalid", "name": "temperature", "value": "25.5"}, nil),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "update with unsupported content type",
			fields: fields{
				key: "",
			},
			args: args{
				r: func() *http.Request {
					req := httptest.NewRequest("POST", "/update", nil)
					req.Header.Set("Content-Type", "application/xml")
					return req
				}(),
			},
			wantStatusCode: http.StatusUnsupportedMediaType,
		},
		{
			name: "update with service error",
			fields: fields{
				key: "",
			},
			args: args{
				r: func() *http.Request {
					metric := models.Metric{ID: "temperature", MType: "gauge"}
					body, _ := json.Marshal(metric)
					req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					return req
				}(),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockMetricsService{}

			switch {
			case strings.Contains(tt.name, "successfully"):
				mockService.updateMetricFunc = func(ctx context.Context, metric models.Metric) error {
					return nil
				}
			case tt.name == "update with service error":
				mockService.updateMetricFunc = func(ctx context.Context, metric models.Metric) error {
					return errors.New("service error")
				}
			}

			h := &TestHTTPHandlers{
				metricsService: mockService,
				key:            tt.fields.key,
			}

			recorder := httptest.NewRecorder()
			h.UpdateMetrics(recorder, tt.args.r)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf("UpdateMetrics() status code = %v, want %v", recorder.Code, tt.wantStatusCode)
			}

			if tt.contentType != "" {
				actualContentType := recorder.Header().Get("Content-Type")
				if !strings.Contains(actualContentType, tt.contentType) {
					t.Errorf("UpdateMetrics() content type = %v, should contain %v", actualContentType, tt.contentType)
				}
			}
		})
	}
}
