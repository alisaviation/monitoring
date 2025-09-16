// Package agent implements the metrics collection and reporting functionality.
//
// The Agent collects system metrics at regular intervals and sends them to a monitoring server.
package agent

import (
	"context"
	"crypto/rsa"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/agent/collector"
	"github.com/alisaviation/monitoring/internal/agent/sender"
	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/models"
)

// Agent represents the metrics collection agent.
type Agent struct {
	config         config.Agent
	collector      *collector.Collector
	sender         *sender.Sender
	workerPool     *sender.WorkerPool
	metricsChan    chan map[string]*models.Metric
	metricsBuffer  map[string]*models.Metric
	bufferMutex    sync.Mutex
	wg             sync.WaitGroup
	shutdownSignal chan struct{}
	publicKey      *rsa.PublicKey
}

// NewAgent creates a new Agent instance with the given configuration.
func NewAgent(conf config.Agent) *Agent {
	var publicKey *rsa.PublicKey
	var err error

	if conf.CryptoKey != "" {
		publicKey, err = helpers.LoadPublicKey(conf.CryptoKey)
		if err != nil {
			logger.Log.Error("Failed to load public key", zap.Error(err))
		} else {
			logger.Log.Info("Public key loaded successfully")
		}
	}
	return &Agent{
		config:         conf,
		collector:      collector.NewCollector(),
		sender:         sender.NewSender(conf.ServerAddress, conf.Key, publicKey),
		workerPool:     sender.NewWorkerPool(conf.RateLimit),
		metricsChan:    make(chan map[string]*models.Metric, conf.RateLimit*10),
		metricsBuffer:  make(map[string]*models.Metric, 100),
		shutdownSignal: make(chan struct{}),
		publicKey:      publicKey,
	}
}

// Run starts the agent's metric collection and reporting processes.
// It runs until a shutdown signal is received or the context is cancelled.
// Returns an error if the agent fails to start.
func (a *Agent) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()
	logger.Log.Info("Agent started, waiting for shutdown signals")

	a.startWorkers(ctx)

	<-ctx.Done()
	logger.Log.Info("Shutdown signal received, initiating graceful shutdown")

	a.wg.Wait()
	close(a.metricsChan)
	a.workerPool.Wait()
	a.sendRemainingMetrics(context.Background())
	logger.Log.Info("Agent shutdown complete")

	return nil
}

func (a *Agent) startWorkers(ctx context.Context) {
	a.wg.Add(1)
	go a.runMetricsCollector(ctx)

	a.wg.Add(1)
	go a.runGopsutilCollector(ctx)

	a.wg.Add(1)
	go a.runMetricsProcessor(ctx)

	a.wg.Add(1)
	go a.runMetricsSender(ctx)

}

func (a *Agent) runMetricsCollector(ctx context.Context) {
	defer a.wg.Done()

	pollTicker := time.NewTicker(a.config.PollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			metrics := a.collector.CollectMetrics()
			select {
			case a.metricsChan <- metrics:
				logger.Log.Info("Collected runtime metrics")
			default:
				logger.Log.Info("Metrics channel full, dropping runtime metrics")
			}
		}
	}
}

func (a *Agent) runGopsutilCollector(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := a.collector.СollectGopsutilMetrics()
			select {
			case a.metricsChan <- metrics:
				logger.Log.Info("Collected gopsutil metrics")
			default:
				logger.Log.Info("Metrics channel full, dropping gopsutil metrics")
			}
		}
	}
}

func (a *Agent) runMetricsProcessor(ctx context.Context) {
	defer a.wg.Done()

	for {
		select {
		case <-ctx.Done():
			for {
				select {
				case metrics := <-a.metricsChan:
					a.bufferMutex.Lock()
					collector.UpdateMetricsBuffer(a.metricsBuffer, metrics)
					a.bufferMutex.Unlock()
				default:
					return
				}
			}
		case metrics := <-a.metricsChan:
			a.bufferMutex.Lock()
			collector.UpdateMetricsBuffer(a.metricsBuffer, metrics)
			a.bufferMutex.Unlock()
		}
	}
}

func (a *Agent) runMetricsSender(ctx context.Context) {
	defer a.wg.Done()

	reportTicker := time.NewTicker(a.config.ReportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.sendRemainingMetrics(ctx)
			return
		case <-reportTicker.C:
			a.sendCurrentMetrics(ctx)
		}
	}
}

func (a *Agent) sendCurrentMetrics(ctx context.Context) {
	a.bufferMutex.Lock()
	defer a.bufferMutex.Unlock()

	if len(a.metricsBuffer) == 0 {
		return
	}

	metricsCopy := make(map[string]*models.Metric, len(a.metricsBuffer))
	for k, v := range a.metricsBuffer {
		metricsCopy[k] = v
	}

	a.workerPool.Submit(func() {
		if err := a.sender.SendMetricsBatch(ctx, metricsCopy, a.config.Key); err != nil {
			logger.Log.Error("Failed to send metrics batch", zap.Error(err))
			return
		}
	})

	a.metricsBuffer = make(map[string]*models.Metric)
}

func (a *Agent) sendRemainingMetrics(ctx context.Context) {
	a.bufferMutex.Lock()
	defer a.bufferMutex.Unlock()

	if len(a.metricsBuffer) == 0 {
		return
	}

	logger.Log.Info("Sending remaining metrics before shutdown",
		zap.Int("count", len(a.metricsBuffer)))

	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var sendWg sync.WaitGroup
	sendWg.Add(1)

	a.workerPool.Submit(func() {
		defer sendWg.Done()
		if err := a.sender.SendMetricsBatch(sendCtx, a.metricsBuffer, a.config.Key); err != nil {
			logger.Log.Error("Failed to send final metrics batch", zap.Error(err))
		} else {
			logger.Log.Info("Final metrics batch sent successfully")
		}
	})
	sendWg.Wait()
}
