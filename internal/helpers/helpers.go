package helpers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/storage"
)

const (
	MaxRetries   = 3
	InitialDelay = 1 * time.Second
	SecondDelay  = 3 * time.Second
	ThirdDelay   = 5 * time.Second
)

func MethodCheck(methods []string) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			for _, method := range methods {
				if r.Method == method {
					next(w, r)
					return
				}
			}
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func FormatFloat(value float64) string {
	formatted := fmt.Sprintf("%.3f", value)
	return strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
}

func CheckAndSaveMetrics(ctx context.Context, storage storage.Storage, prevGauges map[string]float64, prevCounters map[string]int64) {
	currentGauges, _ := storage.Gauges(ctx)
	currentCounters, _ := storage.Counters(ctx)

	gaugeChanged := len(prevGauges) != len(currentGauges)
	if !gaugeChanged {
		for k, v := range currentGauges {
			if prevGauges[k] != v {
				gaugeChanged = true
				break
			}
		}
	}

	counterChanged := len(prevCounters) != len(currentCounters)
	if !counterChanged {
		for k, v := range currentCounters {
			if prevCounters[k] != v {
				counterChanged = true
				break
			}
		}
	}

	if gaugeChanged || counterChanged {
		if err := storage.Save(); err != nil {
			logger.Log.Error("Error saving metrics", zap.Error(err))
		}
	}
}

func IsRetriablePostgresError(err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}

	switch pqErr.Code {
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

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}
func CalculateHash(data []byte, key string) string {
	if key == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
