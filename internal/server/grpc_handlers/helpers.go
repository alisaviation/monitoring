package grpc_handlers

import (
	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/proto/brief/rpc"
)

func (h *GRPCHandlers) protoToModel(protoMetric *rpc.Metric) models.Metric {
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

func (h *GRPCHandlers) modelToProto(metric models.Metric) *rpc.Metric {
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

func (h *GRPCHandlers) isValidMetric(metric models.Metric) bool {
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
