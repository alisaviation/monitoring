package sender

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
)

type WorkerPool struct {
	workers chan struct{}
	wg      sync.WaitGroup
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		workers: make(chan struct{}, maxWorkers),
	}
}

func (wp *WorkerPool) Submit(task func()) {
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		wp.workers <- struct{}{}
		defer func() { <-wp.workers }()
		task()
	}()
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
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
	case errors.As(err, &pqErr) && helpers.IsRetriablePostgresError(pqErr):
		return true
	case errors.As(err, &respErr) && isRetriableHTTPStatus(respErr.Response.StatusCode()):
		return true
	default:
		return false
	}
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

func (s *Sender) prepareRequest(ctx context.Context, endpoint string, data []byte, key string) (*resty.Request, error) {
	compressedData, err := s.compressData(data)
	if err != nil {
		logger.Log.Error("Error compressing data", zap.Error(err))
		return nil, err
	}
	req := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compressedData)

	if s.publicKey != nil {
		req.SetHeader("X-Content-Encrypted", "hybrid-rsa-aes")
	}
	if s.key != "" {
		hash := helpers.CalculateHash(data, key)
		req.SetHeader("HashSHA256", hash)
	}

	return req, nil
}

func (s *Sender) sendWithRetry(ctx context.Context, endpoint string, data []byte, result interface{}, key string) error {
	retryDelays := [helpers.MaxRetries]time.Duration{helpers.InitialDelay, helpers.SecondDelay, helpers.ThirdDelay}
	var lastErr error

	for attempt := 0; attempt <= helpers.MaxRetries; attempt++ {
		req, err := s.prepareRequest(ctx, endpoint, data, key)
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
