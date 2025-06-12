package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zl
	return nil
}

func RequestResponseLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		ww := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		responseHeaders := make(map[string]string)
		for k, v := range ww.Header() {
			if len(v) > 0 {
				responseHeaders[k] = v[0]
			}
		}

		Log.Info("HTTP request handled",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.statusCode),
			zap.Int("size", ww.size),
			zap.Duration("duration", duration),
			zap.Any("request_headers", headers),
			zap.Any("response_headers", responseHeaders),
			zap.String("hash_header", r.Header.Get("HashSHA256")),
			zap.String("response_hash", ww.Header().Get("HashSHA256")),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
	headers    http.Header
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

func (rw *responseWriter) Header() http.Header {
	if rw.headers == nil {
		rw.headers = make(http.Header)
	}
	return rw.ResponseWriter.Header()
}
