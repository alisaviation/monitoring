package main

import (
	"log"
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
	if collectorInstance == nil {
		log.Fatal("Collector instance is nil")
	}
	senderInstance := sender.NewSender(serverAddress)
	if senderInstance == nil {
		log.Fatal("Sender instance is nil")
	}

	metricsBuffer := make(map[string]models.Metric)
	lastReportTime := time.Now()

	//go func() {
	//	if err := http.ListenAndServe(":8080", nil); err != nil {
	//		fmt.Printf("Error starting HTTP server: %v\n", err)
	//	}
	//}()
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

		time.Sleep(pollInterval)

		currentTime := time.Now()
		if currentTime.Sub(lastReportTime) >= reportInterval {
			senderInstance.SendMetrics(metricsBuffer)
			metricsBuffer = make(map[string]models.Metric)
			lastReportTime = currentTime
		}
	}
}
