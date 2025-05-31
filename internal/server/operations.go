package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"

	"github.com/alisaviation/monitoring/internal/models"
)

const (
	maxRetries   = 3
	initialDelay = 1 * time.Second
	secondDelay  = 3 * time.Second
	thirdDelay   = 5 * time.Second
)

func (s *Server) isRetriableDBError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
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
	}
	return false
}

func updateMetricInTx(tx *sql.Tx, metric models.Metric) error {
	switch metric.MType {
	case models.Gauge:

		if metric.Value == nil {
			return fmt.Errorf("value is required for gauge")
		}
		_, err := tx.Exec(`
            INSERT INTO gauges (name, value)
            VALUES ($1, $2)
            ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
        `, metric.ID, *metric.Value)
		return err
	case models.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("delta is required for counter")
		}
		_, err := tx.Exec(`
            INSERT INTO counters (name, value)
            VALUES ($1, $2)
            ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
        `, metric.ID, *metric.Delta)
		return err
	default:
		return fmt.Errorf("invalid metric type")
	}
}

func (s *Server) execInTransactionWithRetry(ctx context.Context, fn func(tx *sql.Tx) error) error {
	retryDelays := [maxRetries]time.Duration{initialDelay, secondDelay, thirdDelay}
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			if s.isRetriableDBError(err) {
				lastErr = err
				goto retry
			}
			return fmt.Errorf("begin transaction failed: %w", err)
		}

		if err := fn(tx); err != nil {
			tx.Rollback()

			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == pgerrcode.UniqueViolation {
				return fmt.Errorf("unique violation: %w", err)
			}

			if s.isRetriableDBError(err) {
				lastErr = err
				goto retry
			}
			return err
		}

		if err := tx.Commit(); err != nil {
			if s.isRetriableDBError(err) {
				lastErr = err
				goto retry
			}
			return fmt.Errorf("commit failed: %w", err)
		}

		return nil

	retry:
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelays[attempt]):
				continue
			}
		}
	}

	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}
