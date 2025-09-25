package grpchandlers

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/proto/brief/rpc"
)

// MockMetricsService
type mockMetricsService struct {
	updateMetricsBatchFunc func(ctx context.Context, metrics []models.Metric) error
	getMetricFunc          func(ctx context.Context, metricType, metricName string) (*models.Metric, error)
	pingFunc               func(ctx context.Context) error
}

func (m *mockMetricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metric) error {
	if m.updateMetricsBatchFunc != nil {
		return m.updateMetricsBatchFunc(ctx, metrics)
	}
	return nil
}

func (m *mockMetricsService) GetMetric(ctx context.Context, metricType, metricName string) (*models.Metric, error) {
	if m.getMetricFunc != nil {
		return m.getMetricFunc(ctx, metricType, metricName)
	}
	return nil, nil
}

func (m *mockMetricsService) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

// TestGRPCHandlers - временная структура для тестирования
type TestGRPCHandlers struct {
	rpc.UnimplementedMonitoringServiceServer
	metricsService *mockMetricsService
	key            string
	privateKey     interface{}
}

func (h *TestGRPCHandlers) Ping(ctx context.Context, req *rpc.PingRequest) (*rpc.PingResponse, error) {
	if err := h.metricsService.Ping(ctx); err != nil {
		return &rpc.PingResponse{
			Success: false,
		}, status.Error(codes.Internal, "database unavailable")
	}

	return &rpc.PingResponse{
		Success: true,
	}, nil
}

func (h *TestGRPCHandlers) Update(ctx context.Context, req *rpc.UpdateRequest) (*rpc.UpdateResponse, error) {
	if len(req.Metrics) == 0 {
		return &rpc.UpdateResponse{
			Status: "error",
			Error:  "no metrics provided",
		}, status.Error(codes.InvalidArgument, "no metrics provided")
	}

	metrics := make([]models.Metric, 0, len(req.Metrics))
	for _, protoMetric := range req.Metrics {
		metric := h.protoToModel(protoMetric)
		metrics = append(metrics, metric)
	}

	if err := h.metricsService.UpdateMetricsBatch(ctx, metrics); err != nil {
		return &rpc.UpdateResponse{
			Status: "error",
			Error:  err.Error(),
		}, status.Error(codes.Internal, err.Error())
	}

	return &rpc.UpdateResponse{
		Status: "success",
	}, nil
}

func (h *TestGRPCHandlers) Value(ctx context.Context, req *rpc.ValueRequest) (*rpc.ValueResponse, error) {
	metric, err := h.metricsService.GetMetric(ctx, req.MetricType, req.MetricName)
	if err != nil {
		return &rpc.ValueResponse{
			Error: "metric not found",
		}, status.Error(codes.NotFound, "metric not found")
	}

	protoMetric := h.modelToProto(*metric)
	return &rpc.ValueResponse{
		Metric: protoMetric,
	}, nil
}

func (h *TestGRPCHandlers) Updates(stream rpc.MonitoringService_UpdatesServer) error {
	var metrics []models.Metric

	for {
		metric, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}

		modelMetric := h.protoToModel(metric)
		if !h.isValidMetric(modelMetric) {
			return status.Error(codes.InvalidArgument, "invalid metric data in stream")
		}
		metrics = append(metrics, modelMetric)
	}

	if err := h.metricsService.UpdateMetricsBatch(stream.Context(), metrics); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return stream.SendAndClose(&rpc.UpdateResponse{
		Status: "success",
	})
}

func (h *TestGRPCHandlers) protoToModel(protoMetric *rpc.Metric) models.Metric {
	metric := models.Metric{
		ID:    protoMetric.Id,
		MType: protoMetric.Mtype,
		Hash:  protoMetric.Hash,
	}

	if protoMetric.Delta != 0 || protoMetric.Mtype == models.Counter {
		delta := protoMetric.Delta
		metric.Delta = &delta
	}

	if protoMetric.Value != 0 || protoMetric.Mtype == models.Gauge {
		value := protoMetric.Value
		metric.Value = &value
	}

	return metric
}

func (h *TestGRPCHandlers) modelToProto(metric models.Metric) *rpc.Metric {
	protoMetric := &rpc.Metric{
		Id:    metric.ID,
		Mtype: metric.MType,
		Hash:  metric.Hash,
	}

	if metric.Delta != nil {
		protoMetric.Delta = *metric.Delta
	}

	if metric.Value != nil {
		protoMetric.Value = *metric.Value
	}

	return protoMetric
}

