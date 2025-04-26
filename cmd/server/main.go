package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/cmd/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

func main() {
	defaultAddress := "localhost:8080"
	addressEnv := os.Getenv("ADDRESS")
	address := flag.String("a", defaultAddress, "HTTP server endpoint address")

	flag.Parse()

	if len(flag.Args()) > 0 {
		log.Fatalf("Unknown flags: %v", flag.Args())
	}

	var serverAddress string
	if addressEnv != "" {
		serverAddress = addressEnv
	} else {
		serverAddress = *address
	}

	memStorage := storage.NewMemStorage()
	if err := run(memStorage, serverAddress); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}

func run(memStorage *storage.MemStorage, addr string) error {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(updateMetrics(memStorage)).(http.HandlerFunc))
	r.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(getValue(memStorage)).(http.HandlerFunc))
	r.Get("/", getMetricsList(memStorage))

	return http.ListenAndServe(addr, r)
}
