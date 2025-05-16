package storage

import (
	"encoding/json"
	"os"
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MemStorage) SetGauge(name string, value float64) {
	m.gauges[name] = value
}

func (m *MemStorage) AddCounter(name string, value int64) {
	m.counters[name] += value
}

func (m *MemStorage) GetGauge(name string) (*float64, bool) {
	value, exists := m.gauges[name]
	return &value, exists
}

func (m *MemStorage) GetCounter(name string) (*int64, bool) {
	value, exists := m.counters[name]
	return &value, exists
}

func (m *MemStorage) Gauges() map[string]float64 {
	return m.gauges
}

func (m *MemStorage) Counters() map[string]int64 {
	return m.counters
}

func (m *MemStorage) Save(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	data := map[string]interface{}{
		"gauges":   m.gauges,
		"counters": m.counters,
	}

	jsonData, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, jsonData, 0666)
}

func (m *MemStorage) Load(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var metricsData map[string]interface{}
	if err := json.Unmarshal(data, &metricsData); err != nil {
		return err
	}

	gauges, ok := metricsData["gauges"].(map[string]interface{})
	if ok {
		for id, value := range gauges {
			if val, ok := value.(float64); ok {
				m.gauges[id] = val
			}
		}
	}

	counters, ok := metricsData["counters"].(map[string]interface{})
	if ok {
		for id, value := range counters {
			if val, ok := value.(float64); ok {
				m.counters[id] = int64(val)
			}
		}
	}

	return nil
}
