package helpers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alisaviation/monitoring/internal/config"
)

func MethodCheck(methods []string) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			for _, method := range methods {
				if r.Method == method {
					next(w, r)
					return
				}
			}
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func FormatFloat(value float64) string {
	formatted := fmt.Sprintf("%.3f", value)
	return strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
}

func CheckEnvServerVariables(conf *config.Server) {
	if address := os.Getenv("ADDRESS"); address != "" {
		conf.ServerAddress = address
	}
	if storeInterval := os.Getenv("STORE_INTERVAL"); storeInterval != "" {
		interval, err := strconv.Atoi(storeInterval)
		if err != nil {
			log.Fatalf("Invalid STORE_INTERVAL: %v", err)
		}
		conf.StoreInterval = interval
	}
	if filePath := os.Getenv("FILE_STORAGE_PATH"); filePath != "" {
		conf.FileStoragePath = filePath
	}
	if restore := os.Getenv("RESTORE"); restore != "" {
		restoreValue, err := strconv.ParseBool(restore)
		if err != nil {
			log.Fatalf("Invalid RESTORE: %v", err)
		}
		conf.Restore = restoreValue

	}
}
func CheckEnvAgentVariables(conf *config.Agent) {
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
}