func (h *TestGRPCHandlers) isValidMetric(metric models.Metric) bool {
	if metric.ID == "" || metric.MType == "" {
		return false
	}

	switch metric.MType {
	case models.Gauge:
		return metric.Value != nil
	case models.Counter:
		return metric.Delta != nil
	default:
		return false
	}
}

type mockUpdatesServer struct {
	recvFunc         func() (*rpc.Metric, error)
	sendAndCloseFunc func(*rpc.UpdateResponse) error
	contextFunc      func() context.Context
}

func (m *mockUpdatesServer) Recv() (*rpc.Metric, error) {
	if m.recvFunc != nil {
		return m.recvFunc()
	}
	return nil, nil
}

func (m *mockUpdatesServer) SendAndClose(resp *rpc.UpdateResponse) error {
	if m.sendAndCloseFunc != nil {
		return m.sendAndCloseFunc(resp)
	}
	return nil
}

func (m *mockUpdatesServer) Context() context.Context {
	if m.contextFunc != nil {
		return m.contextFunc()
	}
	return context.Background()
}

func (m *mockUpdatesServer) Send(*rpc.Metric) error {
	return nil
}

func (m *mockUpdatesServer) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockUpdatesServer) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockUpdatesServer) SetTrailer(md metadata.MD) {
}
func (m *mockUpdatesServer) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockUpdatesServer) RecvMsg(msg interface{}) error {
	return nil
}

