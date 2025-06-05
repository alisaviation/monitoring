package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
)

type PostgresStorage struct {
	DB *sql.DB
}

func NewPostgresStorageFromDB(ctx context.Context, db *sql.DB) (*PostgresStorage, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if err := createTables(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &PostgresStorage{DB: db}, nil
}

func createTables(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS gauges (
			name TEXT PRIMARY KEY,
			value DOUBLE PRECISION NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create gauges table: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS counters (
			name TEXT PRIMARY KEY,
			value BIGINT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create counters table: %w", err)
	}

	return tx.Commit()
}

func (p *PostgresStorage) SetGauge(ctx context.Context, name string, value float64) error {
	_, err := p.DB.ExecContext(ctx, `
		INSERT INTO gauges (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
	`, name, value)
	return err
}

func (p *PostgresStorage) AddCounter(ctx context.Context, name string, value int64) error {
	_, err := p.DB.ExecContext(ctx, `
		INSERT INTO counters (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
	`, name, value)
	return err
}

func (p *PostgresStorage) GetGauge(ctx context.Context, name string) (*float64, error) {
	var value float64
	err := p.DB.QueryRowContext(ctx, `
		SELECT value FROM gauges WHERE name = $1
	`, name).Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (p *PostgresStorage) GetCounter(ctx context.Context, name string) (*int64, error) {
	var value int64
	err := p.DB.QueryRowContext(ctx, `
		SELECT value FROM counters WHERE name = $1
	`, name).Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (p *PostgresStorage) Gauges(ctx context.Context) (map[string]float64, error) {
	rows, err := p.DB.QueryContext(ctx, "SELECT name, value FROM gauges")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	gauges := make(map[string]float64)
	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		gauges[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return gauges, nil
}

func (p *PostgresStorage) Counters(ctx context.Context) (map[string]int64, error) {
	rows, err := p.DB.QueryContext(ctx, "SELECT name, value FROM counters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counters := make(map[string]int64)
	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		counters[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counters, nil
}
func (p *PostgresStorage) Save() error {
	return nil
}
func (p *PostgresStorage) Close() error {
	return p.DB.Close()
}

func (s *PostgresStorage) IsUniqueViolationError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == pgerrcode.UniqueViolation {
		return true
	}
	return false
}
