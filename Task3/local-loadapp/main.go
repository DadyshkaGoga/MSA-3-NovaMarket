// Local arm64 stand-in for ghcr.io/yandex-practicum/scaletestapp (the official image is amd64-only
// and won't run on Apple Silicon). Same endpoints: GET / returns the pod id and bumps the counter,
// GET /metrics exposes http_requests_total. The production deployment.yaml uses the official image.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
)

var (
	requests int64
	mu       sync.Mutex
	ballast  [][]byte // held memory grows with load and drives the memory HPA
)

func main() {
	host, _ := os.Hostname()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requests, 1)
		// ~64 KiB per request, capped near 20 MiB: crosses the 80% target (of the 20Mi request)
		// but stays under the 30Mi limit, so it scales without OOM.
		mu.Lock()
		if len(ballast) < 320 {
			ballast = append(ballast, make([]byte, 64*1024))
		}
		mu.Unlock()
		fmt.Fprintf(w, "pod=%s requests=%d\n", host, atomic.LoadInt64(&requests))
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# HELP http_requests_total Total HTTP requests to /\n")
		fmt.Fprintf(w, "# TYPE http_requests_total counter\n")
		fmt.Fprintf(w, "http_requests_total %d\n", atomic.LoadInt64(&requests))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
