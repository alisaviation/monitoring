package collector

import (
	"math/rand"
	"testing"

	"github.com/alisaviation/monitoring/internal/models"
)

func TestCollectMetrics(t *testing.T) {

	mockReader := &MockMemStatsReader{}
	c := &Collector{
		metrics: make(map[string]*models.Metric),
		reader:  mockReader,
	}
	c.initMetrics()

	metrics := c.CollectMetrics()
	expectedMetrics := []string{
		models.Alloc, models.BuckHashSys, models.Frees, models.GCCPUFraction, models.GCSys, models.HeapAlloc, models.HeapIdle, models.HeapInuse,
		models.HeapObjects, models.HeapReleased, models.HeapSys, models.LastGC, models.Lookups, models.MCacheInuse, models.MCacheSys, models.MSpanInuse,
		models.MSpanSys, models.Mallocs, models.NextGC, models.NumForcedGC, models.NumGC, models.OtherSys, models.PauseTotalNs, models.StackInuse,
		models.StackSys, models.Sys, models.TotalAlloc, models.RandomValue, models.PollCount,
	}

	for _, id := range expectedMetrics {
		if _, exists := metrics[id]; !exists {
			t.Errorf("Метрика %s отсутствует", id)
		}
	}
	tests := []struct {
		Name  string
		Value float64
		Type  models.MetricType
	}{
		{Name: models.Alloc, Value: 1000, Type: models.Gauge},
		{Name: models.BuckHashSys, Value: 2000, Type: models.Gauge},
		{Name: models.Frees, Value: 3000, Type: models.Gauge},
		{Name: models.GCCPUFraction, Value: 0.1, Type: models.Gauge},
		{Name: models.GCSys, Value: 4000, Type: models.Gauge},
		{Name: models.HeapAlloc, Value: 5000, Type: models.Gauge},
		{Name: models.HeapIdle, Value: 6000, Type: models.Gauge},
		{Name: models.HeapInuse, Value: 7000, Type: models.Gauge},
		{Name: models.HeapObjects, Value: 8000, Type: models.Gauge},
		{Name: models.HeapReleased, Value: 9000, Type: models.Gauge},
		{Name: models.HeapSys, Value: 10000, Type: models.Gauge},
		{Name: models.LastGC, Value: 11000, Type: models.Gauge},
		{Name: models.Lookups, Value: 12000, Type: models.Gauge},
		{Name: models.MCacheInuse, Value: 13000, Type: models.Gauge},
		{Name: models.MCacheSys, Value: 14000, Type: models.Gauge},
		{Name: models.MSpanInuse, Value: 15000, Type: models.Gauge},
		{Name: models.MSpanSys, Value: 16000, Type: models.Gauge},
		{Name: models.Mallocs, Value: 17000, Type: models.Gauge},
		{Name: models.NextGC, Value: 18000, Type: models.Gauge},
		{Name: models.NumForcedGC, Value: 19000, Type: models.Gauge},
		{Name: models.NumGC, Value: 20000, Type: models.Gauge},
		{Name: models.OtherSys, Value: 21000, Type: models.Gauge},
		{Name: models.PauseTotalNs, Value: 22000, Type: models.Gauge},
		{Name: models.StackInuse, Value: 23000, Type: models.Gauge},
		{Name: models.StackSys, Value: 24000, Type: models.Gauge},
		{Name: models.Sys, Value: 25000, Type: models.Gauge},
		{Name: models.TotalAlloc, Value: 26000, Type: models.Gauge},
		{Name: models.RandomValue, Value: rand.Float64(), Type: models.Gauge},
		{Name: models.PollCount, Value: 1.0, Type: models.Counter},
	}

	for _, test := range tests {
		metric, exists := metrics[test.Name]
		if !exists {
			t.Errorf("Метрика %s отсутствует", test.Name)
			continue
		}

		if metric.Type != test.Type {
			t.Errorf("Неверный тип метрики %s: ожидается %s, получено %s", test.Name, test.Type, metric.Type)
		}

		if test.Name == "RandomValue" {
			// RandomValue не может быть точно проверен, так как он генерируется случайным образом
			continue
		}

		if metric.Value != test.Value {
			t.Errorf("Неверное значение метрики %s: ожидается %v, получено %v", test.Name, test.Value, metric.Value)
		}
	}
}
