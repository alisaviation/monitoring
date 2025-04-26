package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/cmd/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server endpoint address")
	interval := flag.Duration("i", 10*time.Second, "Interval for some periodic task (in seconds)")
	timeout := flag.Duration("t", 5*time.Second, "Timeout for some operation (in seconds)")

	flag.Parse()

	if len(flag.Args()) > 0 {
		log.Fatalf("Unknown flags: %v", flag.Args())
	}

	memStorage := storage.NewMemStorage()
	if err := run(memStorage, *addr, *interval, *timeout); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}

func run(memStorage *storage.MemStorage, addr string, interval, timeout time.Duration) error {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(updateMetrics(memStorage)).(http.HandlerFunc))
	r.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(getValue(memStorage)).(http.HandlerFunc))
	r.Get("/", getMetricsList(memStorage))

	log.Printf("Server will use interval: %v and timeout: %v", interval, timeout)

	return http.ListenAndServe(addr, r)
}
