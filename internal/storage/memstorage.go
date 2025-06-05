package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"sync"
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	mu       sync.Mutex
	filePath string
}

func NewMemStorage(filePath string) *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		filePath: filePath,
	}
}

func (m *MemStorage) SetGauge(ctx context.Context, name string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
	return nil
}

func (m *MemStorage) AddCounter(ctx context.Context, name string, value int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
	return nil
}

func (m *MemStorage) GetGauge(ctx context.Context, name string) (*float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.gauges[name]
	if !exists {
		return nil, sql.ErrNoRows
	}
	return &value, nil
}

func (m *MemStorage) GetCounter(ctx context.Context, name string) (*int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.counters[name]
	if !exists {
		return nil, sql.ErrNoRows
	}
	return &value, nil
}

func (m *MemStorage) Gauges(ctx context.Context) (map[string]float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gauges, nil
}

func (m *MemStorage) Counters(ctx context.Context) (map[string]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters, nil
}

func (m *MemStorage) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data := struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}{
		Gauges:   m.gauges,
		Counters: m.counters,
	}
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	jsonData = append(jsonData, '\n')

	file, err := os.Create(m.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	return err
}

func (m *MemStorage) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Open(m.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		Gauges   map[string]float64 `json:"gauges"`
		Counters map[string]int64   `json:"counters"`
	}{}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	m.gauges = data.Gauges
	m.counters = data.Counters

	return nil
}

func (m *MemStorage) IsUniqueViolationError(err error) bool {
	return false
}
