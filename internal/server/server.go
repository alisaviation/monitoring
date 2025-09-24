// Package server implements the monitoring server that receives and stores metrics.
package server

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/alisaviation/monitoring/internal/config"
	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/middleware"
	"github.com/alisaviation/monitoring/internal/server/grpc_handlers"
	"github.com/alisaviation/monitoring/internal/server/http_handlers"
	"github.com/alisaviation/monitoring/internal/service"
	"github.com/alisaviation/monitoring/internal/storage"
	"github.com/alisaviation/monitoring/proto/brief/rpc"
)

// ServerApp represents the main server application.
// It manages the HTTP server, storage backend, and application lifecycle.
type ServerApp struct {
	config         config.Server
	storage        storage.Storage
	db             *sql.DB
	httpServer     *http.Server
	grpcServer     *grpc.Server
	shutdownSignal chan struct{}
	wg             sync.WaitGroup
	mu             sync.RWMutex
	privateKey     *rsa.PrivateKey
	metricsService *service.MetricsService
}

// NewServerApp creates a new ServerApp instance with the given configuration.
func NewServerApp(conf config.Server) *ServerApp {
	var privateKey *rsa.PrivateKey
	var err error

	if conf.CryptoKey != "" {
		privateKey, err = helpers.LoadPrivateKey(conf.CryptoKey)
		if err != nil {
			logger.Log.Error("Failed to load private key",
				zap.String("path", conf.CryptoKey),
				zap.Error(err))
		} else {
			logger.Log.Info("Private key loaded successfully",
				zap.String("path", conf.CryptoKey))
		}
	}

	return &ServerApp{
		config:         conf,
		shutdownSignal: make(chan struct{}),
		privateKey:     privateKey,
	}
}

// Run starts the server application.
// It initializes storage, starts the HTTP server, and handles shutdown signals.
// Returns an error if the application fails to start.
func (s *ServerApp) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.initStorage(ctx); err != nil {
		return err
	}

	s.metricsService = service.NewMetricsService(s.storage, s.db)

	go s.handleSignals(cancel)
	if err := s.startHTTPServer(); err != nil {
		return err
	}
	if err := s.startGRPCServer(); err != nil {
		return err
	}

	select {
	case <-s.shutdownSignal:
		logger.Log.Info("Shutdown signal received")
	case <-ctx.Done():
		logger.Log.Info("Context cancelled")
	}

	s.shutdown(ctx)
	logger.Log.Info("Server shutdown complete")
	return nil
}

func (s *ServerApp) initStorage(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config.DatabaseDSN != "" {
		logger.Log.Info("Connecting to db", zap.String("dsn", s.config.DatabaseDSN))

		db, err := sql.Open("postgres", s.config.DatabaseDSN)
		if err != nil {
			logger.Log.Fatal("Failed to connect to database", zap.Error(err))
			return err
		}
		s.db = db

		storageInstance, err := storage.NewPostgresStorageFromDB(ctx, db)
		if err != nil {
			logger.Log.Fatal("Failed to create Postgres storage", zap.Error(err))
			db.Close()
			return err
		}
		s.storage = storageInstance
		logger.Log.Info("Successfully connected to database")
	} else {
		memStorage := storage.NewMemStorage(s.config.FileStoragePath)

		if s.config.Restore {
			if err := memStorage.Load(); err != nil {
				logger.Log.Info("Could not load metrics from file", zap.Error(err))
			} else {
				logger.Log.Info("Metrics loaded from file", zap.String("path", s.config.FileStoragePath))
			}
		}

		s.storage = memStorage

		if s.config.StoreInterval > 0 {
			s.wg.Add(1)
			go s.runPeriodicSaver()
		} else {
			logger.Log.Info("Synchronous save mode enabled")
		}
	}
	return nil
}

func (s *ServerApp) runPeriodicSaver() {
	defer s.wg.Done()

	saveTicker := time.NewTicker(s.config.StoreInterval)
	defer saveTicker.Stop()

	for {
		select {
		case <-s.shutdownSignal:
			s.saveMetrics()
			return
		case <-saveTicker.C:
			s.saveMetrics()
		}
	}
}

