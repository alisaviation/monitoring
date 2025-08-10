package agent

import (
	"context"
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
}

func NewAgent(conf config.Agent) *Agent {
	return &Agent{
		config:         conf,
		collector:      collector.NewCollector(),
		sender:         sender.NewSender(conf.ServerAddress, conf.Key),
		workerPool:     sender.NewWorkerPool(conf.RateLimit),
		metricsChan:    make(chan map[string]*models.Metric, conf.RateLimit*10),
		metricsBuffer:  make(map[string]*models.Metric, 100),
		shutdownSignal: make(chan struct{}),
	}
}

func (a *Agent) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go a.handleSignals(cancel)
	a.startWorkers(ctx)

	select {
	case <-a.shutdownSignal:
		logger.Log.Info("Shutdown signal received")
	case <-ctx.Done():
		logger.Log.Info("Context cancelled")
	}
	a.wg.Wait()
	a.workerPool.Wait()
	logger.Log.Info("Shutting down agent")
	return nil
}

func (a *Agent) handleSignals(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Log.Info("Received signal, agent is shutting down...", zap.String("signal", sig.String()))
	cancel()
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
			return
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

	a.workerPool.Submit(func() {
		if err := a.sender.SendMetricsBatch(ctx, a.metricsBuffer, a.config.Key); err != nil {
			logger.Log.Error("Failed to send final metrics batch", zap.Error(err))
		}
	})
}
