package collector

import (
	"math/rand"
	"runtime"

	"github.com/alisaviation/monitoring/internal/models"
)

type MemStatsReader interface {
	ReadMemStats(*runtime.MemStats)
}

type RealMemStatsReader struct{}

func (r *RealMemStatsReader) ReadMemStats(ms *runtime.MemStats) {
	runtime.ReadMemStats(ms)
}

type Collector struct {
	metrics map[string]*models.Metric
	reader  MemStatsReader
}

func NewCollector() *Collector {
	c := &Collector{
		metrics: make(map[string]*models.Metric),
		reader:  &RealMemStatsReader{},
	}
	c.initMetrics()
	return c
}

func (c *Collector) initMetrics() {
	c.metrics[models.Alloc] = &models.Metric{ID: models.Alloc, Value: new(float64), MType: models.Gauge}
	c.metrics[models.BuckHashSys] = &models.Metric{ID: models.BuckHashSys, Value: new(float64), MType: models.Gauge}
	c.metrics[models.Frees] = &models.Metric{ID: models.Frees, Value: new(float64), MType: models.Gauge}
	c.metrics[models.GCCPUFraction] = &models.Metric{ID: models.GCCPUFraction, Value: new(float64), MType: models.Gauge}
	c.metrics[models.GCSys] = &models.Metric{ID: models.GCSys, Value: new(float64), MType: models.Gauge}
	c.metrics[models.HeapAlloc] = &models.Metric{ID: models.HeapAlloc, Value: new(float64), MType: models.Gauge}
	c.metrics[models.HeapIdle] = &models.Metric{ID: models.HeapIdle, Value: new(float64), MType: models.Gauge}
	c.metrics[models.HeapInuse] = &models.Metric{ID: models.HeapInuse, Value: new(float64), MType: models.Gauge}
	c.metrics[models.HeapObjects] = &models.Metric{ID: models.HeapObjects, Value: new(float64), MType: models.Gauge}
	c.metrics[models.HeapReleased] = &models.Metric{ID: models.HeapReleased, Value: new(float64), MType: models.Gauge}
	c.metrics[models.HeapSys] = &models.Metric{ID: models.HeapSys, Value: new(float64), MType: models.Gauge}
	c.metrics[models.LastGC] = &models.Metric{ID: models.LastGC, Value: new(float64), MType: models.Gauge}
	c.metrics[models.Lookups] = &models.Metric{ID: models.Lookups, Value: new(float64), MType: models.Gauge}
	c.metrics[models.MCacheInuse] = &models.Metric{ID: models.MCacheInuse, Value: new(float64), MType: models.Gauge}
	c.metrics[models.MCacheSys] = &models.Metric{ID: models.MCacheSys, Value: new(float64), MType: models.Gauge}
	c.metrics[models.MSpanInuse] = &models.Metric{ID: models.MSpanInuse, Value: new(float64), MType: models.Gauge}
	c.metrics[models.MSpanSys] = &models.Metric{ID: models.MSpanSys, Value: new(float64), MType: models.Gauge}
	c.metrics[models.Mallocs] = &models.Metric{ID: models.Mallocs, Value: new(float64), MType: models.Gauge}
	c.metrics[models.NextGC] = &models.Metric{ID: models.NextGC, Value: new(float64), MType: models.Gauge}
	c.metrics[models.NumForcedGC] = &models.Metric{ID: models.NumForcedGC, Value: new(float64), MType: models.Gauge}
	c.metrics[models.NumGC] = &models.Metric{ID: models.NumGC, Value: new(float64), MType: models.Gauge}
	c.metrics[models.OtherSys] = &models.Metric{ID: models.OtherSys, Value: new(float64), MType: models.Gauge}
	c.metrics[models.PauseTotalNs] = &models.Metric{ID: models.PauseTotalNs, Value: new(float64), MType: models.Gauge}
	c.metrics[models.StackInuse] = &models.Metric{ID: models.StackInuse, Value: new(float64), MType: models.Gauge}
	c.metrics[models.StackSys] = &models.Metric{ID: models.StackSys, Value: new(float64), MType: models.Gauge}
	c.metrics[models.Sys] = &models.Metric{ID: models.Sys, Value: new(float64), MType: models.Gauge}
	c.metrics[models.TotalAlloc] = &models.Metric{ID: models.TotalAlloc, Value: new(float64), MType: models.Gauge}

	c.metrics[models.RandomValue] = &models.Metric{ID: models.RandomValue, Value: new(float64), MType: models.Gauge}
	c.metrics[models.PollCount] = &models.Metric{ID: models.PollCount, Delta: new(int64), MType: models.Counter}
}

