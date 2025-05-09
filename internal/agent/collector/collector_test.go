package collector

import (
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
			t.Errorf("Metric %s not found", id)
		}
	}

	tests := []struct {
		ID    string
		Value float64
		Type  string
		Delta int64
	}{
		{ID: models.Alloc, Value: 1000, Type: models.Gauge},
		{ID: models.BuckHashSys, Value: 2000, Type: models.Gauge},
		{ID: models.Frees, Value: 3000, Type: models.Gauge},
		{ID: models.GCCPUFraction, Value: 0.1, Type: models.Gauge},
		{ID: models.GCSys, Value: 4000, Type: models.Gauge},
		{ID: models.HeapAlloc, Value: 5000, Type: models.Gauge},
		{ID: models.HeapIdle, Value: 6000, Type: models.Gauge},
		{ID: models.HeapInuse, Value: 7000, Type: models.Gauge},
		{ID: models.HeapObjects, Value: 8000, Type: models.Gauge},
		{ID: models.HeapReleased, Value: 9000, Type: models.Gauge},
		{ID: models.HeapSys, Value: 10000, Type: models.Gauge},
		{ID: models.LastGC, Value: 11000, Type: models.Gauge},
		{ID: models.Lookups, Value: 12000, Type: models.Gauge},
		{ID: models.MCacheInuse, Value: 13000, Type: models.Gauge},
		{ID: models.MCacheSys, Value: 14000, Type: models.Gauge},
		{ID: models.MSpanInuse, Value: 15000, Type: models.Gauge},
		{ID: models.MSpanSys, Value: 16000, Type: models.Gauge},
		{ID: models.Mallocs, Value: 17000, Type: models.Gauge},
		{ID: models.NextGC, Value: 18000, Type: models.Gauge},
		{ID: models.NumForcedGC, Value: 19000, Type: models.Gauge},
		{ID: models.NumGC, Value: 20000, Type: models.Gauge},
		{ID: models.OtherSys, Value: 21000, Type: models.Gauge},
		{ID: models.PauseTotalNs, Value: 22000, Type: models.Gauge},
		{ID: models.StackInuse, Value: 23000, Type: models.Gauge},
		{ID: models.StackSys, Value: 24000, Type: models.Gauge},
		{ID: models.Sys, Value: 25000, Type: models.Gauge},
		{ID: models.TotalAlloc, Value: 26000, Type: models.Gauge},
		{ID: models.RandomValue, Type: models.Gauge},
		{ID: models.PollCount, Delta: 1, Type: models.Counter},
	}

	for _, test := range tests {
		metric, exists := metrics[test.ID]
		if !exists {
			t.Errorf("Metric %s not found", test.ID)
			continue
		}

		if metric.MType != test.Type {
			t.Errorf("Invalid metric type %s: expected %s, got %s", test.ID, test.Type, metric.MType)
		}

		if test.ID == models.RandomValue {
			// RandomValue не может быть точно проверен, так как он генерируется случайным образом
			if metric.Value == nil {
				t.Errorf("Metric %s: Value is nil", test.ID)
			}
			continue
		}

		if test.ID == models.PollCount {
			if metric.Delta == nil {
				t.Errorf("Metric %s: Delta is nil", test.ID)
				continue
			}
			if *metric.Delta != test.Delta {
				t.Errorf("Invalid metric delta %s: expected %d, got %d", test.ID, test.Delta, *metric.Delta)
			}
			continue
		}

		if metric.Value == nil {
			t.Errorf("Metric %s: Value is nil", test.ID)
			continue
		}
		if *metric.Value != test.Value {
			t.Errorf("Invalid metric value %s: expected %v, got %v", test.ID, test.Value, *metric.Value)
		}
	}
}

