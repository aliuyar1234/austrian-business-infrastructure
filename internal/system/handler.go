package system

import (
	"net/http"
	"runtime"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
)

var (
	// Version is set at build time
	Version = "dev"
	// BuildTime is set at build time
	BuildTime = "unknown"
	// GitCommit is set at build time
	GitCommit = "unknown"
)

var startTime = time.Now()

// Handler handles system HTTP requests
type Handler struct {
	metrics *Metrics
}

// NewHandler creates a new system handler
func NewHandler(metrics *Metrics) *Handler {
	return &Handler{metrics: metrics}
}

// RegisterRoutes registers system routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	router.Handle("GET /api/v1/system/info", requireAuth(requireAdmin(http.HandlerFunc(h.Info))))
	router.Handle("GET /api/v1/system/metrics", requireAuth(requireAdmin(http.HandlerFunc(h.GetMetrics))))
}

// SystemInfo represents system information
type SystemInfo struct {
	Version    string         `json:"version"`
	BuildTime  string         `json:"build_time"`
	GitCommit  string         `json:"git_commit"`
	Uptime     string         `json:"uptime"`
	UptimeSecs int64          `json:"uptime_secs"`
	GoVersion  string         `json:"go_version"`
	OS         string         `json:"os"`
	Arch       string         `json:"arch"`
	NumCPU     int            `json:"num_cpu"`
	Memory     *MemoryStats   `json:"memory"`
}

// MemoryStats represents memory statistics
type MemoryStats struct {
	Alloc      uint64 `json:"alloc_bytes"`
	TotalAlloc uint64 `json:"total_alloc_bytes"`
	Sys        uint64 `json:"sys_bytes"`
	NumGC      uint32 `json:"num_gc"`
}

// Info handles GET /api/v1/system/info
func (h *Handler) Info(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(startTime)

	info := SystemInfo{
		Version:    Version,
		BuildTime:  BuildTime,
		GitCommit:  GitCommit,
		Uptime:     formatDuration(uptime),
		UptimeSecs: int64(uptime.Seconds()),
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
		Memory: &MemoryStats{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
	}

	api.JSONResponse(w, http.StatusOK, info)
}

// MetricsResponse represents the metrics response
type MetricsResponse struct {
	Requests       *RequestMetrics `json:"requests"`
	ActiveSessions int64           `json:"active_sessions"`
}

// RequestMetrics represents request metrics
type RequestMetrics struct {
	Total       int64   `json:"total"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	ErrorRate   float64 `json:"error_rate"`
}

// GetMetrics handles GET /api/v1/system/metrics
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if h.metrics == nil {
		api.JSONResponse(w, http.StatusOK, MetricsResponse{
			Requests: &RequestMetrics{},
		})
		return
	}

	api.JSONResponse(w, http.StatusOK, MetricsResponse{
		Requests: &RequestMetrics{
			Total:       h.metrics.RequestCount(),
			AvgLatencyMs: h.metrics.AverageLatency(),
			ErrorRate:   h.metrics.ErrorRate(),
		},
		ActiveSessions: h.metrics.ActiveSessions(),
	})
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return formatInt(days) + "d " + formatInt(hours) + "h " + formatInt(minutes) + "m"
	}
	if hours > 0 {
		return formatInt(hours) + "h " + formatInt(minutes) + "m " + formatInt(seconds) + "s"
	}
	if minutes > 0 {
		return formatInt(minutes) + "m " + formatInt(seconds) + "s"
	}
	return formatInt(seconds) + "s"
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	var s string
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
