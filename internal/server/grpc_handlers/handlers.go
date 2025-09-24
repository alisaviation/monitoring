// Package grpc_handlers implements gRPC server handlers for the monitoring service.
package grpc_handlers

import (
	"context"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/alisaviation/monitoring/proto/brief/rpc"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/service"
)

// GRPCHandlers implements the gRPC service interface for metric operations.
type GRPCHandlers struct {
	rpc.UnimplementedMonitoringServiceServer
	metricsService *service.MetricsService
	key            string
	privateKey     interface{}
}

// NewGRPCHandlers creates a new instance of GRPCHandlers with the required dependencies.
func NewGRPCHandlers(metricsService *service.MetricsService, key string, privateKey interface{}) *GRPCHandlers {
	return &GRPCHandlers{
		metricsService: metricsService,
		key:            key,
		privateKey:     privateKey,
	}
}

// Update handles batch metric updates via unary RPC call.
func (h *GRPCHandlers) Update(ctx context.Context, req *rpc.UpdateRequest) (*rpc.UpdateResponse, error) {
	//var metricsToProcess []*rpc.Metric
	//
	//if len(req.EncryptedData) > 0 {
	//	metricsToProcess = req.Metrics
	//} else if len(req.Metrics) > 0 {
	//	metricsToProcess = req.Metrics
	//} else {
	//	return &rpc.UpdateResponse{
	//		Status: "error",
	//		Error:  "no metrics provided",
	//	}, status.Error(codes.InvalidArgument, "no metrics provided")
	//}

	//if len(metricsToProcess) == 0 {
	if len(req.Metrics) == 0 {
		return &rpc.UpdateResponse{
			Status: "error",
			Error:  "no metrics provided",
		}, status.Error(codes.InvalidArgument, "no metrics provided")
	}

	//metrics := make([]models.Metric, 0, len(metricsToProcess))
	//for _, protoMetric := range metricsToProcess {
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

// Value retrieves a specific metric by type and name.
func (h *GRPCHandlers) Value(ctx context.Context, req *rpc.ValueRequest) (*rpc.ValueResponse, error) {
	//var metricType, metricName string
	//
	//if len(req.EncryptedData) > 0 {
	//	metricType = req.MetricType
	//	metricName = req.MetricName
	//} else {
	//	metricType = req.MetricType
	//	metricName = req.MetricName
	//}

	//metric, err := h.metricsService.GetMetric(ctx, metricType, metricName)
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

// Ping performs a health check on the underlying storage.
func (h *GRPCHandlers) Ping(ctx context.Context, req *rpc.PingRequest) (*rpc.PingResponse, error) {
	if err := h.metricsService.Ping(ctx); err != nil {
		return &rpc.PingResponse{
			Success: false,
		}, status.Error(codes.Internal, "database unavailable")
	}

	return &rpc.PingResponse{
		Success: true,
	}, nil
}

// Updates handles streaming metric updates via bidirectional RPC.
// Accepts a stream of metrics and processes them in batch when the stream ends.
func (h *GRPCHandlers) Updates(stream rpc.MonitoringService_UpdatesServer) error {
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
