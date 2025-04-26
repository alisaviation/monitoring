package main

import (
	"fmt"
	"time"

	"github.com/alisaviation/monitoring/cmd/agent/collector"
	"github.com/alisaviation/monitoring/cmd/agent/sender"

	"github.com/alisaviation/monitoring/internal/models"
)

func main() {
	pollInterval := 2 * time.Second
	reportInterval := 10 * time.Second
	serverAddress := "http://localhost:8080"

	collectorInstance := collector.NewCollector()
	senderInstance := sender.NewSender(serverAddress)

	metricsBuffer := make(map[string]models.Metric)

	for {
		metrics := collectorInstance.CollectMetrics()

		for id, metric := range metrics {
			if metric.Type == models.Counter {
				if existingMetric, exists := metricsBuffer[id]; exists {
					metricsBuffer[id] = models.Metric{
						Name:  existingMetric.Name,
						Value: existingMetric.Value + metric.Value,
						Type:  existingMetric.Type,
					}
					fmt.Println("count", metricsBuffer)
				} else {
					metricsBuffer[id] = metric
				}
			} else {
				metricsBuffer[id] = metric
			}
		}

		time.Sleep(pollInterval)

		if time.Now().UnixNano()%int64(reportInterval) == 0 {
			senderInstance.SendMetrics(metricsBuffer)
			metricsBuffer = make(map[string]models.Metric)
		}
	}
}
