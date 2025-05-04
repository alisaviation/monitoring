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
	c.metrics[models.Alloc] = &models.Metric{Name: models.Alloc, Value: 0, Type: models.Gauge}
	c.metrics[models.BuckHashSys] = &models.Metric{Name: models.BuckHashSys, Value: 0, Type: models.Gauge}
	c.metrics[models.Frees] = &models.Metric{Name: models.Frees, Value: 0, Type: models.Gauge}
	c.metrics[models.GCCPUFraction] = &models.Metric{Name: models.GCCPUFraction, Value: 0, Type: models.Gauge}
	c.metrics[models.GCSys] = &models.Metric{Name: models.GCSys, Value: 0, Type: models.Gauge}
	c.metrics[models.HeapAlloc] = &models.Metric{Name: models.HeapAlloc, Value: 0, Type: models.Gauge}
	c.metrics[models.HeapIdle] = &models.Metric{Name: models.HeapIdle, Value: 0, Type: models.Gauge}
	c.metrics[models.HeapInuse] = &models.Metric{Name: models.HeapInuse, Value: 0, Type: models.Gauge}
	c.metrics[models.HeapObjects] = &models.Metric{Name: models.HeapObjects, Value: 0, Type: models.Gauge}
	c.metrics[models.HeapReleased] = &models.Metric{Name: models.HeapReleased, Value: 0, Type: models.Gauge}
	c.metrics[models.HeapSys] = &models.Metric{Name: models.HeapSys, Value: 0, Type: models.Gauge}
	c.metrics[models.LastGC] = &models.Metric{Name: models.LastGC, Value: 0, Type: models.Gauge}
	c.metrics[models.Lookups] = &models.Metric{Name: models.Lookups, Value: 0, Type: models.Gauge}
	c.metrics[models.MCacheInuse] = &models.Metric{Name: models.MCacheInuse, Value: 0, Type: models.Gauge}
	c.metrics[models.MCacheSys] = &models.Metric{Name: models.MCacheSys, Value: 0, Type: models.Gauge}
	c.metrics[models.MSpanInuse] = &models.Metric{Name: models.MSpanInuse, Value: 0, Type: models.Gauge}
	c.metrics[models.MSpanSys] = &models.Metric{Name: models.MSpanSys, Value: 0, Type: models.Gauge}
	c.metrics[models.Mallocs] = &models.Metric{Name: models.Mallocs, Value: 0, Type: models.Gauge}
	c.metrics[models.NextGC] = &models.Metric{Name: models.NextGC, Value: 0, Type: models.Gauge}
	c.metrics[models.NumForcedGC] = &models.Metric{Name: models.NumForcedGC, Value: 0, Type: models.Gauge}
	c.metrics[models.NumGC] = &models.Metric{Name: models.NumGC, Value: 0, Type: models.Gauge}
	c.metrics[models.OtherSys] = &models.Metric{Name: models.OtherSys, Value: 0, Type: models.Gauge}
	c.metrics[models.PauseTotalNs] = &models.Metric{Name: models.PauseTotalNs, Value: 0, Type: models.Gauge}
	c.metrics[models.StackInuse] = &models.Metric{Name: models.StackInuse, Value: 0, Type: models.Gauge}
	c.metrics[models.StackSys] = &models.Metric{Name: models.StackSys, Value: 0, Type: models.Gauge}
	c.metrics[models.Sys] = &models.Metric{Name: models.Sys, Value: 0, Type: models.Gauge}
	c.metrics[models.TotalAlloc] = &models.Metric{Name: models.TotalAlloc, Value: 0, Type: models.Gauge}

	c.metrics[models.RandomValue] = &models.Metric{Name: models.RandomValue, Value: 0, Type: models.Gauge}
	c.metrics[models.PollCount] = &models.Metric{Name: models.PollCount, Value: 0, Type: models.Counter}
}

func (c *Collector) CollectMetrics() map[string]*models.Metric {
	var memStats runtime.MemStats
	c.reader.ReadMemStats(&memStats)

	c.metrics[models.Alloc].Value = float64(memStats.Alloc)
	c.metrics[models.BuckHashSys].Value = float64(memStats.BuckHashSys)
	c.metrics[models.Frees].Value = float64(memStats.Frees)
	c.metrics[models.GCCPUFraction].Value = memStats.GCCPUFraction
	c.metrics[models.GCSys].Value = float64(memStats.GCSys)
	c.metrics[models.HeapAlloc].Value = float64(memStats.HeapAlloc)
	c.metrics[models.HeapIdle].Value = float64(memStats.HeapIdle)
	c.metrics[models.HeapInuse].Value = float64(memStats.HeapInuse)
	c.metrics[models.HeapObjects].Value = float64(memStats.HeapObjects)
	c.metrics[models.HeapReleased].Value = float64(memStats.HeapReleased)
	c.metrics[models.HeapSys].Value = float64(memStats.HeapSys)
	c.metrics[models.LastGC].Value = float64(memStats.LastGC)
	c.metrics[models.Lookups].Value = float64(memStats.Lookups)
	c.metrics[models.MCacheInuse].Value = float64(memStats.MCacheInuse)
	c.metrics[models.MCacheSys].Value = float64(memStats.MCacheSys)
	c.metrics[models.MSpanInuse].Value = float64(memStats.MSpanInuse)
	c.metrics[models.MSpanSys].Value = float64(memStats.MSpanSys)
	c.metrics[models.Mallocs].Value = float64(memStats.Mallocs)
	c.metrics[models.NextGC].Value = float64(memStats.NextGC)
	c.metrics[models.NumForcedGC].Value = float64(memStats.NumForcedGC)
	c.metrics[models.NumGC].Value = float64(memStats.NumGC)
	c.metrics[models.OtherSys].Value = float64(memStats.OtherSys)
	c.metrics[models.PauseTotalNs].Value = float64(memStats.PauseTotalNs)
	c.metrics[models.StackInuse].Value = float64(memStats.StackInuse)
	c.metrics[models.StackSys].Value = float64(memStats.StackSys)
	c.metrics[models.Sys].Value = float64(memStats.Sys)
	c.metrics[models.TotalAlloc].Value = float64(memStats.TotalAlloc)

	c.metrics[models.RandomValue].Value = rand.Float64()
	c.metrics[models.PollCount].Value += 1

	return c.metrics
}

func UpdateMetricsBuffer(metricsBuffer map[string]*models.Metric, metrics map[string]*models.Metric) {
	for name, metric := range metrics {
		if metric.Type == models.Counter {
			if existingMetric, exists := metricsBuffer[name]; exists {
				metricsBuffer[name] = &models.Metric{
					Name:  existingMetric.Name,
					Value: existingMetric.Value + metric.Value,
					Type:  existingMetric.Type,
				}
			} else {
				metricsBuffer[name] = metric
			}
		} else {
			metricsBuffer[name] = metric
		}
	}
}
