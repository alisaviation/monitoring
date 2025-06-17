package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
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

	logger.Log.Info("Rate limit configuration", zap.Int("value", conf.RateLimit))

	collectorInstance := collector.NewCollector()
	senderInstance := sender.NewSender(conf.ServerAddress, conf.Key)
	workerPool := sender.NewWorkerPool(conf.RateLimit)
	bufferSize := conf.RateLimit * 10
	metricsChan := make(chan map[string]*models.Metric, bufferSize)

	var bufferMutex sync.Mutex
	metricsBuffer := make(map[string]*models.Metric)

	go func() {
		logger.Log.Info("Goroutine started: metrics collection")

		pollTicker := time.NewTicker(conf.PollInterval)
		defer pollTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				metrics := collectorInstance.CollectMetrics()
				select {
				case metricsChan <- metrics:
					logger.Log.Info("Collected runtime metrics", zap.Int("count", len(metrics)))
				default:
					logger.Log.Info("Metrics channel is full, dropping metrics batch")
				}
			}
		}
	}()

	go func() {
		logger.Log.Info("Goroutine started: gopsutil metrics collection")
		pollTicker := time.NewTicker(conf.PollInterval)
		defer pollTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				metrics := collectorInstance.СollectGopsutilMetrics()
				select {
				case metricsChan <- metrics:
					logger.Log.Info("Collected gopsutil metrics", zap.Int("count", len(metrics)))
				default:
					logger.Log.Info("Metrics channel is full, dropping metrics batch")
				}
			}
		}
	}()

	go func() {
		logger.Log.Info("Goroutine started: Update metrics")
		for {
			select {
			case <-ctx.Done():
				return
			case metrics := <-metricsChan:
				bufferMutex.Lock()
				collector.UpdateMetricsBuffer(metricsBuffer, metrics)
				bufferMutex.Unlock()
			}
		}
	}()

	go func() {
		logger.Log.Debug("Goroutine started: Sending metrics")
		reportTicker := time.NewTicker(conf.ReportInterval)
		defer reportTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				bufferMutex.Lock()
				if len(metricsBuffer) > 0 {
					senderInstance.SendMetrics(ctx, metricsBuffer, conf.Key, workerPool)
					metricsBuffer = make(map[string]*models.Metric)
				}
				bufferMutex.Unlock()
				return

			case <-reportTicker.C:

				bufferMutex.Lock()
				if len(metricsBuffer) > 0 {
					senderInstance.SendMetrics(ctx, metricsBuffer, conf.Key, workerPool)
					metricsBuffer = make(map[string]*models.Metric)
				}
				bufferMutex.Unlock()
			}
		}
	}()

	<-ctx.Done()
	workerPool.Wait()
	logger.Log.Info("Shutting down agent...")
}
