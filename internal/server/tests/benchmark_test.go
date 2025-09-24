package tests

//
//import (
//	"bytes"
//	"context"
//	"encoding/json"
//	"net/http/httptest"
//	"testing"
//
//	"github.com/alisaviation/monitoring/internal/models"
//	server2 "github.com/alisaviation/monitoring/internal/server"
//	"github.com/alisaviation/monitoring/internal/storage"
//)
//
//func BenchmarkUpdateMetricsHandler(b *testing.B) {
//	storage := storage.NewMemStorage("")
//	server := server2.NewServer(storage, nil)
//
//	b.Run("Gauge", func(b *testing.B) {
//		for i := 0; i < b.N; i++ {
//			req := httptest.NewRequest("POST", "/update/gauge/test_metric/123.45", nil)
//			w := httptest.NewRecorder()
//			server.UpdateMetrics(w, req)
//		}
//	})
//
//	b.Run("Counter", func(b *testing.B) {
//		for i := 0; i < b.N; i++ {
//			req := httptest.NewRequest("POST", "/update/counter/test_metric/100", nil)
//			w := httptest.NewRecorder()
//			server.UpdateMetrics(w, req)
//		}
//	})
//}
//
//func BenchmarkGetValueHandler(b *testing.B) {
//	storage := storage.NewMemStorage("")
//	_ = storage.SetGauge(context.Background(), "test_metric", 123.45)
//	server := server2.NewServer(storage, nil)
//
//	b.Run("Gauge", func(b *testing.B) {
//		for i := 0; i < b.N; i++ {
//			req := httptest.NewRequest("GET", "/value/gauge/test_metric", nil)
//			w := httptest.NewRecorder()
//			server.GetValue(w, req)
//		}
//	})
//}
//
//func BenchmarkUpdateBatchMetricsHandler(b *testing.B) {
//	storage := storage.NewMemStorage("")
//	server := server2.NewServer(storage, nil)
//
//	metrics := []models.Metric{
//		{
//			ID:    "metric1",
//			MType: models.Gauge,
//			Value: func() *float64 { v := 1.23; return &v }(),
//		},
//		{
//			ID:    "metric2",
//			MType: models.Counter,
//			Delta: func() *int64 { d := int64(10); return &d }(),
//		},
//	}
//
//	jsonData, _ := json.Marshal(metrics)
//
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		req := httptest.NewRequest("POST", "/updates/", bytes.NewReader(jsonData))
//		req.Header.Set("Content-Type", "application/json")
//		w := httptest.NewRecorder()
//		server.UpdateBatchMetrics(w, req)
//	}
//}
//
//func BenchmarkServerHandlers(b *testing.B) {
//	stor := storage.NewMemStorage("")
//	srv := server2.NewServer(stor, nil)
//
//	b.Run("Ping", func(b *testing.B) {
//		req := httptest.NewRequest("GET", "/ping", nil)
//		for i := 0; i < b.N; i++ {
//			w := httptest.NewRecorder()
//			srv.PingHandler(w, req)
//		}
//	})
//
//	b.Run("GetMetricsList", func(b *testing.B) {
//		_ = stor.SetGauge(context.Background(), "gauge1", 1.23)
//		_ = stor.AddCounter(context.Background(), "counter1", 10)
//
//		req := httptest.NewRequest("GET", "/", nil)
//		for i := 0; i < b.N; i++ {
//			w := httptest.NewRecorder()
//			srv.GetMetricsList(w, req)
//		}
//	})
//}
