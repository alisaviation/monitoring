package middleware

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alisaviation/monitoring/internal/helpers"
	"github.com/alisaviation/monitoring/internal/storage"
)

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		if acceptsGzip {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			w = gzipWriter{ResponseWriter: w, Writer: gz}
		}
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "invalid gzip data", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		next.ServeHTTP(w, r)
	})
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (g gzipWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}

func SyncSaveMiddleware(storeInterval time.Duration, storage storage.Storage) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				var prevGauges map[string]float64
				var prevCounters map[string]int64

				if storeInterval == 0 {
					prevGauges = make(map[string]float64)
					prevCounters = make(map[string]int64)

					gauges, err := storage.Gauges(r.Context())
					if err != nil {
						log.Println("Error getting gauges:", err)
					} else {
						for k, v := range gauges {
							prevGauges[k] = v
						}
					}

					counters, err := storage.Counters(r.Context())
					if err != nil {
						log.Println("Error getting counters:", err)
					} else {
						for k, v := range counters {
							prevCounters[k] = v
						}
					}

				}
				ww := &responseWriterWrapper{
					ResponseWriter: w,
					onWriteHeader: func() {
						if storeInterval == 0 {
							helpers.CheckAndSaveMetrics(r.Context(), storage, prevGauges, prevCounters)
						}
					},
				}
				next.ServeHTTP(ww, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type responseWriterWrapper struct {
	http.ResponseWriter
	onWriteHeader func()
	headerWritten bool
	mu            sync.Mutex
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.headerWritten {
		w.headerWritten = true
		if w.onWriteHeader != nil {
			w.onWriteHeader()
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	w.mu.Lock()
	if !w.headerWritten {
		w.headerWritten = true
		if w.onWriteHeader != nil {
			w.onWriteHeader()
		}
	}
	w.mu.Unlock()
	return w.ResponseWriter.Write(b)
}

func HashCheckMiddleware(key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			hashHeader := r.Header.Get("HashSHA256")
			if hashHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad Request: cannot read body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			computedHash := helpers.CalculateHash(body, key)
			if computedHash != hashHeader {
				http.Error(w, "Bad Request: invalid hash", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func KeyContextMiddleware(key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key != "" {
				ctx := context.WithValue(r.Context(), "key", key)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}
