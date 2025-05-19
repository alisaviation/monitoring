package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/middleware"
	"github.com/alisaviation/monitoring/internal/server"
	"github.com/alisaviation/monitoring/internal/server/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

func main() {
	var saveTicker *time.Ticker
	conf := config.SetConfigServer()
	if len(flag.Args()) > 0 {
		log.Fatalf("Unknown flags: %v", flag.Args())
	}

	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}

	memStorage := storage.NewMemStorage(conf.FileStoragePath)

	if conf.Restore {
		if err := memStorage.Load(); err != nil {
			logger.Log.Info("Could not load metrics from file", zap.Error(err))
		} else {
			logger.Log.Info("Metrics loaded from file", zap.String("path", conf.FileStoragePath))
		}
	}

	if conf.StoreInterval > 0 {
		saveTicker = time.NewTicker(conf.StoreInterval)
		go func() {
			for range saveTicker.C {
				if err := memStorage.Save(); err != nil {
					logger.Log.Error("Error saving metrics", zap.Error(err))
				} else {
					logger.Log.Debug("Metrics saved to file", zap.String("path", conf.FileStoragePath))
				}
			}
		}()
	} else {
		logger.Log.Info("Synchronous save mode enabled")
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{Addr: conf.ServerAddress}
	go func() {
		if err := run(memStorage, srv, conf.StoreInterval); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error running server: %v", err)
		}
	}()
	logger.Log.Info("Server started", zap.String("address", conf.ServerAddress))

	<-done
	logger.Log.Info("Server is shutting down...")
	if saveTicker != nil {
		saveTicker.Stop()
	}
	if err := memStorage.Save(); err != nil {
		logger.Log.Error("Error saving metrics on shutdown", zap.Error(err))
	} else {
		logger.Log.Info("Metrics saved on shutdown")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server shutdown failed", zap.Error(err))
	}
	logger.Log.Info("Server stopped")
}

func run(memStorage *storage.MemStorage, srv *http.Server, storeInterval time.Duration) error {
	srvr := &server.Server{MemStorage: memStorage}
	r := chi.NewRouter()
	r.Use(logger.RequestResponseLogger)
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.SyncSaveMiddleware(storeInterval, memStorage))

	r.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(srvr.UpdateMetrics))
	r.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(srvr.GetValue))
	r.Post("/update/", helpers.MethodCheck([]string{http.MethodPost})(srvr.UpdateMetrics))
	r.Get("/value/", helpers.MethodCheck([]string{http.MethodGet})(srvr.GetValue))
	r.Post("/value/", helpers.MethodCheck([]string{http.MethodPost})(srvr.GetValue))
	r.Get("/", helpers.MethodCheck([]string{http.MethodGet})(srvr.GetMetricsList))

	srv.Handler = r
	return srv.ListenAndServe()
}
