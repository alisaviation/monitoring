package main

import (
	"time"

	"github.com/alisaviation/monitoring/internal/agent/collector"
	"github.com/alisaviation/monitoring/internal/agent/sender"
	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/models"
)

//func main() {
//	conf := config.SetConfigAgent()
//
//	collectorInstance := collector.NewCollector()
//	senderInstance := sender.NewSender(conf.ServerAddress)
//	metricsBuffer := make(map[string]*models.Metric)
//
//	pollTicker := time.NewTicker(conf.PollInterval)
//	reportTicker := time.NewTicker(conf.ReportInterval)
//	defer pollTicker.Stop()
//	defer reportTicker.Stop()
//
//	for {
//		select {
//		case <-pollTicker.C:
//			metrics := collectorInstance.CollectMetrics()
//			collector.UpdateMetricsBuffer(metricsBuffer, metrics)
//
//		case <-reportTicker.C:
//			senderInstance.SendMetrics(metricsBuffer)
//			metricsBuffer = make(map[string]*models.Metric)
//			for name, metric := range metricsBuffer {
//				receivedMetric, err := senderInstance.GetMetric(metric)
//				if err != nil {
//					logger.Log.Error("Error getting metric:", zap.Error(err))
//					continue
//				}
//				metricsBuffer[name] = receivedMetric
//			}
//		}
//	}
//}

func main() {
	conf := config.SetConfigAgent()

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
			if len(metricsBuffer) > 0 {
				senderInstance.SendMetricsBatch(metricsBuffer)
				metricsBuffer = make(map[string]*models.Metric)
			}
		}
	}
}
