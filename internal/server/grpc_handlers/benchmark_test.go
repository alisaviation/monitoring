package grpc_handlers

import (
	"context"
	"testing"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/proto/brief/rpc"
)

type benchmarkMetricsService struct{}

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
		return nil, nil
	}
}

func (b *benchmarkMetricsService) Ping(ctx context.Context) error {
	return nil
}

type BenchmarkGRPCHandlers struct {
	rpc.UnimplementedMonitoringServiceServer
	metricsService *benchmarkMetricsService
}

func (h *BenchmarkGRPCHandlers) protoToModel(protoMetric *rpc.Metric) models.Metric {
	metric := models.Metric{
		ID:    protoMetric.Id,
		MType: protoMetric.Mtype,
		Hash:  protoMetric.Hash,
	}

	if protoMetric.Delta != 0 || protoMetric.Mtype == models.Counter {
		delta := protoMetric.Delta
		metric.Delta = &delta
	}

	if protoMetric.Value != 0 || protoMetric.Mtype == models.Gauge {
		value := protoMetric.Value
		metric.Value = &value
	}

	return metric
}

func (h *BenchmarkGRPCHandlers) modelToProto(metric models.Metric) *rpc.Metric {
	protoMetric := &rpc.Metric{
		Id:    metric.ID,
		Mtype: metric.MType,
		Hash:  metric.Hash,
	}

	if metric.Delta != nil {
		protoMetric.Delta = *metric.Delta
	}

	if metric.Value != nil {
		protoMetric.Value = *metric.Value
	}

	return protoMetric
}

func (h *BenchmarkGRPCHandlers) isValidMetric(metric models.Metric) bool {
	if metric.ID == "" || metric.MType == "" {
		return false
	}

	switch metric.MType {
	case models.Gauge:
		return metric.Value != nil
	case models.Counter:
		return metric.Delta != nil
	default:
		return false
	}
}

func BenchmarkPing(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{
		metricsService: &benchmarkMetricsService{},
	}
	ctx := context.Background()
	req := &rpc.PingRequest{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := handler.Ping(ctx, req)
		if err != nil {
			b.Fatalf("Ping failed: %v", err)
		}
	}
}

func BenchmarkValue(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{
		metricsService: &benchmarkMetricsService{},
	}
	ctx := context.Background()

	benchmarks := []struct {
		name       string
		metricType string
		metricName string
	}{
		{"Gauge", "gauge", "temperature"},
		{"Counter", "counter", "requests"},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			req := &rpc.ValueRequest{
				MetricType: bb.metricType,
				MetricName: bb.metricName,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := handler.Value(ctx, req)
				if err != nil {
					b.Fatalf("Value failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkUpdate(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{
		metricsService: &benchmarkMetricsService{},
	}
	ctx := context.Background()

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
			metrics := make([]*rpc.Metric, bb.numMetrics)
			for i := 0; i < bb.numMetrics; i++ {
				if i%2 == 0 {
					metrics[i] = &rpc.Metric{
						Id:    string(rune('a' + i%26)),
						Mtype: "gauge",
						Value: 25.5 + float64(i),
					}
				} else {
					metrics[i] = &rpc.Metric{
						Id:    string(rune('a' + i%26)),
						Mtype: "counter",
						Delta: int64(100 + i),
					}
				}
			}

			req := &rpc.UpdateRequest{
				Metrics: metrics,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := handler.Update(ctx, req)
				if err != nil {
					b.Fatalf("Update failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkProtoToModel(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{}

	benchmarks := []struct {
		name   string
		metric *rpc.Metric
	}{
		{"Gauge", &rpc.Metric{
			Id:    "test_gauge",
			Mtype: "gauge",
			Value: 25.5,
		}},
		{"Counter", &rpc.Metric{
			Id:    "test_counter",
			Mtype: "counter",
			Delta: 100,
		}},
		{"WithHash", &rpc.Metric{
			Id:    "test_with_hash",
			Mtype: "gauge",
			Value: 25.5,
			Hash:  "abc123",
		}},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = handler.protoToModel(bb.metric)
			}
		})
	}
}

func BenchmarkModelToProto(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{}

	value := 25.5
	delta := int64(100)

	benchmarks := []struct {
		name   string
		metric models.Metric
	}{
		{"Gauge", models.Metric{
			ID:    "test_gauge",
			MType: "gauge",
			Value: &value,
		}},
		{"Counter", models.Metric{
			ID:    "test_counter",
			MType: "counter",
			Delta: &delta,
		}},
		{"WithHash", models.Metric{
			ID:    "test_with_hash",
			MType: "gauge",
			Value: &value,
			Hash:  "abc123",
		}},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = handler.modelToProto(bb.metric)
			}
		})
	}
}

