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

	c.metrics["Alloc"] = models.Metric{Name: "Alloc", Value: float64(memStats.Alloc), Type: models.Gauge}
	c.metrics["BuckHashSys"] = models.Metric{Name: "BuckHashSys", Value: float64(memStats.BuckHashSys), Type: models.Gauge}
	c.metrics["Frees"] = models.Metric{Name: "Frees", Value: float64(memStats.Frees), Type: models.Gauge}
	c.metrics["GCCPUFraction"] = models.Metric{Name: "GCCPUFraction", Value: memStats.GCCPUFraction, Type: models.Gauge}
	c.metrics["GCSys"] = models.Metric{Name: "GCSys", Value: float64(memStats.GCSys), Type: models.Gauge}
	c.metrics["HeapAlloc"] = models.Metric{Name: "HeapAlloc", Value: float64(memStats.HeapAlloc), Type: models.Gauge}
	c.metrics["HeapIdle"] = models.Metric{Name: "HeapIdle", Value: float64(memStats.HeapIdle), Type: models.Gauge}
	c.metrics["HeapInuse"] = models.Metric{Name: "HeapInuse", Value: float64(memStats.HeapInuse), Type: models.Gauge}
	c.metrics["HeapObjects"] = models.Metric{Name: "HeapObjects", Value: float64(memStats.HeapObjects), Type: models.Gauge}
	c.metrics["HeapReleased"] = models.Metric{Name: "HeapReleased", Value: float64(memStats.HeapReleased), Type: models.Gauge}
	c.metrics["HeapSys"] = models.Metric{Name: "HeapSys", Value: float64(memStats.HeapSys), Type: models.Gauge}
	c.metrics["LastGC"] = models.Metric{Name: "LastGC", Value: float64(memStats.LastGC), Type: models.Gauge}
	c.metrics["Lookups"] = models.Metric{Name: "Lookups", Value: float64(memStats.Lookups), Type: models.Gauge}
	c.metrics["MCacheInuse"] = models.Metric{Name: "MCacheInuse", Value: float64(memStats.MCacheInuse), Type: models.Gauge}
	c.metrics["MCacheSys"] = models.Metric{Name: "MCacheSys", Value: float64(memStats.MCacheSys), Type: models.Gauge}
	c.metrics["MSpanInuse"] = models.Metric{Name: "MSpanInuse", Value: float64(memStats.MSpanInuse), Type: models.Gauge}
	c.metrics["MSpanSys"] = models.Metric{Name: "MSpanSys", Value: float64(memStats.MSpanSys), Type: models.Gauge}
	c.metrics["Mallocs"] = models.Metric{Name: "Mallocs", Value: float64(memStats.Mallocs), Type: models.Gauge}
	c.metrics["NextGC"] = models.Metric{Name: "NextGC", Value: float64(memStats.NextGC), Type: models.Gauge}
	c.metrics["NumForcedGC"] = models.Metric{Name: "NumForcedGC", Value: float64(memStats.NumForcedGC), Type: models.Gauge}
	c.metrics["NumGC"] = models.Metric{Name: "NumGC", Value: float64(memStats.NumGC), Type: models.Gauge}
	c.metrics["OtherSys"] = models.Metric{Name: "OtherSys", Value: float64(memStats.OtherSys), Type: models.Gauge}
	c.metrics["PauseTotalNs"] = models.Metric{Name: "PauseTotalNs", Value: float64(memStats.PauseTotalNs), Type: models.Gauge}
	c.metrics["StackInuse"] = models.Metric{Name: "StackInuse", Value: float64(memStats.StackInuse), Type: models.Gauge}
	c.metrics["StackSys"] = models.Metric{Name: "StackSys", Value: float64(memStats.StackSys), Type: models.Gauge}
	c.metrics["Sys"] = models.Metric{Name: "Sys", Value: float64(memStats.Sys), Type: models.Gauge}
	c.metrics["TotalAlloc"] = models.Metric{Name: "TotalAlloc", Value: float64(memStats.TotalAlloc), Type: models.Gauge}

	c.metrics["RandomValue"] = models.Metric{Name: "RandomValue", Value: rand.Float64(), Type: models.Gauge}
	c.metrics["PollCount"] = models.Metric{Name: "PollCount", Value: 1, Type: models.Counter}

	return c.metrics
}
