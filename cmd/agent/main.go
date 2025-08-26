// Package main provides the entry points for the monitoring agent and server applications.
//
// The agent collects system and application metrics and sends them to the server.
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/agent"
	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/logger"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:8080", nil))
	}()

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
