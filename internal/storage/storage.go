package storage

type Storage interface {
	SetGauge(name string, value float64) error
	AddCounter(name string, value int64) error
	GetGauge(name string) (*float64, bool)
	GetCounter(name string) (*int64, bool)
	Gauges() (map[string]float64, error)
	Counters() (map[string]int64, error)
}
