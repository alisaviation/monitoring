package collector

import (
	"runtime"
)

type MockMemStatsReader struct{}

func (m *MockMemStatsReader) ReadMemStats(ms *runtime.MemStats) {
	*ms = runtime.MemStats{
		Alloc:         1000,
		BuckHashSys:   2000,
		Frees:         3000,
		GCCPUFraction: 0.1,
		GCSys:         4000,
		HeapAlloc:     5000,
		HeapIdle:      6000,
		HeapInuse:     7000,
		HeapObjects:   8000,
		HeapReleased:  9000,
		HeapSys:       10000,
		LastGC:        11000,
		Lookups:       12000,
		MCacheInuse:   13000,
		MCacheSys:     14000,
		MSpanInuse:    15000,
		MSpanSys:      16000,
		Mallocs:       17000,
		NextGC:        18000,
		NumForcedGC:   19000,
		NumGC:         20000,
		OtherSys:      21000,
		PauseTotalNs:  22000,
		StackInuse:    23000,
		StackSys:      24000,
		Sys:           25000,
		TotalAlloc:    26000,
	}
}
