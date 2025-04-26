package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/alisaviation/monitoring/cmd/agent/collector"
	"github.com/alisaviation/monitoring/cmd/agent/sender"

	"github.com/alisaviation/monitoring/internal/models"
)

func main() {
	serverAddress := flag.String("a", "http://localhost:8080", "HTTP server endpoint address")
	reportInterval := flag.Duration("r", 10*time.Second, "Report interval for sending metrics (in seconds)")
	pollInterval := flag.Duration("p", 2*time.Second, "Poll interval for collecting metrics (in seconds)")

	flag.Parse()
	if len(flag.Args()) > 0 {
		log.Fatalf("Unknown flags: %v", flag.Args())
	}

	collectorInstance := collector.NewCollector()
	senderInstance := sender.NewSender(*serverAddress)

	metricsBuffer := make(map[string]models.Metric)
	lastReportTime := time.Now()

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

		time.Sleep(*pollInterval)

		if time.Since(lastReportTime) >= *reportInterval {
			senderInstance.SendMetrics(metricsBuffer)
			metricsBuffer = make(map[string]models.Metric)
			lastReportTime = time.Now()
		}
	}
}
