package helpers

import (
	"fmt"
	"net/http"
	"strings"
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
