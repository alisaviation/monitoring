package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.Config{
		Level:       lvl,
		Development: false,
		Sampling:    nil,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	Log, err = cfg.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	return nil
}

func RequestResponseLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		headers := make([]zap.Field, 0, 4)
		if v := r.Header.Get("User-Agent"); v != "" {
			headers = append(headers, zap.String("user_agent", v))
		}
		if v := r.Header.Get("X-Forwarded-For"); v != "" {
			headers = append(headers, zap.String("x_forwarded_for", v))
		}
		if v := r.Header.Get("Content-Type"); v != "" {
			headers = append(headers, zap.String("content_type", v))
		}

		ww := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		fields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.statusCode),
			zap.Int("size", ww.size),
			zap.Duration("duration", duration),
			zap.String("hash_header", r.Header.Get("HashSHA256")),
			zap.String("response_hash", ww.Header().Get("HashSHA256")),
		}
		fields = append(fields, headers...)

		Log.Info("HTTP request", fields...)
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
