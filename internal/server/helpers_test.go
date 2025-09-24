package server

//
//import (
//	"context"
//	"errors"
//	"net/http"
//	"net/http/httptest"
//	"strings"
//	"testing"
//	"time"
//
//	"github.com/go-chi/chi/v5"
//	"github.com/stretchr/testify/mock"
//	"github.com/stretchr/testify/require"
//
//	"github.com/alisaviation/monitoring/internal/helpers"
//	"github.com/alisaviation/monitoring/internal/models"
//)
//
//type MockStorage struct {
//	mock.Mock
//}
//
//func (m *MockStorage) SetGauge(ctx context.Context, name string, value float64) error {
//	args := m.Called(ctx, name, value)
//	return args.Error(0)
//}
//
//func (m *MockStorage) AddCounter(ctx context.Context, name string, value int64) error {
//	args := m.Called(ctx, name, value)
//	return args.Error(0)
//}
//
//func (m *MockStorage) GetGauge(ctx context.Context, name string) (*float64, error) {
//	args := m.Called(ctx, name)
//	return args.Get(0).(*float64), args.Error(1)
//}
//
//func (m *MockStorage) GetCounter(ctx context.Context, name string) (*int64, error) {
//	args := m.Called(ctx, name)
//	return args.Get(0).(*int64), args.Error(1)
//}
//
//func (m *MockStorage) Gauges(ctx context.Context) (map[string]float64, error) {
//	args := m.Called(ctx)
//	return args.Get(0).(map[string]float64), args.Error(1)
//}
//
//func (m *MockStorage) Counters(ctx context.Context) (map[string]int64, error) {
//	args := m.Called(ctx)
//	return args.Get(0).(map[string]int64), args.Error(1)
//}
//
//func (m *MockStorage) Save() error {
//	args := m.Called()
//	return args.Error(0)
//}
//
//func (m *MockStorage) IsUniqueViolationError(err error) bool {
//	args := m.Called(err)
//	return args.Bool(0)
//}
//
//func (m *MockStorage) Close() error {
//	args := m.Called()
//	return args.Error(0)
//}
//
//func TestUpdateMetric(t *testing.T) {
//	mockStorage := &MockStorage{}
//	server := &Server{storage: mockStorage}
//
//	tests := []struct {
//		name          string
//		metric        models.Metric
//		setupMock     func()
//		expectedError error
//	}{
//		{
//			name: "Valid Gauge",
//			metric: models.Metric{
//				ID:    "test_gauge",
//				MType: models.Gauge,
//				Value: float64Ptr(123.45),
//			},
//			setupMock: func() {
//				mockStorage.On("SetGauge", mock.Anything, "test_gauge", 123.45).Return(nil)
//			},
//			expectedError: nil,
//		},
//		{
//			name: "Valid Counter",
//			metric: models.Metric{
//				ID:    "test_counter",
//				MType: models.Counter,
//				Delta: int64Ptr(42),
//			},
//			setupMock: func() {
//				mockStorage.On("AddCounter", mock.Anything, "test_counter", int64(42)).Return(nil)
//			},
//			expectedError: nil,
//		},
//		{
//			name: "Invalid Metric Type",
//			metric: models.Metric{
//				ID:    "invalid",
//				MType: "invalid_type",
//			},
//			setupMock:     func() {},
//			expectedError: &helpers.HTTPError{StatusCode: http.StatusBadRequest, Message: "Bad Request: invalid metric type"},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.setupMock()
//			err := server.updateMetric(context.Background(), tt.metric)
//			if tt.expectedError != nil {
//				require.EqualError(t, err, tt.expectedError.Error())
//			} else {
//				require.NoError(t, err)
//			}
//			mockStorage.AssertExpectations(t)
//		})
//	}
//}
//
//func TestValidateMetric(t *testing.T) {
//	tests := []struct {
//		name          string
//		metric        models.Metric
//		expectedError error
//	}{
//		{
//			name: "Valid Gauge",
//			metric: models.Metric{
//				MType: models.Gauge,
//				Value: float64Ptr(123.45),
//			},
//			expectedError: nil,
//		},
//		{
//			name: "Valid Counter",
//			metric: models.Metric{
//				MType: models.Counter,
//				Delta: int64Ptr(42),
//			},
//			expectedError: nil,
//		},
//		{
//			name: "Missing Gauge Value",
//			metric: models.Metric{
//				MType: models.Gauge,
//			},
//			expectedError: errors.New("bad Request: value is required for gauge"),
//		},
//		{
//			name: "Missing Counter Delta",
//			metric: models.Metric{
//				MType: models.Counter,
//			},
//			expectedError: errors.New("bad Request: delta is required for counter"),
//		},
//		{
//			name: "Invalid Metric Type",
//			metric: models.Metric{
//				MType: "invalid_type",
//			},
//			expectedError: errors.New("bad Request: invalid metric type"),
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			err := validateMetric(tt.metric)
//			if tt.expectedError != nil {
//				require.EqualError(t, err, tt.expectedError.Error())
//			} else {
//				require.NoError(t, err)
//			}
//		})
//	}
//}
//
//func TestGetUpdatedMetrics(t *testing.T) {
//	mockStorage := &MockStorage{}
//	server := &Server{storage: mockStorage}
//
//	tests := []struct {
//		name           string
//		metrics        []models.Metric
//		setupMock      func()
//		expectedResult []models.Metric
//		expectedError  error
//	}{
//		{
//			name: "Single Gauge",
//			metrics: []models.Metric{
//				{ID: "test_gauge", MType: models.Gauge},
//			},
//			setupMock: func() {
//				mockStorage.On("GetGauge", mock.Anything, "test_gauge").Return(float64Ptr(123.45), nil)
//			},
//			expectedResult: []models.Metric{
//				{ID: "test_gauge", MType: models.Gauge, Value: float64Ptr(123.45)},
//			},
//			expectedError: nil,
//		},
//		{
//			name: "Single Counter",
//			metrics: []models.Metric{
//				{ID: "test_counter", MType: models.Counter},
//			},
//			setupMock: func() {
//				mockStorage.On("GetCounter", mock.Anything, "test_counter").Return(int64Ptr(42), nil)
//			},
//			expectedResult: []models.Metric{
//				{ID: "test_counter", MType: models.Counter, Delta: int64Ptr(42)},
//			},
//			expectedError: nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.setupMock()
//			result, err := server.getUpdatedMetrics(context.Background(), tt.metrics)
//			if tt.expectedError != nil {
//				require.EqualError(t, err, tt.expectedError.Error())
//			} else {
//				require.NoError(t, err)
//				require.Equal(t, tt.expectedResult, result)
//			}
//			mockStorage.AssertExpectations(t)
//		})
//	}
//}
//
//func TestUpdateJSONMetrics(t *testing.T) {
//	mockStorage := &MockStorage{}
//	server := &Server{storage: mockStorage}
//
//	tests := []struct {
//		name         string
//		requestBody  string
//		setupMock    func()
//		expectedCode int
//		expectedBody string
//	}{
//		{
//			name:        "Valid Gauge",
//			requestBody: `{"id":"test_gauge","type":"gauge","value":123.45}`,
//			setupMock: func() {
//				mockStorage.On("SetGauge", mock.Anything, "test_gauge", 123.45).Return(nil)
//				mockStorage.On("GetGauge", mock.Anything, "test_gauge").Return(float64Ptr(123.45), nil)
//			},
//			expectedCode: http.StatusOK,
//			expectedBody: `{"id":"test_gauge","type":"gauge","value":123.45}`,
//		},
//		{
//			name:         "Invalid JSON",
//			requestBody:  `invalid json`,
//			setupMock:    func() {},
//			expectedCode: http.StatusBadRequest,
//			expectedBody: "Bad Request: invalid JSON",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.setupMock()
//			req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(tt.requestBody))
//			w := httptest.NewRecorder()
//			server.updateJSONMetrics(context.Background(), w, req, "")
//
//			require.Equal(t, tt.expectedCode, w.Code)
//			if tt.expectedBody != "" {
//				require.Contains(t, w.Body.String(), tt.expectedBody)
//			}
//			mockStorage.AssertExpectations(t)
//		})
//	}
//}
//
//func TestUpdateTextMetrics(t *testing.T) {
//	mockStorage := &MockStorage{}
//	server := &Server{storage: mockStorage}
//
//	tests := []struct {
//		name         string
//		url          string
//		setupMock    func()
//		expectedCode int
//		expectedBody string
//	}{
//		{
//			name: "Valid Gauge",
//			url:  "/update/gauge/test_gauge/123.45",
//			setupMock: func() {
//				mockStorage.On("SetGauge", mock.Anything, "test_gauge", 123.45).Return(nil)
//				mockStorage.On("GetGauge", mock.Anything, "test_gauge").Return(float64Ptr(123.45), nil)
//			},
//			expectedCode: http.StatusOK,
//			expectedBody: `{"id":"test_gauge","type":"gauge","value":123.45}`,
//		},
//		{
//			name:         "Invalid Gauge Value",
//			url:          "/update/gauge/test_gauge/invalid",
//			setupMock:    func() {},
//			expectedCode: http.StatusBadRequest,
//			expectedBody: "Bad Request: invalid gauge value",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			r := chi.NewRouter()
//			r.Post("/update/{type}/{name}/{value}", func(w http.ResponseWriter, r *http.Request) {
//				server.updateTextMetrics(w, r, "")
//			})
//
//			tt.setupMock()
//			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
//			w := httptest.NewRecorder()
//			r.ServeHTTP(w, req)
//
//			require.Equal(t, tt.expectedCode, w.Code)
//			if tt.expectedBody != "" {
//				require.Contains(t, w.Body.String(), tt.expectedBody)
//			}
//			mockStorage.AssertExpectations(t)
//		})
//	}
//}
//
//func TestHandleRetry(t *testing.T) {
//	server := &Server{}
//	ctx := context.Background()
//	retryDelays := [helpers.MaxRetries]time.Duration{10 * time.Millisecond, 20 * time.Millisecond}
//
//	t.Run("Successful retry", func(t *testing.T) {
//		start := time.Now()
//		err := server.handleRetry(ctx, 0, retryDelays, nil)
//		elapsed := time.Since(start)
//
//		require.NoError(t, err)
//		require.GreaterOrEqual(t, elapsed, 10*time.Millisecond)
//	})
//
//	t.Run("Context canceled", func(t *testing.T) {
//		cancelCtx, cancel := context.WithCancel(ctx)
//		cancel()
//
//		start := time.Now()
//		err := server.handleRetry(cancelCtx, 0, retryDelays, nil)
//		elapsed := time.Since(start)
//
//		require.Error(t, err)
//		require.Less(t, elapsed, 10*time.Millisecond)
//	})
//}
//
//func float64Ptr(f float64) *float64 {
//	return &f
//}
//
//func int64Ptr(i int64) *int64 {
//	return &i
//}
