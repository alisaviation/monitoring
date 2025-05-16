package config

import (
	"flag"
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
	config.PollInterval = 2
	config.ReportInterval = 10

	flag.StringVar(&config.ServerAddress, "a", config.ServerAddress, "HTTP server address")
	p := flag.Int64("p", 2, "Poll interval in seconds")
	r := flag.Int64("r", 10, "Report interval in seconds")

	flag.Parse()

	config.PollInterval = time.Duration(*p) * time.Second
	config.ReportInterval = time.Duration(*r) * time.Second

	return config
}

type Server struct {
	ServerAddress   string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
}

func SetConfigServer() Server {
	var config Server
	config.ServerAddress = "localhost:8080"
	config.StoreInterval = 300
	config.FileStoragePath = "metrics.json"
	config.Restore = false

	flag.StringVar(&config.ServerAddress, "a", config.ServerAddress, "Server address")
	flag.IntVar(&config.StoreInterval, "i", config.StoreInterval, "Interval in seconds to store metrics")
	flag.StringVar(&config.FileStoragePath, "f", config.FileStoragePath, "File path")
	flag.BoolVar(&config.Restore, "r", config.Restore, "Restore metrics from file on start")

	flag.Parse()

	return config
}
