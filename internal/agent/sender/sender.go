package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/models"
)

type Sender struct {
	serverAddress string
	client        *resty.Client
}

func NewSender(serverAddress string) *Sender {
	client := resty.New()
	client.SetHeader("Accept-Encoding", "gzip")
	return &Sender{
		serverAddress: serverAddress,
		client:        client,
	}
}

func (s *Sender) SendMetricsBatch(ctx context.Context, metrics map[string]*models.Metric) error {
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
	if err := s.sendWithRetry(ctx, "/updates/", jsonData, nil); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}
	return nil
}

func (s *Sender) sendWithRetry(ctx context.Context, endpoint string, data []byte, result interface{}) error {
	retryDelays := [helpers.MaxRetries]time.Duration{helpers.InitialDelay, helpers.SecondDelay, helpers.ThirdDelay}
	var lastErr error

	for attempt := 0; attempt <= helpers.MaxRetries; attempt++ {
		req, err := s.prepareRequest(ctx, endpoint, data)
		if err != nil {
			logger.Log.Error("Error preparing request", zap.Error(err))
			return err
		}

		if result != nil {
			req.SetResult(result)
		}

		resp, err := req.Post("http://" + s.serverAddress + endpoint)
		if resp != nil {
			defer func() {
				if resp.RawResponse != nil && resp.RawResponse.Body != nil {
					resp.RawResponse.Body.Close()
				}
			}()

			if resp.StatusCode() == http.StatusOK {
				return nil
			}

			if !s.isRetriableError(err) {
				logger.Log.Error("Non-retriable error response",
					zap.String("status", resp.Status()),
					zap.Int("code", resp.StatusCode()))
				return fmt.Errorf("server returned status %d", resp.StatusCode())
			}

			lastErr = fmt.Errorf("HTTP status %d", resp.StatusCode())
			logger.Log.Warn("Retriable error response",
				zap.Int("attempt", attempt+1),
				zap.String("status", resp.Status()),
				zap.Error(lastErr))
		}

		if err != nil {
			if !s.isRetriableError(err) {
				logger.Log.Error("Non-retriable request error", zap.Error(err))
				return fmt.Errorf("%w: %v", ErrNonRetriable, err)
			}
			lastErr = err
			logger.Log.Warn("Retriable request error",
				zap.Int("attempt", attempt+1),
				zap.Error(err))
		}

		if attempt < helpers.MaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelays[attempt]):
				continue
			}
		}
	}

	logger.Log.Error("Max retries exceeded", zap.Error(lastErr))
	return fmt.Errorf("%w: last error: %v", ErrMaxRetriesExceeded, lastErr)
}