func TestUpdateMetricsBuffer(t *testing.T) {
	type args struct {
		metricsBuffer map[string]*models.Metric
		metrics       map[string]*models.Metric
	}
	tests := []struct {
		name           string
		args           args
		expectedBuffer map[string]*models.Metric
	}{
		{
			name: "Update existing Gauge metric",
			args: args{
				metricsBuffer: map[string]*models.Metric{
					models.Alloc: {ID: models.Alloc, Value: new(float64), MType: models.Gauge},
				},
				metrics: map[string]*models.Metric{
					models.Alloc: {ID: models.Alloc, Value: float64Ptr(1000), MType: models.Gauge},
				},
			},
			expectedBuffer: map[string]*models.Metric{
				models.Alloc: {ID: models.Alloc, Value: float64Ptr(1000), MType: models.Gauge},
			},
		},
		{
			name: "Add new Gauge metric",
			args: args{
				metricsBuffer: map[string]*models.Metric{},
				metrics: map[string]*models.Metric{
					models.Alloc: {ID: models.Alloc, Value: float64Ptr(1000), MType: models.Gauge},
				},
			},
			expectedBuffer: map[string]*models.Metric{
				models.Alloc: {ID: models.Alloc, Value: float64Ptr(1000), MType: models.Gauge},
			},
		},
		{
			name: "Update existing Counter metric",
			args: args{
				metricsBuffer: map[string]*models.Metric{
					models.PollCount: {ID: models.PollCount, Delta: int64Ptr(1), MType: models.Counter},
				},
				metrics: map[string]*models.Metric{
					models.PollCount: {ID: models.PollCount, Delta: int64Ptr(1), MType: models.Counter},
				},
			},
			expectedBuffer: map[string]*models.Metric{
				models.PollCount: {ID: models.PollCount, Delta: int64Ptr(2), MType: models.Counter},
			},
		},
		{
			name: "Add new Counter metric",
			args: args{
				metricsBuffer: map[string]*models.Metric{},
				metrics: map[string]*models.Metric{
					models.PollCount: {ID: models.PollCount, Delta: int64Ptr(1), MType: models.Counter},
				},
			},
			expectedBuffer: map[string]*models.Metric{
				models.PollCount: {ID: models.PollCount, Delta: int64Ptr(1), MType: models.Counter},
			},
		},
		{
			name: "Mixed Gauge and Counter metrics",
			args: args{
				metricsBuffer: map[string]*models.Metric{
					models.Alloc:     {ID: models.Alloc, Value: float64Ptr(1000), MType: models.Gauge},
					models.PollCount: {ID: models.PollCount, Delta: int64Ptr(1), MType: models.Counter},
				},
				metrics: map[string]*models.Metric{
					models.Alloc:     {ID: models.Alloc, Value: float64Ptr(2000), MType: models.Gauge},
					models.PollCount: {ID: models.PollCount, Delta: int64Ptr(1), MType: models.Counter},
				},
			},
			expectedBuffer: map[string]*models.Metric{
				models.Alloc:     {ID: models.Alloc, Value: float64Ptr(2000), MType: models.Gauge},
				models.PollCount: {ID: models.PollCount, Delta: int64Ptr(2), MType: models.Counter},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateMetricsBuffer(tt.args.metricsBuffer, tt.args.metrics)

			for name, expectedMetric := range tt.expectedBuffer {
				actualMetric, exists := tt.args.metricsBuffer[name]
				if !exists {
					t.Errorf("Metric %s not found in buffer", name)
					continue
				}

				if actualMetric.ID != expectedMetric.ID {
					t.Errorf("Metric %s: expected ID %s, got %s", name, expectedMetric.ID, actualMetric.ID)
				}
				if actualMetric.MType != expectedMetric.MType {
					t.Errorf("Metric %s: expected type %s, got %s", name, expectedMetric.MType, actualMetric.MType)
				}

				if expectedMetric.MType == models.Gauge {
					if actualMetric.Value == nil {
						t.Errorf("Metric %s: Value is nil", name)
					} else if *actualMetric.Value != *expectedMetric.Value {
						t.Errorf("Metric %s: expected value %v, got %v", name, *expectedMetric.Value, *actualMetric.Value)
					}
				} else if expectedMetric.MType == models.Counter {
					if actualMetric.Delta == nil {
						t.Errorf("Metric %s: Delta is nil", name)
					} else if *actualMetric.Delta != *expectedMetric.Delta {
						t.Errorf("Metric %s: expected delta %d, got %d", name, *expectedMetric.Delta, *actualMetric.Delta)
					}
				}
			}
		})
	}
}

// Helper functions to create pointers to float64 and int64
func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}
