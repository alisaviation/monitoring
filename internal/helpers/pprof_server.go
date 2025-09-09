package helpers

import (
	"log"
	"net/http"
	"net/http/pprof"
)

func StartPProfServer(addr string) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		mux.Handle("/debug/pprof/heap", http.HandlerFunc(pprof.Handler("heap").ServeHTTP))
		mux.Handle("/debug/pprof/goroutine", http.HandlerFunc(pprof.Handler("goroutine").ServeHTTP))

		server := &http.Server{
			Addr:    addr,
			Handler: mux,
		}
		log.Println(server.ListenAndServe())
	}()
}
