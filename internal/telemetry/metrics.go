package telemetry

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/alexandrmotologa/geogate/internal/geoip"
)

// MetricsHandler exports runtime and application metrics in standard Prometheus text format.
type MetricsHandler struct {
	engine    *geoip.Engine
	startTime time.Time
}

// NewMetricsHandler initializes a metrics handler.
func NewMetricsHandler(engine *geoip.Engine, startTime time.Time) *MetricsHandler {
	return &MetricsHandler{
		engine:    engine,
		startTime: startTime,
	}
}

// ServeHTTP writes the Prometheus metrics exposition format.
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(h.startTime).Seconds()
	totalLookups := h.engine.TotalLookups()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintf(w, "# HELP geogate_lookups_total Total number of IP lookups performed.\n")
	fmt.Fprintf(w, "# TYPE geogate_lookups_total counter\n")
	fmt.Fprintf(w, "geogate_lookups_total %d\n\n", totalLookups)

	fmt.Fprintf(w, "# HELP geogate_uptime_seconds Total runtime in seconds.\n")
	fmt.Fprintf(w, "# TYPE geogate_uptime_seconds gauge\n")
	fmt.Fprintf(w, "geogate_uptime_seconds %.1f\n\n", uptime)

	fmt.Fprintf(w, "# HELP geogate_memory_alloc_bytes Current allocated heap memory in bytes.\n")
	fmt.Fprintf(w, "# TYPE geogate_memory_alloc_bytes gauge\n")
	fmt.Fprintf(w, "geogate_memory_alloc_bytes %d\n\n", m.Alloc)

	fmt.Fprintf(w, "# HELP geogate_memory_sys_bytes Total system memory obtained from OS.\n")
	fmt.Fprintf(w, "# TYPE geogate_memory_sys_bytes gauge\n")
	fmt.Fprintf(w, "geogate_memory_sys_bytes %d\n\n", m.Sys)

	fmt.Fprintf(w, "# HELP geogate_goroutines_count Current number of active goroutines.\n")
	fmt.Fprintf(w, "# TYPE geogate_goroutines_count gauge\n")
	fmt.Fprintf(w, "geogate_goroutines_count %d\n\n", runtime.NumGoroutine())
}
