package httphandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/models"
)

type benchmarkMetricsService struct{}

func (b *benchmarkMetricsService) UpdateMetric(ctx context.Context, metric models.Metric) error {
	return nil
}

func (b *benchmarkMetricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metric) error {
	return nil
}

func (b *benchmarkMetricsService) GetMetric(ctx context.Context, metricType, metricName string) (*models.Metric, error) {
	value := 25.5
	delta := int64(100)

	switch metricType {
	case "gauge":
		return &models.Metric{
			ID:    metricName,
			MType: metricType,
			Value: &value,
		}, nil
	case "counter":
		return &models.Metric{
			ID:    metricName,
			MType: metricType,
			Delta: &delta,
		}, nil
	default:
		return nil, fmt.Errorf("unknown metric type")
	}
}

func (b *benchmarkMetricsService) GetAllMetrics(ctx context.Context) ([]models.Metric, error) {
	value := 25.5
	delta := int64(100)
	return []models.Metric{
		{ID: "temperature", MType: "gauge", Value: &value},
		{ID: "requests", MType: "counter", Delta: &delta},
		{ID: "memory", MType: "gauge", Value: &value},
		{ID: "errors", MType: "counter", Delta: &delta},
	}, nil
}

func (b *benchmarkMetricsService) Ping(ctx context.Context) error {
	return nil
}

type BenchmarkHTTPHandlers struct {
	metricsService *benchmarkMetricsService
	key            string
}

func (h *BenchmarkHTTPHandlers) UpdateMetrics(w http.ResponseWriter, r *http.Request) {
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

func (h *BenchmarkHTTPHandlers) GetValue(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):
		h.getJSONValue(w, r)
	default:
		h.getTextValue(w, r)
	}
}

func (h *BenchmarkHTTPHandlers) UpdateBatchMetrics(w http.ResponseWriter, r *http.Request) {
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

func (h *BenchmarkHTTPHandlers) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.metricsService.Ping(r.Context()); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *BenchmarkHTTPHandlers) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.metricsService.GetAllMetrics(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var response strings.Builder
	response.WriteString("<html><body><h1>Metrics</h1><ul>")

	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				response.WriteString(fmt.Sprintf("<li>%s: %f</li>", metric.ID, *metric.Value))
			}
		case "counter":
			if metric.Delta != nil {
				response.WriteString(fmt.Sprintf("<li>%s: %d</li>", metric.ID, *metric.Delta))
			}
		}
	}

	response.WriteString("</ul></body></html>")
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(response.String()))
}

func (h *BenchmarkHTTPHandlers) updateJSONMetrics(w http.ResponseWriter, r *http.Request) {
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

func (h *BenchmarkHTTPHandlers) updateTextMetrics(w http.ResponseWriter, r *http.Request) {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		http.Error(w, "Bad Request: missing URL parameters", http.StatusBadRequest)
		return
	}

	metricType := rctx.URLParam("type")
	metricName := rctx.URLParam("name")
	valueStr := rctx.URLParam("value")

	metric := models.Metric{
		ID:    metricName,
		MType: metricType,
	}

	switch metric.MType {
	case "gauge":
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "Bad Request: invalid gauge value", http.StatusBadRequest)
			return
		}
		metric.Value = &value
	case "counter":
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

