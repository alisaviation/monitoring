package main

import (
	"flag"
	"log"
	"os"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/server"
)

func main() {
	conf := config.SetConfigServer()
	if len(flag.Args()) > 0 {
		logger.Log.Fatal("Unknown flags", zap.Strings("flags", flag.Args()))
	}

	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}
	defer logger.Log.Sync()

	app := server.NewServerApp(conf)
	if err := app.Run(); err != nil {
		logger.Log.Error("Application failed", zap.Error(err))
		os.Exit(1)
	}
}
