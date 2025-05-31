package sender

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/models"
)

const (
	maxRetries   = 3
	initialDelay = 1 * time.Second
	secondDelay  = 3 * time.Second
	thirdDelay   = 5 * time.Second
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

func (s *Sender) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		logger.Log.Error("gzip write error: ", zap.Error(err))
		return nil, err
	}
	if err := gz.Close(); err != nil {
		logger.Log.Error("gzip close error: ", zap.Error(err))
		return nil, err
	}
	return buf.Bytes(), nil
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

func (s *Sender) isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	var (
		netErr  net.Error
		dnsErr  *net.DNSError
		pqErr   *pq.Error
		respErr *resty.ResponseError
	)

	switch {
	case errors.As(err, &netErr) && netErr.Timeout():
		return true
	case errors.As(err, &dnsErr) && dnsErr.IsTemporary:
		return true
	case errors.As(err, &pqErr) && isRetriablePqError(pqErr):
		return true
	case errors.As(err, &respErr) && isRetriableHTTPStatus(respErr.Response.StatusCode()):
		return true
	default:
		return false
	}
}

func isRetriablePqError(err *pq.Error) bool {
	switch err.Code {
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.SQLClientUnableToEstablishSQLConnection,
		pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
		pgerrcode.TransactionResolutionUnknown,
		pgerrcode.SerializationFailure:
		return true
	}
	return false
}

func isRetriableHTTPStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout
}

var (
	ErrMaxRetriesExceeded = errors.New("maximum retry attempts exceeded")
	ErrNonRetriable       = errors.New("non-retriable error occurred")
	ErrEmptyBatch         = errors.New("metrics batch is empty")
)

func (s *Sender) sendWithRetry(ctx context.Context, endpoint string, data []byte, result interface{}) error {
	retryDelays := [maxRetries]time.Duration{initialDelay, secondDelay, thirdDelay}
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
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

		if attempt < maxRetries {
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

func (s *Sender) prepareRequest(ctx context.Context, endpoint string, data []byte) (*resty.Request, error) {
	compressedData, err := s.compressData(data)
	if err != nil {
		logger.Log.Error("Error compressing data", zap.Error(err))
		return nil, err
	}

	return s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compressedData), nil
}
