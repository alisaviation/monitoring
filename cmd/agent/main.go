package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alisaviation/monitoring/cmd/agent/collector"
	"github.com/alisaviation/monitoring/cmd/agent/sender"

	"github.com/alisaviation/monitoring/internal/models"
)

func main() {
	defaultServerAddress := "http://localhost:8080"
	defaultReportInterval := 10 * time.Second
	defaultPollInterval := 2 * time.Second

	serverAddressEnv := os.Getenv("ADDRESS")
	reportIntervalEnv := os.Getenv("REPORT_INTERVAL")
	pollIntervalEnv := os.Getenv("POLL_INTERVAL")

	if serverAddressEnv != "" {
		defaultServerAddress = serverAddressEnv
	}
	if reportIntervalEnv != "" {
		var err error
		defaultReportInterval, err = time.ParseDuration(reportIntervalEnv + "s")
		if err != nil {
			log.Fatalf("Invalid REPORT_INTERVAL value: %v", err)
		}
	}
	if pollIntervalEnv != "" {
		var err error
		defaultPollInterval, err = time.ParseDuration(pollIntervalEnv + "s")
		if err != nil {
			log.Fatalf("Invalid POLL_INTERVAL value: %v", err)
		}
	}

	serverAddress := flag.String("a", defaultServerAddress, "HTTP server endpoint address")
	reportInterval := flag.Duration("r", defaultReportInterval, "Report interval for sending metrics (in seconds)")
	pollInterval := flag.Duration("p", defaultPollInterval, "Poll interval for collecting metrics (in seconds)")

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
