package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	DB *sql.DB
}

func NewPostgresStorageFromDB(db *sql.DB) (*PostgresStorage, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &PostgresStorage{DB: db}, nil
}

func createTables(db *sql.DB) error {
	ctx := context.Background()
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

func (p *PostgresStorage) SetGauge(name string, value float64) error {
	ctx := context.Background()
	_, err := p.DB.ExecContext(ctx, `
		INSERT INTO gauges (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
	`, name, value)
	return err
}

func (p *PostgresStorage) AddCounter(name string, value int64) error {
	ctx := context.Background()
	_, err := p.DB.ExecContext(ctx, `
		INSERT INTO counters (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value
	`, name, value)
	return err
}

func (p *PostgresStorage) GetGauge(name string) (*float64, bool) {
	ctx := context.Background()
	var value float64
	err := p.DB.QueryRowContext(ctx, `
		SELECT value FROM gauges WHERE name = $1
	`, name).Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	return &value, true
}

func (p *PostgresStorage) GetCounter(name string) (*int64, bool) {
	ctx := context.Background()
	var value int64
	err := p.DB.QueryRowContext(ctx, `
		SELECT value FROM counters WHERE name = $1
	`, name).Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	return &value, true
}

func (p *PostgresStorage) Gauges() (map[string]float64, error) {
	ctx := context.Background()
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
	return gauges, nil
}

func (p *PostgresStorage) Counters() (map[string]int64, error) {
	ctx := context.Background()
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
	return counters, nil
}
func (p *PostgresStorage) Save() error {
	return nil
}
func (p *PostgresStorage) Close() error {
	return p.DB.Close()
}
