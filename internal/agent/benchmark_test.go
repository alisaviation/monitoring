package agent

import (
	"context"
	"testing"
	"time"

	"github.com/alisaviation/monitoring/internal/config"
)

func BenchmarkAgent(b *testing.B) {
	conf := config.Agent{
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		ServerAddress:  "http://localhost:8080",
		RateLimit:      10,
	}

	b.Run("Run", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			agent := NewAgent(conf)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			go agent.Run()
			<-ctx.Done()
		}
	})

	b.Run("CollectAndSend", func(b *testing.B) {
		agent := NewAgent(conf)
		ctx := context.Background()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			metrics := agent.collector.CollectMetrics()
			_ = agent.sender.SendMetricsBatch(ctx, metrics, "")
		}
	})
}
