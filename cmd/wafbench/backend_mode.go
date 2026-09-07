package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func runBackend(args []string) error {
	fs := flag.NewFlagSet("backend", flag.ContinueOnError)
	listen := fs.String("listen", "127.0.0.1:18081", "listen address")
	responseBytes := fs.Int("response-bytes", 16384, "response body size")
	contentType := fs.String("content-type", "application/json", "response Content-Type")
	delay := fs.Duration("delay", 0, "optional deterministic backend delay")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *responseBytes < 0 {
		return fmt.Errorf("--response-bytes must be >= 0")
	}
	body := sizedJSON(*responseBytes, "backend")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *delay > 0 {
			time.Sleep(*delay)
		}
		w.Header().Set("Content-Type", *contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("X-WAFBench-Backend", "1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
	srv := &http.Server{Addr: *listen, Handler: h, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 120 * time.Second}
	log.Printf("wafbench backend listening on %s, response=%d bytes", *listen, len(body))
	return srv.ListenAndServe()
}
