package sender

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/models"
)

type Sender struct {
	serverAddress string
	client        *resty.Client
	key           string
}

func NewSender(serverAddress string, key string) *Sender {
	client := resty.New()
	client.SetHeader("Accept-Encoding", "gzip")
	return &Sender{
		serverAddress: serverAddress,
		client:        client,
		key:           key,
	}
}

func (s *Sender) SendMetricsBatch(ctx context.Context, metrics map[string]*models.Metric, key string) error {
	if len(metrics) == 0 {
		logger.Log.Warn("Error, the batch is empty")
		return ErrEmptyBatch
	}

	metricsList := make([]models.Metric, 0, len(metrics))
	for name, metric := range metrics {
		batchMetrics := models.Metric{
			ID:    name,
			MType: metric.MType,
		}
		if metric.MType == models.Gauge {
			batchMetrics.Value = metric.Value
		}
		if metric.MType == models.Counter {
			batchMetrics.Delta = metric.Delta
		}
		metricsList = append(metricsList, batchMetrics)
	}

	jsonData, err := json.Marshal(metricsList)
	if err != nil {
		logger.Log.Error("Error marshaling JSON", zap.Error(err))
		return fmt.Errorf("marshal failed: %w", err)
	}
	if err := s.sendWithRetry(ctx, "/updates/", jsonData, nil, key); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}
	return nil
}
