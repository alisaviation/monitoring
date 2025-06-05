package sender

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
)

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
