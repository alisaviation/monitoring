package storage

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func BenchmarkMemStorage(b *testing.B) {
	storage := NewMemStorage("")

	b.Run("SetGauge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := storage.SetGauge(context.Background(), "test_gauge", 123.45)
			require.NoError(b, err)
		}
	})

	b.Run("AddCounter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := storage.AddCounter(context.Background(), "test_counter", 1)
			require.NoError(b, err)
		}
	})

	b.Run("GetGauge", func(b *testing.B) {
		err := storage.SetGauge(context.Background(), "test_gauge", 123.45)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val, err := storage.GetGauge(context.Background(), "test_gauge")
			require.NoError(b, err)
			require.NotNil(b, val)
		}
	})

	b.Run("GetCounter", func(b *testing.B) {
		err := storage.AddCounter(context.Background(), "test_counter", 1)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val, err := storage.GetCounter(context.Background(), "test_counter")
			require.NoError(b, err)
			require.NotNil(b, val)
		}
	})

	b.Run("Gauges", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			err := storage.SetGauge(context.Background(),
				"test_gauge_"+string(rune(i)), float64(i))
			require.NoError(b, err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gauges, err := storage.Gauges(context.Background())
			require.NoError(b, err)
			require.Greater(b, len(gauges), 0)
		}
	})

	b.Run("Counters", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			err := storage.AddCounter(context.Background(),
				"test_counter_"+string(rune(i)), int64(i))
			require.NoError(b, err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			counters, err := storage.Counters(context.Background())
			require.NoError(b, err)
			require.Greater(b, len(counters), 0)
		}
	})
}

func BenchmarkPostgresStorage(b *testing.B) {
	ctx := context.Background()
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS gauges").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS counters").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	storage, err := NewPostgresStorageFromDB(ctx, db)
	require.NoError(b, err)

	b.Run("SetGauge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mock.ExpectExec(`^INSERT INTO gauges \(name, value\) VALUES \(\$1, \$2\) ON CONFLICT \(name\) DO UPDATE SET value = EXCLUDED\.value$`).
				WithArgs("test_gauge", 123.45).
				WillReturnResult(sqlmock.NewResult(1, 1))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := storage.SetGauge(ctx, "test_gauge", 123.45)
			require.NoError(b, err)
		}
	})

	b.Run("AddCounter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mock.ExpectExec(`^INSERT INTO counters \(name, value\) VALUES \(\$1, \$2\) ON CONFLICT \(name\) DO UPDATE SET value = counters\.value \+ EXCLUDED\.value$`).
				WithArgs("test_counter", int64(1)).
				WillReturnResult(sqlmock.NewResult(1, 1))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := storage.AddCounter(ctx, "test_counter", 1)
			require.NoError(b, err)
		}
	})

	b.Run("GetGauge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mock.ExpectQuery(`^SELECT value FROM gauges WHERE name = \$1$`).
				WithArgs("test_gauge").
				WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(123.45))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val, err := storage.GetGauge(ctx, "test_gauge")
			require.NoError(b, err)
			require.NotNil(b, val)
		}
	})

	b.Run("GetCounter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mock.ExpectQuery(`^SELECT value FROM counters WHERE name = \$1$`).
				WithArgs("test_counter").
				WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val, err := storage.GetCounter(ctx, "test_counter")
			require.NoError(b, err)
			require.NotNil(b, val)
		}
	})

	require.NoError(b, mock.ExpectationsWereMet())
}
