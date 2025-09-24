package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// ServerConfig represents server configuration from JSON file
type ServerConfig struct {
	Address       string `json:"address"`
	GRPCAddress   string `json:"grpc_address"`
	Restore       bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
	Key           string `json:"key"`
	TrustedSubnet string `json:"trusted_subnet"`
}

// AgentConfig represents agent configuration from JSON file
type AgentConfig struct {
	Address        string `json:"address"`
	GRPCAddress    string `json:"grpc_address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
	Key            string `json:"key"`
	RateLimit      int    `json:"rate_limit"`
	UseGRPC        bool   `json:"use_grpc"`
}

// loadConfigFromFile loads configuration from JSON file
func loadConfigFromFile(filename string, config interface{}) error {
	if filename == "" {
		return nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// parseDuration parses duration string with fallback to seconds if number provided
func parseDuration(durationStr string, defaultValue time.Duration) time.Duration {
	if durationStr == "" {
		return defaultValue
	}

	if dur, err := time.ParseDuration(durationStr); err == nil {
		return dur
	}

	if seconds, err := strconv.Atoi(durationStr); err == nil {
		return time.Duration(seconds) * time.Second
	}

	return defaultValue
}
