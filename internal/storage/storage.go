package storage

import (
	"context"
)

type Storage interface {
	SetGauge(ctx context.Context, name string, value float64) error
	AddCounter(ctx context.Context, name string, value int64) error
	GetGauge(ctx context.Context, name string) (*float64, error)
	GetCounter(ctx context.Context, name string) (*int64, error)
	Gauges(ctx context.Context) (map[string]float64, error)
	Counters(ctx context.Context) (map[string]int64, error)
	Save() error
	//IsUniqueViolationError() error
}
