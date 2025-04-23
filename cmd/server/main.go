package main

import (
	"net/http"

	"github.com/alisaviation/monitoring/internal/storage"
)

func main() {
	memStorage := storage.NewMemStorage()
	if err := run(memStorage); err != nil {
		panic(err)
	}
}

func run(memStorage *storage.MemStorage) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", methodCheck([]string{http.MethodPost})(updateMetrics(memStorage)))
	return http.ListenAndServe(`:8080`, mux)
}
