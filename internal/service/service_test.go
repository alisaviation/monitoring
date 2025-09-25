package service

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/storage"
)

type MockStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	err      error
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MockStorage) SetGauge(ctx context.Context, name string, value float64) error {
	if m.err != nil {
		return m.err
	}
	m.gauges[name] = value
	return nil
}

func (m *MockStorage) GetGauge(ctx context.Context, name string) (*float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	if value, exists := m.gauges[name]; exists {
		return &value, nil
	}
	return nil, errors.New("not found")
}

func (m *MockStorage) AddCounter(ctx context.Context, name string, delta int64) error {
	if m.err != nil {
		return m.err
	}
	m.counters[name] += delta
	return nil
}

func (m *MockStorage) GetCounter(ctx context.Context, name string) (*int64, error) {
	if m.err != nil {
		return nil, m.err
	}
	if value, exists := m.counters[name]; exists {
		return &value, nil
	}
	return nil, errors.New("not found")
}

func (m *MockStorage) Gauges(ctx context.Context) (map[string]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.gauges, nil
}

func (m *MockStorage) Counters(ctx context.Context) (map[string]int64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.counters, nil
}

func (m *MockStorage) Save() error {
	return nil
}

func (m *MockStorage) IsUniqueViolationError(err error) bool {
	return false
}

func (m *MockStorage) SetError(err error) {
	m.err = err
}

type MockDB struct {
	shouldFailBegin  bool
	shouldFailCommit bool
}

func NewMockDB() *sql.DB {
	return nil
}

func (m *MockDB) PingContext(ctx context.Context) error {
	return nil
}

type fields struct {
	storage storage.Storage
	db      *sql.DB
}

type args struct {
	storage storage.Storage
	db      *sql.DB
}

