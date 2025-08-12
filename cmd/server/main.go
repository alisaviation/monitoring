// Package main provides the entry points for the monitoring agent and server applications.
//
// The server receives and stores metrics, providing HTTP endpoints for metric management.
package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/pprof"
	"os"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/server"
)

func main() {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		mux.Handle("/debug/pprof/heap", http.HandlerFunc(pprof.Handler("heap").ServeHTTP))
		mux.Handle("/debug/pprof/goroutine", http.HandlerFunc(pprof.Handler("goroutine").ServeHTTP))

		server := &http.Server{
			Addr:    "localhost:9090",
			Handler: mux,
		}
		log.Println(server.ListenAndServe())
	}()

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
