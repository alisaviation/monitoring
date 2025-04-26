package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/cmd/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

func main() {
	memStorage := storage.NewMemStorage()
	if err := run(memStorage); err != nil {
		panic(err)
	}
}

func run(memStorage *storage.MemStorage) error {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(updateMetrics(memStorage)).(http.HandlerFunc))
	r.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(getValue(memStorage)).(http.HandlerFunc))
	r.Get("/", getMetricsList(memStorage))

	return http.ListenAndServe(`:8080`, r)
}
