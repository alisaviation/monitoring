// Package main provides the entry points for the monitoring agent and server applications.
//
// The agent collects system and application metrics and sends them to the server.
package main

import (
	"log"
	_ "net/http/pprof"

	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/agent"
	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
)

// Build information variables set during compilation
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	helpers.PrintBuildInfo(buildVersion, buildDate, buildCommit)
	helpers.StartPProfServer("localhost:8080")

	conf := config.SetConfigAgent()

	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Log.Sync()

	agentInstance := agent.NewAgent(conf)
	if err := agentInstance.Run(); err != nil {
		logger.Log.Error("Agent failed", zap.Error(err))
	}
}
