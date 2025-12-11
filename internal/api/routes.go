package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

// Router wraps http.ServeMux with additional functionality
type Router struct {
	mux         *http.ServeMux
	middlewares []Middleware
	logger      *slog.Logger
}

// NewRouter creates a new router
func NewRouter(logger *slog.Logger) *Router {
	return &Router{
		mux:    http.NewServeMux(),
		logger: logger,
	}
}

// Use adds middleware to the router
func (r *Router) Use(mw Middleware) {
	r.middlewares = append(r.middlewares, mw)
}

// Handle registers a handler for a pattern
func (r *Router) Handle(pattern string, handler http.Handler) {
	r.mux.Handle(pattern, handler)
}

// HandleFunc registers a handler function for a pattern
func (r *Router) HandleFunc(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, handler)
}

// Group creates a route group with shared prefix and middlewares
func (r *Router) Group(prefix string, middlewares ...Middleware) *RouteGroup {
	return &RouteGroup{
		router:      r,
		prefix:      prefix,
		middlewares: middlewares,
	}
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler := http.Handler(r.mux)
	handler = Chain(handler, r.middlewares...)
	handler.ServeHTTP(w, req)
}

// RouteGroup represents a group of routes with shared prefix
type RouteGroup struct {
	router      *Router
	prefix      string
	middlewares []Middleware
}

// Handle registers a handler in the group
func (g *RouteGroup) Handle(pattern string, handler http.Handler) {
	fullPattern := g.prefix + pattern
	wrapped := Chain(handler, g.middlewares...)
	g.router.mux.Handle(fullPattern, wrapped)
}

// HandleFunc registers a handler function in the group
func (g *RouteGroup) HandleFunc(pattern string, handler http.HandlerFunc) {
	g.Handle(pattern, handler)
}

// JSON response helpers

// JSONResponse sends a JSON response
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// JSONError sends a JSON error response
func JSONError(w http.ResponseWriter, status int, message string, code string) {
	JSONResponse(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// JSONErrorWithDetails sends a JSON error response with details
func JSONErrorWithDetails(w http.ResponseWriter, status int, message string, code string, details map[string]string) {
	JSONResponse(w, status, ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}

// Common error codes
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeValidation          = "VALIDATION_ERROR"
	ErrCodeInternalError       = "INTERNAL_ERROR"
	ErrCodeRateLimited         = "RATE_LIMITED"
	ErrCodeInvalidCredentials  = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired        = "TOKEN_EXPIRED"
	ErrCodeInvalidToken        = "INVALID_TOKEN"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
)

// Standard error responses

// BadRequest sends a 400 response
func BadRequest(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusBadRequest, message, ErrCodeBadRequest)
}

// Unauthorized sends a 401 response
func Unauthorized(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusUnauthorized, message, ErrCodeUnauthorized)
}

// Forbidden sends a 403 response
func Forbidden(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusForbidden, message, ErrCodeForbidden)
}

// NotFound sends a 404 response
func NotFound(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusNotFound, message, ErrCodeNotFound)
}

// Conflict sends a 409 response
func Conflict(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusConflict, message, ErrCodeConflict)
}

// InternalError sends a 500 response
func InternalError(w http.ResponseWriter) {
	JSONError(w, http.StatusInternalServerError, "Internal server error", ErrCodeInternalError)
}

// ValidationError sends a 400 response with validation details
func ValidationError(w http.ResponseWriter, details map[string]string) {
	JSONErrorWithDetails(w, http.StatusBadRequest, "Validation failed", ErrCodeValidation, details)
}

// RateLimited sends a 429 response
func RateLimited(w http.ResponseWriter, retryAfter int) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	JSONError(w, http.StatusTooManyRequests, "Rate limit exceeded", ErrCodeRateLimited)
}
