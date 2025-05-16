package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/middleware"
	"github.com/alisaviation/monitoring/internal/server"
	"github.com/alisaviation/monitoring/internal/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

func main() {
	conf := config.SetConfigServer()
	if len(flag.Args()) > 0 {
		log.Fatalf("Unknown flags: %v", flag.Args())
	}
	if address := os.Getenv("ADDRESS"); address != "" {
		conf.ServerAddress = address
	}

	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}

	memStorage := storage.NewMemStorage()
	if err := run(memStorage, conf.ServerAddress); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}

func run(memStorage *storage.MemStorage, addr string) error {
	srvr := &server.Server{MemStorage: memStorage}
	r := chi.NewRouter()
	r.Use(logger.RequestResponseLogger)
	r.Use(middleware.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(srvr.UpdateMetrics))
	r.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(srvr.GetValue))
	r.Post("/update/", helpers.MethodCheck([]string{http.MethodPost})(srvr.UpdateMetrics))
	r.Get("/value/", helpers.MethodCheck([]string{http.MethodGet})(srvr.GetValue))
	r.Post("/value/", helpers.MethodCheck([]string{http.MethodPost})(srvr.GetValue))
	r.Get("/", helpers.MethodCheck([]string{http.MethodGet})(srvr.GetMetricsList))

	return http.ListenAndServe(addr, r)
}
