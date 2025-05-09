package main

import (
	"os"
	"strconv"
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

	if address := os.Getenv("ADDRESS"); address != "" {
		conf.ServerAddress = address
	}
	if reportIntervalStr := os.Getenv("REPORT_INTERVAL"); reportIntervalStr != "" {
		if reportInterval, err := strconv.Atoi(reportIntervalStr); err == nil {
			conf.ReportInterval = time.Duration(reportInterval) * time.Second
		}
	}
	if pollIntervalStr := os.Getenv("POLL_INTERVAL"); pollIntervalStr != "" {
		if pollInterval, err := strconv.Atoi(pollIntervalStr); err == nil {
			conf.PollInterval = time.Duration(pollInterval) * time.Second
		}
	}

	collectorInstance := collector.NewCollector()
	senderInstance := sender.NewSender(conf.ServerAddress)
	metricsBuffer := make(map[string]*models.Metric)

	pollTicker := time.NewTicker(conf.PollInterval)
	reportTicker := time.NewTicker(conf.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-pollTicker.C:
			metrics := collectorInstance.CollectMetrics()
			collector.UpdateMetricsBuffer(metricsBuffer, metrics)

		case <-reportTicker.C:
			senderInstance.SendMetrics(metricsBuffer)
			metricsBuffer = make(map[string]*models.Metric)
			for name, metric := range metricsBuffer {
				receivedMetric, err := senderInstance.GetMetric(metric)
				if err != nil {
					logger.Log.Error("Error getting metric:", zap.Error(err))
					continue
				}
				metricsBuffer[name] = receivedMetric
			}
		}
	}
}
