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
	var err error

	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return fmt.Errorf("gauge value is required")
		}
		_, err = tx.ExecContext(context.Background(), `
            INSERT INTO gauges (name, value)
            VALUES ($1, $2)
            ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
        `, metric.ID, *metric.Value)
	case models.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("counter delta is required")
		}
		_, err = tx.ExecContext(context.Background(), `
            INSERT INTO counters (name, value)
            VALUES ($1, $2)
            ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
        `, metric.ID, *metric.Delta)
	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}
	return err
}

func (s *Server) execInTransactionWithRetry(ctx context.Context, fn func(tx *sql.Tx) error) error {
	retryDelays := [helpers.MaxRetries]time.Duration{helpers.InitialDelay, helpers.SecondDelay, helpers.ThirdDelay}
	var lastErr error

	for attempt := 0; attempt <= helpers.MaxRetries; attempt++ {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			if helpers.IsRetriablePostgresError(err) {
				lastErr = err
				if retryErr := s.handleRetry(ctx, attempt, retryDelays, lastErr); retryErr != nil {
					return retryErr
				}
				continue
			}
			return fmt.Errorf("begin transaction failed: %w", err)
		}

		fnErr := fn(tx)
		if fnErr != nil {
			tx.Rollback()

			var pqErr *pq.Error
			if errors.As(fnErr, &pqErr) && pqErr.Code == pgerrcode.UniqueViolation {
				return fmt.Errorf("unique violation: %w", fnErr)
			}

			if helpers.IsRetriablePostgresError(fnErr) {
				lastErr = fnErr
				if retryErr := s.handleRetry(ctx, attempt, retryDelays, lastErr); retryErr != nil {
					return retryErr
				}
				continue
			}
			return fnErr
		}

		commitErr := tx.Commit()
		if commitErr != nil {
			if helpers.IsRetriablePostgresError(commitErr) {
				lastErr = commitErr
				if retryErr := s.handleRetry(ctx, attempt, retryDelays, lastErr); retryErr != nil {
					return retryErr
				}
				continue
			}
			return fmt.Errorf("commit failed: %w", commitErr)
		}

		return nil
	}

	return fmt.Errorf("after %d attempts: %w", helpers.MaxRetries, lastErr)
}
