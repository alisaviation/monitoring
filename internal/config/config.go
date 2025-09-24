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
	GRPCAddress    string
	PollInterval   time.Duration
	ReportInterval time.Duration
	Key            string
	RateLimit      int
	CryptoKey      string
	UseGRPC        bool
}

func SetConfigAgent() Agent {
	var config Agent
	var configFile string

	flag.StringVar(&configFile, "c", "", "Path to config file")
	flag.StringVar(&configFile, "config", "", "Path to config file")
	address := flag.String("a", "localhost:8080", "HTTP server address")
	grpcAddress := flag.String("ga", "localhost:8081", "gRPC server address")
	poll := flag.Int64("p", 2, "Poll interval in seconds")
	report := flag.Int64("r", 10, "Report interval in seconds")
	key := flag.String("k", "", "Hash key")
	limit := flag.Int("l", 5, "Rate limit")
	cryptoKey := flag.String("crypto-key", "", "Path to public key for encryption")
	useGRPC := flag.Bool("grpc_handlers", false, "Use gRPC instead of HTTP")

	flag.Parse()

	defaultConfig := Agent{
		ServerAddress:  "localhost:8080",
		GRPCAddress:    "localhost:8081",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		Key:            "",
		RateLimit:      5,
		CryptoKey:      "",
		UseGRPC:        false,
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
			if fileConfig.GRPCAddress != "" {
				config.GRPCAddress = fileConfig.GRPCAddress
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
			if fileConfig.UseGRPC {
				config.UseGRPC = fileConfig.UseGRPC
			}
		}
	}

	config.ServerAddress = *address
	config.GRPCAddress = *grpcAddress
	config.PollInterval = time.Duration(*poll) * time.Second
	config.ReportInterval = time.Duration(*report) * time.Second
	config.Key = *key
	config.RateLimit = *limit
	config.CryptoKey = *cryptoKey
	config.UseGRPC = *useGRPC

	if envAddress, exists := os.LookupEnv("ADDRESS"); exists {
		config.ServerAddress = envAddress
	}
	if envGRPCAddress, exists := os.LookupEnv("GRPC_ADDRESS"); exists {
		config.GRPCAddress = envGRPCAddress
	}
	if envReportInterval, exists := os.LookupEnv("REPORT_INTERVAL"); exists {
		if reportInterval, err := strconv.Atoi(envReportInterval); err == nil {
			config.ReportInterval = time.Duration(reportInterval) * time.Second
		}
	}
	if envPollInterval, exists := os.LookupEnv("POLL_INTERVAL"); exists {
		if pollInterval, err := strconv.Atoi(envPollInterval); err == nil {
			config.PollInterval = time.Duration(pollInterval) * time.Second
		}
	}
	if envKey, exists := os.LookupEnv("KEY"); exists {
		config.Key = envKey
	}
	if envLimit, exists := os.LookupEnv("RATE_LIMIT"); exists {
		if ratelimit, err := strconv.Atoi(envLimit); err == nil {
			config.RateLimit = ratelimit
		}
	}
	if envCryptoKey, exists := os.LookupEnv("CRYPTO_KEY"); exists {
		config.CryptoKey = envCryptoKey
	}
	if envUseGRPC, exists := os.LookupEnv("USE_GRPC"); exists {
		if useGRPC, err := strconv.ParseBool(envUseGRPC); err == nil {
			config.UseGRPC = useGRPC
		}
	}
	if envConfigFile, exists := os.LookupEnv("CONFIG"); exists {
		configFile = envConfigFile
		var fileConfig AgentConfig
		if err := loadConfigFromFile(configFile, &fileConfig); err == nil {
			if config.ServerAddress == "localhost:8080" && fileConfig.Address != "" {
				config.ServerAddress = fileConfig.Address
			}
			if config.GRPCAddress == "localhost:8081" && fileConfig.GRPCAddress != "" {
				config.GRPCAddress = fileConfig.GRPCAddress
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
			if !config.UseGRPC && fileConfig.UseGRPC {
				config.UseGRPC = fileConfig.UseGRPC
			}
		}
	}

	return config
}

type Server struct {
	ServerAddress   string
	GRPCAddress     string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	Key             string
	CryptoKey       string
	TrustedSubnet   string
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
	grpcAddress := flag.String("ga", "localhost:8081", "gRPC server address")
	databaseDSN := flag.String("d", "", "Database connection string (DSN)")
	key := flag.String("k", "", "Hash key")
	cryptoKey := flag.String("crypto-key", "", "Path to private key for decryption")
	trustedSubnet := flag.String("t", "", "Trusted subnet in CIDR notation")
	flag.Parse()

	defaultConfig := Server{
		ServerAddress:   "localhost:8080",
		GRPCAddress:     "localhost:8081",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "metrics.json",
		Restore:         true,
		DatabaseDSN:     "",
		Key:             "",
		CryptoKey:       "",
		TrustedSubnet:   "",
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
			if fileConfig.GRPCAddress != "" {
				config.GRPCAddress = fileConfig.GRPCAddress
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
			if fileConfig.TrustedSubnet != "" {
				config.TrustedSubnet = fileConfig.TrustedSubnet
			}
			config.Restore = fileConfig.Restore
		}
	}

	config.ServerAddress = *address
	config.GRPCAddress = *grpcAddress
	config.StoreInterval = time.Duration(*storeInt) * time.Second
	config.FileStoragePath = *filePath
	config.Restore = *restore
	config.DatabaseDSN = *databaseDSN
	config.Key = *key
	config.CryptoKey = *cryptoKey
	config.TrustedSubnet = *trustedSubnet

	if envAddress, exists := os.LookupEnv("ADDRESS"); exists {
		config.ServerAddress = envAddress
	}
	if envGRPCAddress, exists := os.LookupEnv("GRPC_ADDRESS"); exists {
		config.GRPCAddress = envGRPCAddress
	}
	if envStoreInterval, exists := os.LookupEnv("STORE_INTERVAL"); exists {
		if storeInterval, err := strconv.Atoi(envStoreInterval); err == nil {
			config.StoreInterval = time.Duration(storeInterval) * time.Second
		}
	}
	if envFilePath, exists := os.LookupEnv("FILE_STORAGE_PATH"); exists {
		config.FileStoragePath = envFilePath
	}
	if envRestore, exists := os.LookupEnv("RESTORE"); exists {
		if restoreVal, err := strconv.ParseBool(envRestore); err == nil {
			config.Restore = restoreVal
		}
	}
	if envDatabaseDSN, exists := os.LookupEnv("DATABASE_DSN"); exists {
		config.DatabaseDSN = envDatabaseDSN
	}
	if envKey, exists := os.LookupEnv("KEY"); exists {
		config.Key = envKey
	}
	if envCryptoKey, exists := os.LookupEnv("CRYPTO_KEY"); exists {
		config.CryptoKey = envCryptoKey
	}
	if envTrustedSubnet, exists := os.LookupEnv("TRUSTED_SUBNET"); exists {
		config.TrustedSubnet = envTrustedSubnet
	}
	if envConfigFile, exists := os.LookupEnv("CONFIG"); exists {
		configFile = envConfigFile
		var fileConfig ServerConfig
		if err := loadConfigFromFile(configFile, &fileConfig); err == nil {
			if config.ServerAddress == "localhost:8080" && fileConfig.Address != "" {
				config.ServerAddress = fileConfig.Address
			}
			if config.GRPCAddress == "localhost:8081" && fileConfig.GRPCAddress != "" {
				config.GRPCAddress = fileConfig.GRPCAddress
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
			if config.TrustedSubnet == "" && fileConfig.TrustedSubnet != "" {
				config.TrustedSubnet = fileConfig.TrustedSubnet
			}
		}
	}

	return config
}
