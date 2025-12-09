package system

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects system metrics
type Metrics struct {
	requestCount   int64
	errorCount     int64
	totalLatencyNs int64
	activeSessions int64

	// For rolling window calculations
	mu             sync.Mutex
	recentRequests []requestRecord
	windowSize     time.Duration
}

type requestRecord struct {
	timestamp time.Time
	latencyNs int64
	isError   bool
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		windowSize:     5 * time.Minute,
		recentRequests: make([]requestRecord, 0, 1000),
	}
}

// RecordRequest records a request
func (m *Metrics) RecordRequest(latency time.Duration, isError bool) {
	atomic.AddInt64(&m.requestCount, 1)
	atomic.AddInt64(&m.totalLatencyNs, int64(latency))

	if isError {
		atomic.AddInt64(&m.errorCount, 1)
	}

	// Add to rolling window
	m.mu.Lock()
	m.recentRequests = append(m.recentRequests, requestRecord{
		timestamp: time.Now(),
		latencyNs: int64(latency),
		isError:   isError,
	})
	m.cleanup()
	m.mu.Unlock()
}

// cleanup removes old records outside the window
func (m *Metrics) cleanup() {
	cutoff := time.Now().Add(-m.windowSize)
	i := 0
	for ; i < len(m.recentRequests); i++ {
		if m.recentRequests[i].timestamp.After(cutoff) {
			break
		}
	}
	if i > 0 {
		m.recentRequests = m.recentRequests[i:]
	}
}

// RequestCount returns total request count
func (m *Metrics) RequestCount() int64 {
	return atomic.LoadInt64(&m.requestCount)
}

// ErrorCount returns total error count
func (m *Metrics) ErrorCount() int64 {
	return atomic.LoadInt64(&m.errorCount)
}

// AverageLatency returns average latency in milliseconds
func (m *Metrics) AverageLatency() float64 {
	count := atomic.LoadInt64(&m.requestCount)
	if count == 0 {
		return 0
	}
	totalNs := atomic.LoadInt64(&m.totalLatencyNs)
	return float64(totalNs) / float64(count) / 1e6
}

// ErrorRate returns the error rate as a percentage
func (m *Metrics) ErrorRate() float64 {
	count := atomic.LoadInt64(&m.requestCount)
	if count == 0 {
		return 0
	}
	errors := atomic.LoadInt64(&m.errorCount)
	return float64(errors) / float64(count) * 100
}

// RecentAverageLatency returns average latency in the recent window
func (m *Metrics) RecentAverageLatency() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanup()

	if len(m.recentRequests) == 0 {
		return 0
	}

	var total int64
	for _, r := range m.recentRequests {
		total += r.latencyNs
	}
	return float64(total) / float64(len(m.recentRequests)) / 1e6
}

// RecentErrorRate returns error rate in the recent window
func (m *Metrics) RecentErrorRate() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanup()

	if len(m.recentRequests) == 0 {
		return 0
	}

	var errors int
	for _, r := range m.recentRequests {
		if r.isError {
			errors++
		}
	}
	return float64(errors) / float64(len(m.recentRequests)) * 100
}

// SetActiveSessions sets the active session count
func (m *Metrics) SetActiveSessions(count int64) {
	atomic.StoreInt64(&m.activeSessions, count)
}

// IncrementActiveSessions increments active session count
func (m *Metrics) IncrementActiveSessions() {
	atomic.AddInt64(&m.activeSessions, 1)
}

// DecrementActiveSessions decrements active session count
func (m *Metrics) DecrementActiveSessions() {
	atomic.AddInt64(&m.activeSessions, -1)
}

// ActiveSessions returns active session count
func (m *Metrics) ActiveSessions() int64 {
	return atomic.LoadInt64(&m.activeSessions)
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.requestCount, 0)
	atomic.StoreInt64(&m.errorCount, 0)
	atomic.StoreInt64(&m.totalLatencyNs, 0)
	atomic.StoreInt64(&m.activeSessions, 0)

	m.mu.Lock()
	m.recentRequests = m.recentRequests[:0]
	m.mu.Unlock()
}