func BenchmarkIsValidMetric(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{}

	value := 25.5
	delta := int64(100)

	benchmarks := []struct {
		name   string
		metric models.Metric
	}{
		{"ValidGauge", models.Metric{
			ID:    "test_gauge",
			MType: "gauge",
			Value: &value,
		}},
		{"ValidCounter", models.Metric{
			ID:    "test_counter",
			MType: "counter",
			Delta: &delta,
		}},
		{"InvalidNoID", models.Metric{
			ID:    "",
			MType: "gauge",
			Value: &value,
		}},
		{"InvalidNoType", models.Metric{
			ID:    "test",
			MType: "",
			Value: &value,
		}},
		{"InvalidGaugeNoValue", models.Metric{
			ID:    "test",
			MType: "gauge",
			Value: nil,
		}},
		{"InvalidCounterNoDelta", models.Metric{
			ID:    "test",
			MType: "counter",
			Delta: nil,
		}},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = handler.isValidMetric(bb.metric)
			}
		})
	}
}

func BenchmarkUpdateWithDifferentPayloadSizes(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{
		metricsService: &benchmarkMetricsService{},
	}
	ctx := context.Background()
	createMetrics := func(numMetrics, dataSize int) []*rpc.Metric {
		metrics := make([]*rpc.Metric, numMetrics)
		for i := 0; i < numMetrics; i++ {
			id := string(make([]byte, dataSize))
			metrics[i] = &rpc.Metric{
				Id:    id,
				Mtype: "gauge",
				Value: 25.5,
			}
		}
		return metrics
	}

	benchmarks := []struct {
		name       string
		numMetrics int
		dataSize   int
	}{
		{"SmallPayload", 10, 10},
		{"MediumPayload", 50, 50},
		{"LargePayload", 100, 100},
		{"ManySmallMetrics", 1000, 5},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			metrics := createMetrics(bb.numMetrics, bb.dataSize)
			req := &rpc.UpdateRequest{
				Metrics: metrics,
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := handler.Update(ctx, req)
				if err != nil {
					b.Fatalf("Update failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkConcurrentPing(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{
		metricsService: &benchmarkMetricsService{},
	}
	ctx := context.Background()
	req := &rpc.PingRequest{}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := handler.Ping(ctx, req)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
		}
	})
}

func BenchmarkConcurrentValue(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{
		metricsService: &benchmarkMetricsService{},
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			var req *rpc.ValueRequest
			if counter%2 == 0 {
				req = &rpc.ValueRequest{
					MetricType: "gauge",
					MetricName: "temperature",
				}
			} else {
				req = &rpc.ValueRequest{
					MetricType: "counter",
					MetricName: "requests",
				}
			}

			_, err := handler.Value(ctx, req)
			if err != nil {
				b.Fatalf("Value failed: %v", err)
			}
			counter++
		}
	})
}

func BenchmarkMemoryUsage(b *testing.B) {
	handler := &BenchmarkGRPCHandlers{
		metricsService: &benchmarkMetricsService{},
	}
	ctx := context.Background()
	metrics := make([]*rpc.Metric, 1000)
	for i := 0; i < 1000; i++ {
		metrics[i] = &rpc.Metric{
			Id:    string(rune('a' + i%26)),
			Mtype: "gauge",
			Value: 25.5 + float64(i),
		}
	}

	req := &rpc.UpdateRequest{
		Metrics: metrics,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := handler.Update(ctx, req)
		if err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}
