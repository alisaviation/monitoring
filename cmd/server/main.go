// Package main provides the entry points for the monitoring agent and server applications.
//
// The server receives and stores metrics, providing HTTP endpoints for metric management.
package main

import (
	"flag"
	"log"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/server"
)

// Build information variables set during compilation
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	helpers.PrintBuildInfo(buildVersion, buildDate, buildCommit)
	helpers.StartPProfServer("localhost:9090")

	conf := config.SetConfigServer()
	if len(flag.Args()) > 0 {
		logger.Log.Fatal("Unknown flags", zap.Strings("flags", flag.Args()))
		return
	}

	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("Error initializing logger: %v", err)
		return
	}
	defer logger.Log.Sync()

	app := server.NewServerApp(conf)
	if err := app.Run(); err != nil {
		logger.Log.Error("Application failed", zap.Error(err))
	}
}
