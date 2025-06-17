package main

import (
	"log"
	"os"

	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/agent"
	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/logger"
)

func main() {
	conf := config.SetConfigAgent()

	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Log.Sync()

	agentInstance := agent.NewAgent(conf)
	if err := agentInstance.Run(); err != nil {
		logger.Log.Error("Agent failed", zap.Error(err))
		os.Exit(1)
	}

}
