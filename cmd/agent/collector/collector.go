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
	metrics map[string]models.Metric
	reader  MemStatsReader
}

func NewCollector() *Collector {
	return &Collector{
		metrics: make(map[string]models.Metric),
		reader:  &RealMemStatsReader{},
	}
}

func (c *Collector) CollectMetrics() map[string]models.Metric {
	var memStats runtime.MemStats
	c.reader.ReadMemStats(&memStats)

	c.metrics["Alloc"] = models.Metric{ID: "Alloc", Value: float64(memStats.Alloc), Type: models.Gauge}
	c.metrics["BuckHashSys"] = models.Metric{ID: "BuckHashSys", Value: float64(memStats.BuckHashSys), Type: models.Gauge}
	c.metrics["Frees"] = models.Metric{ID: "Frees", Value: float64(memStats.Frees), Type: models.Gauge}
	c.metrics["GCCPUFraction"] = models.Metric{ID: "GCCPUFraction", Value: memStats.GCCPUFraction, Type: models.Gauge}
	c.metrics["GCSys"] = models.Metric{ID: "GCSys", Value: float64(memStats.GCSys), Type: models.Gauge}
	c.metrics["HeapAlloc"] = models.Metric{ID: "HeapAlloc", Value: float64(memStats.HeapAlloc), Type: models.Gauge}
	c.metrics["HeapIdle"] = models.Metric{ID: "HeapIdle", Value: float64(memStats.HeapIdle), Type: models.Gauge}
	c.metrics["HeapInuse"] = models.Metric{ID: "HeapInuse", Value: float64(memStats.HeapInuse), Type: models.Gauge}
	c.metrics["HeapObjects"] = models.Metric{ID: "HeapObjects", Value: float64(memStats.HeapObjects), Type: models.Gauge}
	c.metrics["HeapReleased"] = models.Metric{ID: "HeapReleased", Value: float64(memStats.HeapReleased), Type: models.Gauge}
	c.metrics["HeapSys"] = models.Metric{ID: "HeapSys", Value: float64(memStats.HeapSys), Type: models.Gauge}
	c.metrics["LastGC"] = models.Metric{ID: "LastGC", Value: float64(memStats.LastGC), Type: models.Gauge}
	c.metrics["Lookups"] = models.Metric{ID: "Lookups", Value: float64(memStats.Lookups), Type: models.Gauge}
	c.metrics["MCacheInuse"] = models.Metric{ID: "MCacheInuse", Value: float64(memStats.MCacheInuse), Type: models.Gauge}
	c.metrics["MCacheSys"] = models.Metric{ID: "MCacheSys", Value: float64(memStats.MCacheSys), Type: models.Gauge}
	c.metrics["MSpanInuse"] = models.Metric{ID: "MSpanInuse", Value: float64(memStats.MSpanInuse), Type: models.Gauge}
	c.metrics["MSpanSys"] = models.Metric{ID: "MSpanSys", Value: float64(memStats.MSpanSys), Type: models.Gauge}
	c.metrics["Mallocs"] = models.Metric{ID: "Mallocs", Value: float64(memStats.Mallocs), Type: models.Gauge}
	c.metrics["NextGC"] = models.Metric{ID: "NextGC", Value: float64(memStats.NextGC), Type: models.Gauge}
	c.metrics["NumForcedGC"] = models.Metric{ID: "NumForcedGC", Value: float64(memStats.NumForcedGC), Type: models.Gauge}
	c.metrics["NumGC"] = models.Metric{ID: "NumGC", Value: float64(memStats.NumGC), Type: models.Gauge}
	c.metrics["OtherSys"] = models.Metric{ID: "OtherSys", Value: float64(memStats.OtherSys), Type: models.Gauge}
	c.metrics["PauseTotalNs"] = models.Metric{ID: "PauseTotalNs", Value: float64(memStats.PauseTotalNs), Type: models.Gauge}
	c.metrics["StackInuse"] = models.Metric{ID: "StackInuse", Value: float64(memStats.StackInuse), Type: models.Gauge}
	c.metrics["StackSys"] = models.Metric{ID: "StackSys", Value: float64(memStats.StackSys), Type: models.Gauge}
	c.metrics["Sys"] = models.Metric{ID: "Sys", Value: float64(memStats.Sys), Type: models.Gauge}
	c.metrics["TotalAlloc"] = models.Metric{ID: "TotalAlloc", Value: float64(memStats.TotalAlloc), Type: models.Gauge}

	c.metrics["RandomValue"] = models.Metric{ID: "RandomValue", Value: rand.Float64(), Type: models.Gauge}
	c.metrics["PollCount"] = models.Metric{ID: "PollCount", Value: 1, Type: models.Counter}

	return c.metrics
}
