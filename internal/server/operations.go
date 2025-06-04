package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/models"
)

func updateMetricInTx(tx *sql.Tx, metric models.Metric) error {
	switch metric.MType {
	case models.Gauge:
		_, err := tx.Exec(`
            INSERT INTO gauges (name, value)
            VALUES ($1, $2)
            ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
        `, metric.ID, *metric.Value)
		return err
	case models.Counter:
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
	retryDelays := [helpers.MaxRetries]time.Duration{helpers.InitialDelay, helpers.SecondDelay, helpers.ThirdDelay}
	var lastErr error

	for attempt := 0; attempt <= helpers.MaxRetries; attempt++ {
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			if helpers.IsRetriablePostgresError(err) {
				lastErr = err
				if err := s.handleRetry(ctx, attempt, retryDelays, lastErr); err != nil {
					return err
				}
				continue
			}
			return fmt.Errorf("begin transaction failed: %w", err)
		}

		if err := fn(tx); err != nil {
			tx.Rollback()

			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == pgerrcode.UniqueViolation {
				return fmt.Errorf("unique violation: %w", err)
			}

			if helpers.IsRetriablePostgresError(err) {
				lastErr = err
				if err := s.handleRetry(ctx, attempt, retryDelays, lastErr); err != nil {
					return err
				}
				continue
			}
			return err
		}

		if err := tx.Commit(); err != nil {
			if helpers.IsRetriablePostgresError(err) {
				lastErr = err
				if err := s.handleRetry(ctx, attempt, retryDelays, lastErr); err != nil {
					return err
				}
				continue
			}
			return fmt.Errorf("commit failed: %w", err)
		}

		return nil
	}

	return fmt.Errorf("after %d attempts: %w", helpers.MaxRetries, lastErr)

}

func (s *Server) handleRetry(ctx context.Context, attempt int, retryDelays [helpers.MaxRetries]time.Duration, lastErr error) error {
	if attempt < helpers.MaxRetries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelays[attempt]):
			return nil
		}
	}
	return fmt.Errorf("after %d attempts: %w", helpers.MaxRetries, lastErr)
}
