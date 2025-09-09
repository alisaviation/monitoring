package sender

import (
	"context"
	"strconv"
	"testing"

	"github.com/alisaviation/monitoring/internal/models"
)

func BenchmarkSendMetricsBatch(b *testing.B) {
	s := NewSender("http://localhost:8080", "", nil)
	metrics := make(map[string]*models.Metric)

	for i := 0; i < 100; i++ {
		name := "metric" + strconv.Itoa(i)
		value := float64(i)
		metrics[name] = &models.Metric{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		}
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = s.SendMetricsBatch(ctx, metrics, "")
	}
}
