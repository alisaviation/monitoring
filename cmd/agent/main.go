package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/agent/collector"
	"github.com/alisaviation/monitoring/internal/agent/sender"
	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/models"
)

func main() {
	conf := config.SetConfigAgent()

	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Log.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Log.Info("Received signal, agent is shutting down...", zap.String("signal", sig.String()))
		cancel()
	}()

	collectorInstance := collector.NewCollector()
	senderInstance := sender.NewSender(conf.ServerAddress, conf.Key)
	metricsBuffer := make(map[string]*models.Metric)

	pollTicker := time.NewTicker(conf.PollInterval)
	reportTicker := time.NewTicker(conf.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Shutting down agent...")
			if len(metricsBuffer) > 0 {
				if err := senderInstance.SendMetricsBatch(ctx, metricsBuffer, conf.Key); err != nil {
					logger.Log.Error("Failed to send final metrics batch", zap.Error(err))
				}
			}
			return

		case <-pollTicker.C:
			metrics := collectorInstance.CollectMetrics()
			collector.UpdateMetricsBuffer(metricsBuffer, metrics)
			logger.Log.Debug("Collected metrics", zap.Int("count", len(metrics)))

		case <-reportTicker.C:
			if len(metricsBuffer) > 0 {
				if err := senderInstance.SendMetricsBatch(ctx, metricsBuffer, conf.Key); err != nil {
					logger.Log.Error("Failed to send metrics batch", zap.Error(err))
					continue
				}
				logger.Log.Debug("Metrics batch sent", zap.Int("count", len(metricsBuffer)))
				metricsBuffer = make(map[string]*models.Metric)
			}
		}
	}
}
