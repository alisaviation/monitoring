package middleware

import (
	"compress/gzip"
	"io"
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

func SyncSaveMiddleware(storeInterval time.Duration, storage *storage.MemStorage) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				var prevGauges map[string]float64
				var prevCounters map[string]int64

				if storeInterval == 0 {
					prevGauges = make(map[string]float64)
					prevCounters = make(map[string]int64)

					// Обработка значений и ошибок
					gauges, err := storage.Gauges()
					if err != nil {
						// Обработка ошибки, если это необходимо
						// Например, можно записать ошибку в лог
						// log.Println("Error getting gauges:", err)
					} else {
						for k, v := range gauges {
							prevGauges[k] = v
						}
					}

					// Аналогично для счетчиков
					counters, err := storage.Counters()
					if err != nil {
						// Обработка ошибки, если это необходимо
						// log.Println("Error getting counters:", err)
					} else {
						for k, v := range counters {
							prevCounters[k] = v
						}
					}
					//for k, v := range storage.Gauges() {
					//	prevGauges[k] = v
					//}
					//for k, v := range storage.Counters() {
					//	prevCounters[k] = v
					//}
				}
				ww := &responseWriterWrapper{
					ResponseWriter: w,
					onWriteHeader: func() {
						if storeInterval == 0 {
							helpers.CheckAndSaveMetrics(storage, prevGauges, prevCounters)
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
