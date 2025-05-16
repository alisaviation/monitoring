package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/middleware"
	"github.com/alisaviation/monitoring/internal/server"
	"github.com/alisaviation/monitoring/internal/storage"
)

func main() {
	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}

	conf := config.SetConfigServer()
	if len(flag.Args()) > 0 {
		log.Fatalf("Unknown flags: %v", flag.Args())
	}

	helpers.CheckEnvServerVariables(&conf)

	memStorage := storage.NewMemStorage()

	if conf.Restore {
		if err := memStorage.Load(conf.FileStoragePath); err != nil {
			log.Fatalf("Error loading metrics from file: %v", err)
		}
	}

	if err := run(memStorage, conf); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}

func run(memStorage *storage.MemStorage, conf config.Server) error {
	srvr := &server.Server{MemStorage: memStorage, Config: conf}
	r := chi.NewRouter()
	r.Use(logger.RequestResponseLogger)
	r.Use(middleware.GzipMiddleware)

	r.Post("/update/", helpers.MethodCheck([]string{http.MethodPost})(srvr.UpdateMetrics))
	r.Get("/value/", helpers.MethodCheck([]string{http.MethodGet})(srvr.GetValue))
	r.Post("/value/", helpers.MethodCheck([]string{http.MethodPost})(srvr.GetValue))
	r.Get("/", helpers.MethodCheck([]string{http.MethodGet})(srvr.GetMetricsList))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if conf.StoreInterval > 0 {
			ticker := time.NewTicker(time.Duration(conf.StoreInterval) * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := memStorage.Save(conf.FileStoragePath); err != nil {
						log.Fatalf("Error saving metrics to file: %v", err)
					}
				case <-done:
					return
				}
			}
		}
	}()

	if conf.StoreInterval == 0 {
		if err := memStorage.Save(conf.FileStoragePath); err != nil {
			log.Fatalf("Error saving metrics to file: %v", err)
		}
	}

	go func() {
		<-done
		if err := memStorage.Save(conf.FileStoragePath); err != nil {
			log.Fatalf("Error saving metrics to file on shutdown: %v", err)
		}
		os.Exit(0)
	}()

	return http.ListenAndServe(conf.ServerAddress, r)
}
