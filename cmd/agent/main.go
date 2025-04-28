package main

import (
	"strings"
	"time"

	"github.com/alisaviation/monitoring/cmd/agent/collector"
	"github.com/alisaviation/monitoring/cmd/agent/sender"
	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/models"
)

func main() {
	conf := config.SetConfigAgent()
	if !strings.Contains(conf.ServerAddress, "://") {
		conf.ServerAddress = "http://" + conf.ServerAddress
	}
	collectorInstance := collector.NewCollector()
	senderInstance := sender.NewSender(conf.ServerAddress)
	metricsBuffer := make(map[string]models.Metric)
	lastReportTime := time.Now()
	for {
		metrics := collectorInstance.CollectMetrics()
		for name, metric := range metrics {
			if metric.Type == models.Counter {
				if existingMetric, exists := metricsBuffer[name]; exists {
					metricsBuffer[name] = models.Metric{
						Name:  existingMetric.Name,
						Value: existingMetric.Value + metric.Value,
						Type:  existingMetric.Type,
					}
				} else {
					metricsBuffer[name] = metric
				}
			} else {
				metricsBuffer[name] = metric
			}
		}

		time.Sleep(conf.PollInterval)
		currentTime := time.Now()
		if currentTime.Sub(lastReportTime) >= conf.ReportInterval {
			senderInstance.SendMetrics(metricsBuffer)
			metricsBuffer = make(map[string]models.Metric)
			lastReportTime = currentTime
		}
	}
}