func (h *BenchmarkHTTPHandlers) getJSONValue(w http.ResponseWriter, r *http.Request) {
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

func (h *BenchmarkHTTPHandlers) getTextValue(w http.ResponseWriter, r *http.Request) {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		http.Error(w, "Bad Request: missing URL parameters", http.StatusBadRequest)
		return
	}

	metricType := rctx.URLParam("type")
	metricName := rctx.URLParam("name")

	metric, err := h.metricsService.GetMetric(r.Context(), metricType, metricName)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	var response string
	switch metric.MType {
	case "gauge":
		if metric.Value != nil {
			response = fmt.Sprint(*metric.Value)
		}
	case "counter":
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

func (h *BenchmarkHTTPHandlers) respondWithMetric(w http.ResponseWriter, r *http.Request, metric models.Metric) {
	jsonData, err := json.Marshal(metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.setResponseHash(w, jsonData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (h *BenchmarkHTTPHandlers) setResponseHash(w http.ResponseWriter, data []byte) {
	if h.key == "" {
		return
	}
	hash := fmt.Sprintf("hash-%s", h.key)
	w.Header().Set("HashSHA256", hash)
}

func createBenchmarkRequestWithChiParams(method, url string, params map[string]string, body []byte) *http.Request {
	r := httptest.NewRequest(method, url, bytes.NewReader(body))

	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}

	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r
}

func BenchmarkUpdateMetrics(b *testing.B) {
	handler := &BenchmarkHTTPHandlers{
		metricsService: &benchmarkMetricsService{},
		key:            "test-key",
	}

	benchmarks := []struct {
		name        string
		contentType string
		request     *http.Request
	}{
		{
			name:        "JSON",
			contentType: "application/json",
			request: func() *http.Request {
				value := 25.5
				metric := models.Metric{
					ID:    "temperature",
					MType: "gauge",
					Value: &value,
				}
				body, _ := json.Marshal(metric)
				req := httptest.NewRequest("POST", "/update", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
		},
		{
			name:        "TextGauge",
			contentType: "text/plain",
			request: createBenchmarkRequestWithChiParams("POST", "/update/gauge/temperature/25.5",
				map[string]string{"type": "gauge", "name": "temperature", "value": "25.5"}, nil),
		},
		{
			name:        "TextCounter",
			contentType: "text/plain",
			request: createBenchmarkRequestWithChiParams("POST", "/update/counter/requests/10",
				map[string]string{"type": "counter", "name": "requests", "value": "10"}, nil),
		},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				recorder := httptest.NewRecorder()
				handler.UpdateMetrics(recorder, bb.request)
				if recorder.Code != http.StatusOK {
					b.Fatalf("UpdateMetrics failed with status: %d", recorder.Code)
				}
			}
		})
	}
}

func BenchmarkGetValue(b *testing.B) {
	handler := &BenchmarkHTTPHandlers{
		metricsService: &benchmarkMetricsService{},
		key:            "test-key",
	}

	benchmarks := []struct {
		name        string
		contentType string
		request     *http.Request
	}{
		{
			name:        "JSON",
			contentType: "application/json",
			request: func() *http.Request {
				metric := models.Metric{ID: "temperature", MType: "gauge"}
				body, _ := json.Marshal(metric)
				req := httptest.NewRequest("POST", "/value", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
		},
		{
			name: "TextGauge",
			request: createBenchmarkRequestWithChiParams("GET", "/value/gauge/temperature",
				map[string]string{"type": "gauge", "name": "temperature"}, nil),
		},
		{
			name: "TextCounter",
			request: createBenchmarkRequestWithChiParams("GET", "/value/counter/requests",
				map[string]string{"type": "counter", "name": "requests"}, nil),
		},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				recorder := httptest.NewRecorder()
				handler.GetValue(recorder, bb.request)
				if recorder.Code != http.StatusOK {
					b.Fatalf("GetValue failed with status: %d", recorder.Code)
				}
			}
		})
	}
}

func BenchmarkUpdateBatchMetrics(b *testing.B) {
	handler := &BenchmarkHTTPHandlers{
		metricsService: &benchmarkMetricsService{},
		key:            "test-key",
	}

	benchmarks := []struct {
		name       string
		numMetrics int
	}{
		{"SingleMetric", 1},
		{"TenMetrics", 10},
		{"HundredMetrics", 100},
		{"ThousandMetrics", 1000},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			metrics := make([]models.Metric, bb.numMetrics)
			for i := 0; i < bb.numMetrics; i++ {
				if i%2 == 0 {
					value := 25.5 + float64(i)
					metrics[i] = models.Metric{
						ID:    fmt.Sprintf("metric_%d", i),
						MType: "gauge",
						Value: &value,
					}
				} else {
					delta := int64(100 + i)
					metrics[i] = models.Metric{
						ID:    fmt.Sprintf("metric_%d", i),
						MType: "counter",
						Delta: &delta,
					}
				}
			}

			body, _ := json.Marshal(metrics)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest("POST", "/updates", bytes.NewReader(body))
				recorder := httptest.NewRecorder()
				handler.UpdateBatchMetrics(recorder, req)
				if recorder.Code != http.StatusOK {
					b.Fatalf("UpdateBatchMetrics failed with status: %d", recorder.Code)
				}
			}
		})
	}
}

