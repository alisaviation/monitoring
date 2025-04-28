package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/cmd/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server endpoint address")

	flag.Parse()

	if len(flag.Args()) > 0 {
		log.Fatalf("Unknown flags: %v", flag.Args())
	}

	memStorage := storage.NewMemStorage()
	if err := run(memStorage, *addr); err != nil {
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