func (s *ServerApp) saveMetrics() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if memStorage, ok := s.storage.(*storage.MemStorage); ok && memStorage != nil {
		if err := memStorage.Save(); err != nil {
			logger.Log.Error("Error saving metrics", zap.Error(err))
		} else {
			logger.Log.Debug("Metrics saved to file", zap.String("path", s.config.FileStoragePath))
		}
	}
}

func (s *ServerApp) handleSignals(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-sigChan
	logger.Log.Info("Received signal", zap.String("signal", sig.String()))
	close(s.shutdownSignal)
	cancel()
}

func (s *ServerApp) startHTTPServer() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := chi.NewRouter()
	r.Use(
		logger.RequestResponseLogger,
		middleware.GzipMiddleware,
		middleware.SyncSaveMiddleware(s.config.StoreInterval, s.storage),
		middleware.DecryptMiddleware(s.privateKey),
	)

	if s.config.TrustedSubnet != "" {
		r.Use(middleware.TrustedSubnetMiddleware(s.config.TrustedSubnet))
		logger.Log.Info("Trusted subnet protection enabled",
			zap.String("subnet", s.config.TrustedSubnet))
	}

	if s.config.Key != "" {
		r.Use(
			middleware.KeyContextMiddleware(s.config.Key),
			middleware.HashCheckMiddleware(s.config.Key),
		)
	}
	s.registerRoutes(r)

	s.httpServer = &http.Server{
		Addr:    s.config.ServerAddress,
		Handler: r,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		logger.Log.Info("Starting HTTP server", zap.String("address", s.config.ServerAddress))
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("HTTP server failed", zap.Error(err))
		}
	}()
	return nil
}

func (s *ServerApp) startGRPCServer() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	grpcHandlers := grpc_handlers.NewGRPCHandlers(s.metricsService, s.config.Key, s.privateKey)
	grpcInterceptor := middleware.NewGRPCInterceptor(
		s.privateKey,
		s.config.Key,
		s.config.TrustedSubnet,
		s.config.StoreInterval,
		s.storage,
	)

	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpcInterceptor.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpcInterceptor.StreamServerInterceptor()),
	}

	s.grpcServer = grpc.NewServer(serverOpts...)
	rpc.RegisterMonitoringServiceServer(s.grpcServer, grpcHandlers)

	listener, err := net.Listen("tcp", s.config.GRPCAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.GRPCAddress, err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		logger.Log.Info("Starting gRPC server")
		if err := s.grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Log.Error("gRPC server failed", zap.Error(err))
		}
	}()

	return nil
}

func (s *ServerApp) shutdown(ctx context.Context) {
	if s.config.StoreInterval > 0 && s.storage != nil {
		logger.Log.Info("Saving final metrics before shutdown")
		s.saveMetrics()
	}

	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error("HTTP server shutdown failed", zap.Error(err))
		} else {
			logger.Log.Info("HTTP server stopped successfully")
		}
	}

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
		logger.Log.Info("gRPC server stopped successfully")
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			logger.Log.Error("Failed to close database connection", zap.Error(err))
		}
	}
	s.wg.Wait()
}

func (s *ServerApp) registerRoutes(r *chi.Mux) {
	httpHandlers := http_handlers.NewHTTPHandlers(s.metricsService, s.config.Key)

	r.Post("/update/{type}/{name}/{value}", helpers.MethodCheck([]string{http.MethodPost})(httpHandlers.UpdateMetrics))
	r.Get("/value/{type}/{name}", helpers.MethodCheck([]string{http.MethodGet})(httpHandlers.GetValue))
	r.Post("/update/", helpers.MethodCheck([]string{http.MethodPost})(httpHandlers.UpdateMetrics))
	r.Post("/value/", helpers.MethodCheck([]string{http.MethodPost})(httpHandlers.GetValue))
	r.Get("/value/", helpers.MethodCheck([]string{http.MethodGet})(httpHandlers.GetValue))
	r.Get("/", helpers.MethodCheck([]string{http.MethodGet})(httpHandlers.GetMetricsList))
	r.Get("/ping", helpers.MethodCheck([]string{http.MethodGet})(httpHandlers.PingHandler))
	r.Post("/updates/", helpers.MethodCheck([]string{http.MethodPost})(httpHandlers.UpdateBatchMetrics))
}
