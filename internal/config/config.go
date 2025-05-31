package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Agent struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func SetConfigAgent() Agent {
	var config Agent
	config.ServerAddress = "localhost:8080"
	config.PollInterval = 2 * time.Second
	config.ReportInterval = 10 * time.Second

	address := flag.String("a", "localhost:8080", "HTTP server address")
	poll := flag.Int64("p", 2, "Poll interval in seconds")
	report := flag.Int64("r", 10, "Report interval in seconds")

	flag.Parse()
	config.ServerAddress = *address
	config.PollInterval = time.Duration(*poll) * time.Second
	config.ReportInterval = time.Duration(*report) * time.Second

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		config.ServerAddress = envAddress
	}
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		if reportInterval, err := strconv.Atoi(envReportInterval); err == nil {
			config.ReportInterval = time.Duration(reportInterval) * time.Second
		}
	}
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		if pollInterval, err := strconv.Atoi(envPollInterval); err == nil {
			config.PollInterval = time.Duration(pollInterval) * time.Second
		}
	}

	return config
}

type Server struct {
	ServerAddress   string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
}

func SetConfigServer() Server {
	var config Server
	config.ServerAddress = "localhost:8080"
	config.StoreInterval = 300 * time.Second
	config.FileStoragePath = "metrics.json"
	config.Restore = true
	config.DatabaseDSN = ""

	storeInt := flag.Int("i", 300, "Store interval in seconds")
	filePath := flag.String("f", "metrics.json", "File storage path")
	restore := flag.Bool("r", true, "Restore metrics from file")
	address := flag.String("a", "localhost:8080", "HTTP server address")
	databaseDSN := flag.String("d", "", "Database connection string (DSN)")

	flag.Parse()

	config.ServerAddress = *address
	config.StoreInterval = time.Duration(*storeInt) * time.Second
	config.FileStoragePath = *filePath
	config.Restore = *restore
	config.DatabaseDSN = *databaseDSN

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		config.ServerAddress = envAddress
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		if storeInterval, err := strconv.Atoi(envStoreInterval); err == nil {
			config.StoreInterval = time.Duration(storeInterval) * time.Second
		}
	}
	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		config.FileStoragePath = envFilePath
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		if restoreVal, err := strconv.ParseBool(envRestore); err == nil {
			config.Restore = restoreVal
		}
	}
	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		config.DatabaseDSN = envDatabaseDSN
	}

	return config
}