func (c *Collector) CollectMetrics() map[string]*models.Metric {
	var memStats runtime.MemStats
	c.reader.ReadMemStats(&memStats)

	*c.metrics[models.Alloc].Value = float64(memStats.Alloc)
	*c.metrics[models.BuckHashSys].Value = float64(memStats.BuckHashSys)
	*c.metrics[models.Frees].Value = float64(memStats.Frees)
	*c.metrics[models.GCCPUFraction].Value = memStats.GCCPUFraction
	*c.metrics[models.GCSys].Value = float64(memStats.GCSys)
	*c.metrics[models.HeapAlloc].Value = float64(memStats.HeapAlloc)
	*c.metrics[models.HeapIdle].Value = float64(memStats.HeapIdle)
	*c.metrics[models.HeapInuse].Value = float64(memStats.HeapInuse)
	*c.metrics[models.HeapObjects].Value = float64(memStats.HeapObjects)
	*c.metrics[models.HeapReleased].Value = float64(memStats.HeapReleased)
	*c.metrics[models.HeapSys].Value = float64(memStats.HeapSys)
	*c.metrics[models.LastGC].Value = float64(memStats.LastGC)
	*c.metrics[models.Lookups].Value = float64(memStats.Lookups)
	*c.metrics[models.MCacheInuse].Value = float64(memStats.MCacheInuse)
	*c.metrics[models.MCacheSys].Value = float64(memStats.MCacheSys)
	*c.metrics[models.MSpanInuse].Value = float64(memStats.MSpanInuse)
	*c.metrics[models.MSpanSys].Value = float64(memStats.MSpanSys)
	*c.metrics[models.Mallocs].Value = float64(memStats.Mallocs)
	*c.metrics[models.NextGC].Value = float64(memStats.NextGC)
	*c.metrics[models.NumForcedGC].Value = float64(memStats.NumForcedGC)
	*c.metrics[models.NumGC].Value = float64(memStats.NumGC)
	*c.metrics[models.OtherSys].Value = float64(memStats.OtherSys)
	*c.metrics[models.PauseTotalNs].Value = float64(memStats.PauseTotalNs)
	*c.metrics[models.StackInuse].Value = float64(memStats.StackInuse)
	*c.metrics[models.StackSys].Value = float64(memStats.StackSys)
	*c.metrics[models.Sys].Value = float64(memStats.Sys)
	*c.metrics[models.TotalAlloc].Value = float64(memStats.TotalAlloc)

	*c.metrics[models.RandomValue].Value = rand.Float64()
	*c.metrics[models.PollCount].Delta++

	return c.metrics
}

func UpdateMetricsBuffer(metricsBuffer map[string]*models.Metric, metrics map[string]*models.Metric) {
	for name, metric := range metrics {
		if metric.MType == models.Counter {
			if existingMetric, exists := metricsBuffer[name]; exists {
				metricsBuffer[name] = &models.Metric{
					ID:    existingMetric.ID,
					Value: existingMetric.Value,
					Delta: new(int64),
					MType: existingMetric.MType,
				}
				*metricsBuffer[name].Delta = *existingMetric.Delta + *metric.Delta
			} else {
				metricsBuffer[name] = metric
			}
		} else {
			metricsBuffer[name] = metric
		}
	}
}
