package helpers

import (
	"fmt"
	"net/http"
)

func WriteResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)

	switch v := body.(type) {
	case float64:
		w.Write([]byte(fmt.Sprintf("%.2f", v)))
	case int64:
		w.Write([]byte(fmt.Sprintf("%d", v)))
	default:
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
func MethodCheck(methods []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, method := range methods {
				if r.Method == method {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		})
	}
}
