package collector

import (
	"testing"

	"github.com/alisaviation/monitoring/internal/models"
)

func BenchmarkCollectMetrics(b *testing.B) {
	c := NewCollector()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.CollectMetrics()
	}
}

func BenchmarkCollectGopsutilMetrics(b *testing.B) {
	c := NewCollector()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.СollectGopsutilMetrics()
	}
}

func BenchmarkUpdateMetricsBuffer(b *testing.B) {
	c := NewCollector()
	metrics := c.CollectMetrics()
	metricsBuffer := make(map[string]*models.Metric)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		UpdateMetricsBuffer(metricsBuffer, metrics)
	}
}
