// Local arm64-compatible stand-in for ghcr.io/yandex-practicum/scaletestapp.
// Needed only for the live run on Apple Silicon: the official image is amd64-only and won't
// run on an arm64 cluster without emulation. Production manifests (deployment.yaml) reference
// the official image; this is a functional equivalent used to reproduce autoscaling locally.
//
// Same endpoints as the original:
//
//	GET /        -> pod id (hostname); increments http_requests_total; holds some memory
//	GET /metrics -> http_requests_total in Prometheus text format
package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
)

var (
	requests int64
	mu       sync.Mutex
	ballast  [][]byte // retained memory -> utilization grows under load (drives the memory HPA)
)

func main() {
	host, _ := os.Hostname()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requests, 1)
		// Each request retains ~64 KiB, capped at ~20 MiB total, so utilization rises above the
		// 80% threshold (16Mi of the 20Mi request) while staying under the 30Mi limit (no OOM).
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

	http.ListenAndServe(":8080", nil)
}
