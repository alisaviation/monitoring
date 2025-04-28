package config

import (
	"flag"
	"time"
)

type ConfigAgent struct {
	ServerAddress  string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func SetConfigAgent() ConfigAgent {
	var config ConfigAgent

	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "HTTP server address (default: http://localhost:8080)")
	p := flag.Int64("p", 2, "Poll interval in seconds (default: 2 seconds)")
	r := flag.Int64("r", 10, "Report interval in seconds (default: 10 seconds)")

	flag.Parse()

	config.PollInterval = time.Duration(*p) * time.Second
	config.ReportInterval = time.Duration(*r) * time.Second

	return config
}

type ConfigServer struct {
	ServerAddress string
}

func SetConfigServer() ConfigServer {
	var config ConfigServer
	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "HTTP server address (default: localhost:8080)")
	flag.Parse()

	return config
}
