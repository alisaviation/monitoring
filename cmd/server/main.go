package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/config"
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
	memStorage := storage.NewMemStorage()
	if err := run(memStorage, conf.ServerAddress); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}

func run(memStorage *storage.MemStorage, addr string) error {
	srvr := &server.Server{MemStorage: memStorage}
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(srvr.UpdateMetrics))
	r.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(srvr.GetValue))
	r.Get("/", server.GetMetricsList(memStorage))

	return http.ListenAndServe(addr, r)
}
