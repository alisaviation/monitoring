package main

import (
	"log"
	"net/http"
	"net/http/pprof"
	_ "net/http/pprof"
	"os"

	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/agent"
	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/logger"
)

func main() {
	//go func() {
	//	log.Println(http.ListenAndServe("localhost:8080", nil))
	//}()
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

		server := &http.Server{
			Addr:    "localhost:5050",
			Handler: mux,
		}
		log.Println(server.ListenAndServe())
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