func BenchmarkPingHandler(b *testing.B) {
	handler := &BenchmarkHTTPHandlers{
		metricsService: &benchmarkMetricsService{},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/ping", nil)
		recorder := httptest.NewRecorder()
		handler.PingHandler(recorder, req)
		if recorder.Code != http.StatusOK {
			b.Fatalf("PingHandler failed with status: %d", recorder.Code)
		}
	}
}

func BenchmarkGetMetricsList(b *testing.B) {
	handler := &BenchmarkHTTPHandlers{
		metricsService: &benchmarkMetricsService{},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		recorder := httptest.NewRecorder()
		handler.GetMetricsList(recorder, req)
		if recorder.Code != http.StatusOK {
			b.Fatalf("GetMetricsList failed with status: %d", recorder.Code)
		}
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	handler := &BenchmarkHTTPHandlers{
		metricsService: &benchmarkMetricsService{},
		key:            "test-key",
	}

	b.Run("ConcurrentPing", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req := httptest.NewRequest("GET", "/ping", nil)
				recorder := httptest.NewRecorder()
				handler.PingHandler(recorder, req)

				if recorder.Code != http.StatusOK {
					b.Fatalf("PingHandler failed with status: %d", recorder.Code)
				}
			}
		})
	})

	b.Run("ConcurrentGetValue", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				var req *http.Request
				if counter%2 == 0 {
					req = createBenchmarkRequestWithChiParams("GET", "/value/gauge/temperature",
						map[string]string{"type": "gauge", "name": "temperature"}, nil)
				} else {
					req = createBenchmarkRequestWithChiParams("GET", "/value/counter/requests",
						map[string]string{"type": "counter", "name": "requests"}, nil)
				}

				recorder := httptest.NewRecorder()
				handler.GetValue(recorder, req)

				if recorder.Code != http.StatusOK {
					b.Fatalf("GetValue failed with status: %d", recorder.Code)
				}
				counter++
			}
		})
	})
}

func BenchmarkMemoryUsage(b *testing.B) {
	handler := &BenchmarkHTTPHandlers{
		metricsService: &benchmarkMetricsService{},
	}
	largeMetrics := make([]models.Metric, 1000)
	for i := 0; i < 1000; i++ {
		value := 25.5 + float64(i)
		largeMetrics[i] = models.Metric{
			ID:    fmt.Sprintf("metric_%d", i),
			MType: "gauge",
			Value: &value,
		}
	}

	b.Run("LargeBatchUpdate", func(b *testing.B) {
		body, _ := json.Marshal(largeMetrics)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/updates", bytes.NewReader(body))
			recorder := httptest.NewRecorder()
			handler.UpdateBatchMetrics(recorder, req)

			if recorder.Code != http.StatusOK {
				b.Fatalf("UpdateBatchMetrics failed with status: %d", recorder.Code)
			}
		}
	})
}

func BenchmarkContentTypeDetection(b *testing.B) {
	handler := &BenchmarkHTTPHandlers{
		metricsService: &benchmarkMetricsService{},
	}

	benchmarks := []struct {
		name        string
		contentType string
	}{
		{"JSON", "application/json"},
		{"Text", "text/plain"},
		{"Empty", ""},
		{"Multipart", "multipart/form-data"},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest("POST", "/update", nil)
				if bb.contentType != "" {
					req.Header.Set("Content-Type", bb.contentType)
				}

				recorder := httptest.NewRecorder()
				handler.UpdateMetrics(recorder, req)
			}
		})
	}
}
