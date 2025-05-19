package helpers

import (
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/storage"
)

func MethodCheck(methods []string) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			for _, method := range methods {
				if r.Method == method {
					next(w, r)
					return
				}
			}
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func FormatFloat(value float64) string {
	formatted := fmt.Sprintf("%.3f", value)
	return strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
}

func CheckAndSaveMetrics(storage *storage.MemStorage, prevGauges map[string]float64, prevCounters map[string]int64) {
	currentGauges := storage.Gauges()
	currentCounters := storage.Counters()

	gaugeChanged := len(prevGauges) != len(currentGauges)
	if !gaugeChanged {
		for k, v := range currentGauges {
			if prevGauges[k] != v {
				gaugeChanged = true
				break
			}
		}
	}

	counterChanged := len(prevCounters) != len(currentCounters)
	if !counterChanged {
		for k, v := range currentCounters {
			if prevCounters[k] != v {
				counterChanged = true
				break
			}
		}
	}
	if gaugeChanged || counterChanged {
		if err := storage.Save(); err != nil {
			logger.Log.Error("Error saving metrics", zap.Error(err))
		}
	}
}
