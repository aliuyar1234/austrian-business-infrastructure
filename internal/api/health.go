package api

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// HealthChecker provides health check functionality
type HealthChecker interface {
	Health(ctx context.Context) error
}

// HealthService manages health checks
type HealthService struct {
	checks map[string]HealthChecker
	mu     sync.RWMutex
}

// NewHealthService creates a new health service
func NewHealthService() *HealthService {
	return &HealthService{
		checks: make(map[string]HealthChecker),
	}
}

// Register registers a health checker
func (s *HealthService) Register(name string, checker HealthChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[name] = checker
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string                 `json:"status"`
	Checks map[string]CheckResult `json:"checks,omitempty"`
}

// LivenessHandler returns the liveness probe handler
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	}
}

// ReadinessHandler returns the readiness probe handler
func (s *HealthService) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		s.mu.RLock()
		defer s.mu.RUnlock()

		response := HealthResponse{
			Status: "ready",
			Checks: make(map[string]CheckResult),
		}

		allHealthy := true

		for name, checker := range s.checks {
			if err := checker.Health(ctx); err != nil {
				response.Checks[name] = CheckResult{
					Status:  "unhealthy",
					Message: err.Error(),
				}
				allHealthy = false
			} else {
				response.Checks[name] = CheckResult{
					Status: "healthy",
				}
			}
		}

		if !allHealthy {
			response.Status = "not_ready"
			JSONResponse(w, http.StatusServiceUnavailable, response)
			return
		}

		JSONResponse(w, http.StatusOK, response)
	}
}
