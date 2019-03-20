package tracing

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var prometheusRunning bool

// StartPrometheus is a blocking function which starts serving the prometheus
// metrics at /metrics. If you specify nil or 0 for the port, the port
// will default to 2112
func StartPrometheus(port int) {
	if prometheusRunning {
		return
	}

	if port == 0 {
		port = 2112
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	s := &http.Server{
		Addr:           ":" + strconv.Itoa(port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	prometheusRunning = true
	s.ListenAndServe()
}
