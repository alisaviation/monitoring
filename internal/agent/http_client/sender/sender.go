// Package sender implements metrics sending functionality to the monitoring server.
package sender

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/models"
)

// Sender handles sending metrics to the monitoring server.
type Sender struct {
	serverAddress string
	client        *resty.Client
	key           string
	publicKey     *rsa.PublicKey
}

// NewSender creates a new Sender instance with the given server address and key.
func NewSender(serverAddress string, key string, publicKey *rsa.PublicKey) *Sender {
	client := resty.New()
	client.SetHeader("Accept-Encoding", "gzip")

	localIP := getLocalIP()
	if localIP != "" {
		client.SetHeader("X-Real-IP", localIP)
	}

	return &Sender{
		serverAddress: serverAddress,
		client:        client,
		key:           key,
		publicKey:     publicKey,
	}
}

// SendMetricsBatch sends a batch of metrics to the server in a single request.
// ctx is used for request cancellation.
// metrics contains the metrics to send.
// key is used for request signing.
// Returns an error if the send operation fails.
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

	if s.publicKey != nil {
		jsonData, err = helpers.EncryptData(jsonData, s.publicKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}
	}

	if err := s.sendWithRetry(ctx, "/updates/", jsonData, nil, key); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}
	return nil
}