func TestNewMetricsService(t *testing.T) {
	mockStorage := NewMockStorage()
	mockDB := NewMockDB()

	tests := []struct {
		name string
		args args
		want *MetricsService
	}{
		{
			name: "create service with storage and db",
			args: args{
				storage: mockStorage,
				db:      mockDB,
			},
			want: &MetricsService{
				storage: mockStorage,
				db:      mockDB,
			},
		},
		{
			name: "create service with storage only",
			args: args{
				storage: mockStorage,
				db:      nil,
			},
			want: &MetricsService{
				storage: mockStorage,
				db:      nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMetricsService(tt.args.storage, tt.args.db); !reflect.DeepEqual(got.storage, tt.want.storage) || got.db != tt.want.db {
				t.Errorf("NewMetricsService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetricsService_UpdateMetric(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx    context.Context
		metric models.Metric
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "update gauge metric successfully",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args: args{
				ctx: ctx,
				metric: models.Metric{
					ID:    "test_gauge",
					MType: models.Gauge,
					Value: func() *float64 { v := 10.5; return &v }(),
				},
			},
			wantErr: false,
		},
		{
			name: "update counter metric successfully",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args: args{
				ctx: ctx,
				metric: models.Metric{
					ID:    "test_counter",
					MType: models.Counter,
					Delta: func() *int64 { v := int64(10); return &v }(),
				},
			},
			wantErr: false,
		},
		{
			name: "fail when gauge value is nil",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args: args{
				ctx: ctx,
				metric: models.Metric{
					ID:    "test_gauge",
					MType: models.Gauge,
					Value: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "fail when counter delta is nil",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args: args{
				ctx: ctx,
				metric: models.Metric{
					ID:    "test_counter",
					MType: models.Counter,
					Delta: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "fail when metric type is unknown",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args: args{
				ctx: ctx,
				metric: models.Metric{
					ID:    "test_unknown",
					MType: "unknown",
				},
			},
			wantErr: true,
		},
		{
			name: "fail when storage returns error",
			fields: fields{
				storage: func() storage.Storage {
					mock := NewMockStorage()
					mock.SetError(errors.New("storage error"))
					return mock
				}(),
				db: nil,
			},
			args: args{
				ctx: ctx,
				metric: models.Metric{
					ID:    "test_gauge",
					MType: models.Gauge,
					Value: func() *float64 { v := 10.5; return &v }(),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsService{
				storage: tt.fields.storage,
				db:      tt.fields.db,
			}
			if err := s.UpdateMetric(tt.args.ctx, tt.args.metric); (err != nil) != tt.wantErr {
				t.Errorf("UpdateMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetricsService_GetMetric(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx        context.Context
		metricType string
		metricName string
	}

	mockStorage := NewMockStorage()
	mockStorage.gauges["existing_gauge"] = 15.7
	mockStorage.counters["existing_counter"] = 42

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *models.Metric
		wantErr bool
	}{
		{
			name: "get existing gauge metric",
			fields: fields{
				storage: mockStorage,
				db:      nil,
			},
			args: args{
				ctx:        ctx,
				metricType: models.Gauge,
				metricName: "existing_gauge",
			},
			want: &models.Metric{
				ID:    "existing_gauge",
				MType: models.Gauge,
				Value: func() *float64 { v := 15.7; return &v }(),
			},
			wantErr: false,
		},
		{
			name: "get existing counter metric",
			fields: fields{
				storage: mockStorage,
				db:      nil,
			},
			args: args{
				ctx:        ctx,
				metricType: models.Counter,
				metricName: "existing_counter",
			},
			want: &models.Metric{
				ID:    "existing_counter",
				MType: models.Counter,
				Delta: func() *int64 { v := int64(42); return &v }(),
			},
			wantErr: false,
		},
		{
			name: "fail when metric type is unknown",
			fields: fields{
				storage: mockStorage,
				db:      nil,
			},
			args: args{
				ctx:        ctx,
				metricType: "unknown",
				metricName: "test",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "fail when gauge not found",
			fields: fields{
				storage: mockStorage,
				db:      nil,
			},
			args: args{
				ctx:        ctx,
				metricType: models.Gauge,
				metricName: "non_existing_gauge",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "fail when counter not found",
			fields: fields{
				storage: mockStorage,
				db:      nil,
			},
			args: args{
				ctx:        ctx,
				metricType: models.Counter,
				metricName: "non_existing_counter",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsService{
				storage: tt.fields.storage,
				db:      tt.fields.db,
			}
			got, err := s.GetMetric(tt.args.ctx, tt.args.metricType, tt.args.metricName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil {
				if got == nil {
					t.Errorf("GetMetric() got = nil, want %v", tt.want)
					return
				}
				if got.ID != tt.want.ID || got.MType != tt.want.MType {
					t.Errorf("GetMetric() got = %v, want %v", got, tt.want)
				}
				if tt.want.Value != nil && (got.Value == nil || *got.Value != *tt.want.Value) {
					t.Errorf("GetMetric() Value got = %v, want %v", got.Value, tt.want.Value)
				}
				if tt.want.Delta != nil && (got.Delta == nil || *got.Delta != *tt.want.Delta) {
					t.Errorf("GetMetric() Delta got = %v, want %v", got.Delta, tt.want.Delta)
				}
			} else if got != nil {
				t.Errorf("GetMetric() got = %v, want nil", got)
			}
		})
	}
}

func TestMetricsService_GetAllMetrics(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []models.Metric
		wantErr bool
	}{
		{
			name: "get all metrics successfully",
			fields: fields{
				storage: func() storage.Storage {
					mock := NewMockStorage()
					mock.gauges["gauge1"] = 1.1
					mock.gauges["gauge2"] = 2.2
					mock.counters["counter1"] = 10
					mock.counters["counter2"] = 20
					return mock
				}(),
				db: nil,
			},
			args: args{ctx: ctx},
			want: []models.Metric{
				{ID: "gauge1", MType: models.Gauge, Value: func() *float64 { v := 1.1; return &v }()},
				{ID: "gauge2", MType: models.Gauge, Value: func() *float64 { v := 2.2; return &v }()},
				{ID: "counter1", MType: models.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
				{ID: "counter2", MType: models.Counter, Delta: func() *int64 { v := int64(20); return &v }()},
			},
			wantErr: false,
		},
		{
			name: "get empty metrics",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args:    args{ctx: ctx},
			want:    []models.Metric{},
			wantErr: false,
		},
		{
			name: "fail when storage returns error for gauges",
			fields: fields{
				storage: func() storage.Storage {
					mock := NewMockStorage()
					mock.SetError(errors.New("storage error"))
					return mock
				}(),
				db: nil,
			},
			args:    args{ctx: ctx},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsService{
				storage: tt.fields.storage,
				db:      tt.fields.db,
			}
			got, err := s.GetAllMetrics(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("GetAllMetrics() got %d metrics, want %d", len(got), len(tt.want))
				return
			}

			for _, expectedMetric := range tt.want {
				found := false
				for _, actualMetric := range got {
					if actualMetric.ID == expectedMetric.ID && actualMetric.MType == expectedMetric.MType {
						found = true
						if expectedMetric.Value != nil && (actualMetric.Value == nil || *actualMetric.Value != *expectedMetric.Value) {
							t.Errorf("GetAllMetrics() metric %s value mismatch: got %v, want %v",
								actualMetric.ID, actualMetric.Value, expectedMetric.Value)
						}
						if expectedMetric.Delta != nil && (actualMetric.Delta == nil || *actualMetric.Delta != *expectedMetric.Delta) {
							t.Errorf("GetAllMetrics() metric %s delta mismatch: got %v, want %v",
								actualMetric.ID, actualMetric.Delta, expectedMetric.Delta)
						}
						break
					}
				}
				if !found {
					t.Errorf("GetAllMetrics() metric %s not found", expectedMetric.ID)
				}
			}
		})
	}
}

func TestMetricsService_UpdateMetricsBatch(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx     context.Context
		metrics []models.Metric
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "update batch in memory successfully",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args: args{
				ctx: ctx,
				metrics: []models.Metric{
					{ID: "gauge1", MType: models.Gauge, Value: func() *float64 { v := 1.1; return &v }()},
					{ID: "counter1", MType: models.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
				},
			},
			wantErr: false,
		},
		{
			name: "fail when batch is empty",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args: args{
				ctx:     ctx,
				metrics: []models.Metric{},
			},
			wantErr: true,
		},
		{
			name: "fail when metric in batch is invalid",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args: args{
				ctx: ctx,
				metrics: []models.Metric{
					{ID: "gauge1", MType: models.Gauge, Value: nil},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsService{
				storage: tt.fields.storage,
				db:      tt.fields.db,
			}
			if err := s.UpdateMetricsBatch(tt.args.ctx, tt.args.metrics); (err != nil) != tt.wantErr {
				t.Errorf("UpdateMetricsBatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetricsService_Ping(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "ping without db",
			fields: fields{
				storage: NewMockStorage(),
				db:      nil,
			},
			args:    args{ctx: ctx},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsService{
				storage: tt.fields.storage,
				db:      tt.fields.db,
			}
			if err := s.Ping(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetricsService_updateMetricsBatchInMemory(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx     context.Context
		metrics []models.Metric
	}

	mockStorage := NewMockStorage()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "update batch in memory successfully",
			fields: fields{
				storage: mockStorage,
				db:      nil,
			},
			args: args{
				ctx: ctx,
				metrics: []models.Metric{
					{ID: "gauge1", MType: models.Gauge, Value: func() *float64 { v := 1.1; return &v }()},
					{ID: "counter1", MType: models.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
				},
			},
			wantErr: false,
		},
		{
			name: "fail when storage error occurs",
			fields: fields{
				storage: func() storage.Storage {
					mock := NewMockStorage()
					mock.SetError(errors.New("storage error"))
					return mock
				}(),
				db: nil,
			},
			args: args{
				ctx: ctx,
				metrics: []models.Metric{
					{ID: "gauge1", MType: models.Gauge, Value: func() *float64 { v := 1.1; return &v }()},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsService{
				storage: tt.fields.storage,
				db:      tt.fields.db,
			}
			if err := s.updateMetricsBatchInMemory(tt.args.ctx, tt.args.metrics); (err != nil) != tt.wantErr {
				t.Errorf("updateMetricsBatchInMemory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
