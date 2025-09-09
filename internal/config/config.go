// Package config handles configuration management for both agent and server.
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Agent struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
	Key            string
	RateLimit      int
	CryptoKey      string
}

func SetConfigAgent() Agent {
	var config Agent
	var configFile string

	flag.StringVar(&configFile, "c", "", "Path to config file")
	flag.StringVar(&configFile, "config", "", "Path to config file")
	address := flag.String("a", "localhost:8080", "HTTP server address")
	poll := flag.Int64("p", 2, "Poll interval in seconds")
	report := flag.Int64("r", 10, "Report interval in seconds")
	key := flag.String("k", "", "Hash key")
	limit := flag.Int("l", 5, "Rate limit")
	cryptoKey := flag.String("crypto-key", "", "Path to public key for encryption")

	flag.Parse()

	defaultConfig := Agent{
		ServerAddress:  "localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		Key:            "",
		RateLimit:      5,
		CryptoKey:      "",
	}

	config = defaultConfig
	if configFile != "" {
		var fileConfig AgentConfig
		if err := loadConfigFromFile(configFile, &fileConfig); err != nil {
			fmt.Printf("Warning: failed to load config file: %v\n", err)
		} else {
			if fileConfig.Address != "" {
				config.ServerAddress = fileConfig.Address
			}
			if fileConfig.PollInterval != "" {
				config.PollInterval = parseDuration(fileConfig.PollInterval, defaultConfig.PollInterval)
			}
			if fileConfig.ReportInterval != "" {
				config.ReportInterval = parseDuration(fileConfig.ReportInterval, defaultConfig.ReportInterval)
			}
			if fileConfig.Key != "" {
				config.Key = fileConfig.Key
			}
			if fileConfig.RateLimit > 0 {
				config.RateLimit = fileConfig.RateLimit
			}
			if fileConfig.CryptoKey != "" {
				config.CryptoKey = fileConfig.CryptoKey
			}
		}
	}

	config.ServerAddress = *address
	config.PollInterval = time.Duration(*poll) * time.Second
	config.ReportInterval = time.Duration(*report) * time.Second
	config.Key = *key
	config.RateLimit = *limit
	config.CryptoKey = *cryptoKey

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
	if envKey := os.Getenv("KEY"); envKey != "" {
		config.Key = envKey
	}
	if envLimit := os.Getenv("RATE_LIMIT"); envLimit != "" {
		if ratelimit, err := strconv.Atoi(envLimit); err == nil {
			config.RateLimit = ratelimit
		}
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		config.CryptoKey = envCryptoKey
	}
	if envConfigFile := os.Getenv("CONFIG"); envConfigFile != "" {
		configFile = envConfigFile
		var fileConfig AgentConfig
		if err := loadConfigFromFile(configFile, &fileConfig); err == nil {
			if config.ServerAddress == "localhost:8080" && fileConfig.Address != "" {
				config.ServerAddress = fileConfig.Address
			}
			if config.PollInterval == 2*time.Second && fileConfig.PollInterval != "" {
				config.PollInterval = parseDuration(fileConfig.PollInterval, config.PollInterval)
			}
			if config.ReportInterval == 10*time.Second && fileConfig.ReportInterval != "" {
				config.ReportInterval = parseDuration(fileConfig.ReportInterval, config.ReportInterval)
			}
			if config.Key == "" && fileConfig.Key != "" {
				config.Key = fileConfig.Key
			}
			if config.RateLimit == 5 && fileConfig.RateLimit > 0 {
				config.RateLimit = fileConfig.RateLimit
			}
			if config.CryptoKey == "" && fileConfig.CryptoKey != "" {
				config.CryptoKey = fileConfig.CryptoKey
			}
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
	Key             string
	CryptoKey       string
}

func SetConfigServer() Server {
	var config Server
	var configFile string

	flag.StringVar(&configFile, "c", "", "Path to config file")
	flag.StringVar(&configFile, "config", "", "Path to config file")
	storeInt := flag.Int("i", 300, "Store interval in seconds")
	filePath := flag.String("f", "metrics.json", "File storage path")
	restore := flag.Bool("r", true, "Restore metrics from file")
	address := flag.String("a", "localhost:8080", "HTTP server address")
	databaseDSN := flag.String("d", "", "Database connection string (DSN)")
	key := flag.String("k", "", "Hash key")
	cryptoKey := flag.String("crypto-key", "", "Path to private key for decryption")
	flag.Parse()

	defaultConfig := Server{
		ServerAddress:   "localhost:8787",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "metrics.json",
		Restore:         true,
		DatabaseDSN:     "",
		Key:             "",
		CryptoKey:       "",
	}

	config = defaultConfig
	if configFile != "" {
		var fileConfig ServerConfig
		if err := loadConfigFromFile(configFile, &fileConfig); err != nil {
			fmt.Printf("Warning: failed to load config file: %v\n", err)
		} else {
			if fileConfig.Address != "" {
				config.ServerAddress = fileConfig.Address
			}
			if fileConfig.StoreInterval != "" {
				config.StoreInterval = parseDuration(fileConfig.StoreInterval, defaultConfig.StoreInterval)
			}
			if fileConfig.StoreFile != "" {
				config.FileStoragePath = fileConfig.StoreFile
			}
			if fileConfig.DatabaseDSN != "" {
				config.DatabaseDSN = fileConfig.DatabaseDSN
			}
			if fileConfig.Key != "" {
				config.Key = fileConfig.Key
			}
			if fileConfig.CryptoKey != "" {
				config.CryptoKey = fileConfig.CryptoKey
			}
			config.Restore = fileConfig.Restore
		}
	}

	config.ServerAddress = *address
	config.StoreInterval = time.Duration(*storeInt) * time.Second
	config.FileStoragePath = *filePath
	config.Restore = *restore
	config.DatabaseDSN = *databaseDSN
	config.Key = *key
	config.CryptoKey = *cryptoKey

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
	if envKey := os.Getenv("KEY"); envKey != "" {
		config.Key = envKey
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		config.CryptoKey = envCryptoKey
	}
	if envConfigFile := os.Getenv("CONFIG"); envConfigFile != "" {
		configFile = envConfigFile
		var fileConfig ServerConfig
		if err := loadConfigFromFile(configFile, &fileConfig); err == nil {
			if config.ServerAddress == "localhost:8080" && fileConfig.Address != "" {
				config.ServerAddress = fileConfig.Address
			}
			if config.StoreInterval == 300*time.Second && fileConfig.StoreInterval != "" {
				config.StoreInterval = parseDuration(fileConfig.StoreInterval, config.StoreInterval)
			}
			if config.FileStoragePath == "metrics.json" && fileConfig.StoreFile != "" {
				config.FileStoragePath = fileConfig.StoreFile
			}
			if config.DatabaseDSN == "" && fileConfig.DatabaseDSN != "" {
				config.DatabaseDSN = fileConfig.DatabaseDSN
			}
			if config.Key == "" && fileConfig.Key != "" {
				config.Key = fileConfig.Key
			}
			if config.CryptoKey == "" && fileConfig.CryptoKey != "" {
				config.CryptoKey = fileConfig.CryptoKey
			}
			if config.Restore && !fileConfig.Restore {
				config.Restore = fileConfig.Restore
			}
		}
	}

	return config
}
