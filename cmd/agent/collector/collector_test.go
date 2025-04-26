package collector

import (
	"math/rand"
	"testing"

	"github.com/alisaviation/monitoring/internal/models"
)

func TestCollectMetrics(t *testing.T) {

	mockReader := &MockMemStatsReader{}
	c := &Collector{
		metrics: make(map[string]models.Metric),
		reader:  mockReader,
	}

	metrics := c.CollectMetrics()
	expectedMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse",
		"HeapObjects", "HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse",
		"MSpanSys", "Mallocs", "NextGC", "NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs", "StackInuse",
		"StackSys", "Sys", "TotalAlloc", "RandomValue", "PollCount",
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
		{"Alloc", 1000, models.Gauge},
		{"BuckHashSys", 2000, models.Gauge},
		{"Frees", 3000, models.Gauge},
		{"GCCPUFraction", 0.1, models.Gauge},
		{"GCSys", 4000, models.Gauge},
		{"HeapAlloc", 5000, models.Gauge},
		{"HeapIdle", 6000, models.Gauge},
		{"HeapInuse", 7000, models.Gauge},
		{"HeapObjects", 8000, models.Gauge},
		{"HeapReleased", 9000, models.Gauge},
		{"HeapSys", 10000, models.Gauge},
		{"LastGC", 11000, models.Gauge},
		{"Lookups", 12000, models.Gauge},
		{"MCacheInuse", 13000, models.Gauge},
		{"MCacheSys", 14000, models.Gauge},
		{"MSpanInuse", 15000, models.Gauge},
		{"MSpanSys", 16000, models.Gauge},
		{"Mallocs", 17000, models.Gauge},
		{"NextGC", 18000, models.Gauge},
		{"NumForcedGC", 19000, models.Gauge},
		{"NumGC", 20000, models.Gauge},
		{"OtherSys", 21000, models.Gauge},
		{"PauseTotalNs", 22000, models.Gauge},
		{"StackInuse", 23000, models.Gauge},
		{"StackSys", 24000, models.Gauge},
		{"Sys", 25000, models.Gauge},
		{"TotalAlloc", 26000, models.Gauge},
		{"RandomValue", rand.Float64(), models.Gauge},
		{"PollCount", 1.0, models.Counter},
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