func TestGRPCHandlers_Ping(t *testing.T) {
	type args struct {
		ctx context.Context
		req *rpc.PingRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *rpc.PingResponse
		wantErr bool
	}{
		{
			name: "successful ping",
			args: args{
				ctx: context.Background(),
				req: &rpc.PingRequest{},
			},
			want: &rpc.PingResponse{
				Success: true,
			},
			wantErr: false,
		},
		{
			name: "ping with database error",
			args: args{
				ctx: context.Background(),
				req: &rpc.PingRequest{},
			},
			want: &rpc.PingResponse{
				Success: false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockMetricsService{}

			if tt.name == "ping with database error" {
				mockService.pingFunc = func(ctx context.Context) error {
					return errors.New("database unavailable")
				}
			}

			h := &TestGRPCHandlers{
				metricsService: mockService,
			}
			got, err := h.Ping(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Ping() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGRPCHandlers_Update(t *testing.T) {
	type args struct {
		ctx context.Context
		req *rpc.UpdateRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *rpc.UpdateResponse
		wantErr bool
	}{
		{
			name: "successful update with gauge metric",
			args: args{
				ctx: context.Background(),
				req: &rpc.UpdateRequest{
					Metrics: []*rpc.Metric{
						{
							Id:    "test1",
							Mtype: "gauge",
							Value: 25.5,
						},
					},
				},
			},
			want: &rpc.UpdateResponse{
				Status: "success",
			},
			wantErr: false,
		},
		{
			name: "successful update with counter metric",
			args: args{
				ctx: context.Background(),
				req: &rpc.UpdateRequest{
					Metrics: []*rpc.Metric{
						{
							Id:    "test2",
							Mtype: "counter",
							Delta: 10,
						},
					},
				},
			},
			want: &rpc.UpdateResponse{
				Status: "success",
			},
			wantErr: false,
		},
		{
			name: "update with empty metrics",
			args: args{
				ctx: context.Background(),
				req: &rpc.UpdateRequest{
					Metrics: []*rpc.Metric{},
				},
			},
			want: &rpc.UpdateResponse{
				Status: "error",
				Error:  "no metrics provided",
			},
			wantErr: true,
		},
		{
			name: "update with nil metrics",
			args: args{
				ctx: context.Background(),
				req: &rpc.UpdateRequest{},
			},
			want: &rpc.UpdateResponse{
				Status: "error",
				Error:  "no metrics provided",
			},
			wantErr: true,
		},
		{
			name: "update with service error",
			args: args{
				ctx: context.Background(),
				req: &rpc.UpdateRequest{
					Metrics: []*rpc.Metric{
						{
							Id:    "test1",
							Mtype: "gauge",
							Value: 25.5,
						},
					},
				},
			},
			want: &rpc.UpdateResponse{
				Status: "error",
				Error:  "storage error",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockMetricsService{}

			if tt.name == "update with service error" {
				mockService.updateMetricsBatchFunc = func(ctx context.Context, metrics []models.Metric) error {
					return errors.New("storage error")
				}
			} else if tt.name == "successful update with gauge metric" || tt.name == "successful update with counter metric" {
				mockService.updateMetricsBatchFunc = func(ctx context.Context, metrics []models.Metric) error {
					if len(metrics) == 0 {
						return errors.New("expected metrics")
					}
					return nil
				}
			}

			h := &TestGRPCHandlers{
				metricsService: mockService,
			}
			got, err := h.Update(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Update() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGRPCHandlers_Updates(t *testing.T) {
	type args struct {
		stream rpc.MonitoringService_UpdatesServer
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "successful stream updates",
			wantErr: false,
		},
		{
			name:    "stream with recv error",
			wantErr: true,
		},
		{
			name:    "stream with invalid metric",
			wantErr: true,
		},
		{
			name:    "stream with batch update error",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockMetricsService{}
			mockStream := &mockUpdatesServer{}

			switch tt.name {
			case "successful stream updates":
				counter := 0
				mockStream.recvFunc = func() (*rpc.Metric, error) {
					counter++
					if counter > 2 {
						return nil, io.EOF
					}
					return &rpc.Metric{
						Id:    "test1",
						Mtype: "gauge",
						Value: 25.5,
					}, nil
				}
				mockService.updateMetricsBatchFunc = func(ctx context.Context, metrics []models.Metric) error {
					if len(metrics) != 2 {
						return errors.New("expected 2 metrics")
					}
					return nil
				}

			case "stream with recv error":
				mockStream.recvFunc = func() (*rpc.Metric, error) {
					return nil, errors.New("recv error")
				}

			case "stream with invalid metric":
				counter := 0
				mockStream.recvFunc = func() (*rpc.Metric, error) {
					counter++
					if counter > 1 {
						return nil, io.EOF
					}
					return &rpc.Metric{
						Id:    "test1",
						Mtype: "",
						Value: 25.5,
					}, nil
				}

			case "stream with batch update error":
				counter := 0
				mockStream.recvFunc = func() (*rpc.Metric, error) {
					counter++
					if counter > 1 {
						return nil, io.EOF
					}
					return &rpc.Metric{
						Id:    "test1",
						Mtype: "gauge",
						Value: 25.5,
					}, nil
				}
				mockService.updateMetricsBatchFunc = func(ctx context.Context, metrics []models.Metric) error {
					return errors.New("batch update error")
				}
			}

			h := &TestGRPCHandlers{
				metricsService: mockService,
			}
			if err := h.Updates(mockStream); (err != nil) != tt.wantErr {
				t.Errorf("Updates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGRPCHandlers_Value(t *testing.T) {
	type args struct {
		ctx context.Context
		req *rpc.ValueRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *rpc.ValueResponse
		wantErr bool
	}{
		{
			name: "successful get gauge value",
			args: args{
				ctx: context.Background(),
				req: &rpc.ValueRequest{
					MetricType: "gauge",
					MetricName: "temperature",
				},
			},
			want: &rpc.ValueResponse{
				Metric: &rpc.Metric{
					Id:    "test1",
					Mtype: "gauge",
					Value: 25.5,
				},
			},
			wantErr: false,
		},
		{
			name: "successful get counter value",
			args: args{
				ctx: context.Background(),
				req: &rpc.ValueRequest{
					MetricType: "counter",
					MetricName: "requests",
				},
			},
			want: &rpc.ValueResponse{
				Metric: &rpc.Metric{
					Id:    "test2",
					Mtype: "counter",
					Delta: 100,
				},
			},
			wantErr: false,
		},
		{
			name: "get value not found",
			args: args{
				ctx: context.Background(),
				req: &rpc.ValueRequest{
					MetricType: "gauge",
					MetricName: "nonexistent",
				},
			},
			want: &rpc.ValueResponse{
				Error: "metric not found",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockMetricsService{}

			switch tt.name {
			case "successful get gauge value":
				mockService.getMetricFunc = func(ctx context.Context, metricType, metricName string) (*models.Metric, error) {
					value := 25.5
					return &models.Metric{
						ID:    "test1",
						MType: "gauge",
						Value: &value,
					}, nil
				}
			case "successful get counter value":
				mockService.getMetricFunc = func(ctx context.Context, metricType, metricName string) (*models.Metric, error) {
					delta := int64(100)
					return &models.Metric{
						ID:    "test2",
						MType: "counter",
						Delta: &delta,
					}, nil
				}
			case "get value not found":
				mockService.getMetricFunc = func(ctx context.Context, metricType, metricName string) (*models.Metric, error) {
					return nil, errors.New("metric not found")
				}
			}

			h := &TestGRPCHandlers{
				metricsService: mockService,
			}
			got, err := h.Value(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Value() got = %v, want %v", got, tt.want)
			}
		})
	}
}
