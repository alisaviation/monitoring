// Package storage defines the interface for metric storage backends.
package storage

import (
	"context"
)

// Storage defines the interface for metric storage operations.
type Storage interface {
	// SetGauge stores a gauge metric with the given name and value.
	SetGauge(ctx context.Context, name string, value float64) error

	// AddCounter increments a counter metric by the given value.
	AddCounter(ctx context.Context, name string, value int64) error

	// GetGauge retrieves a gauge metric by name.
	// Returns the metric value or nil if not found.
	GetGauge(ctx context.Context, name string) (*float64, error)

	// GetCounter retrieves a counter metric by name.
	// Returns the metric value or nil if not found.
	GetCounter(ctx context.Context, name string) (*int64, error)

	// Gauges returns all stored gauge metrics.
	Gauges(ctx context.Context) (map[string]float64, error)

	// Counters returns all stored counter metrics.
	Counters(ctx context.Context) (map[string]int64, error)

	// Save persists the current state of metrics to durable storage.
	Save() error

	// IsUniqueViolationError checks if an error is due to a unique constraint violation.
	IsUniqueViolationError(err error) bool
}
