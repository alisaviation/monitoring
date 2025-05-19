package storage

import (
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

func (m *MemStorage) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *MemStorage) AddCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *MemStorage) GetGauge(name string) (*float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.gauges[name]
	return &value, exists
}

func (m *MemStorage) GetCounter(name string) (*int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.counters[name]
	return &value, exists
}

func (m *MemStorage) Gauges() map[string]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gauges
}

func (m *MemStorage) Counters() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters
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
