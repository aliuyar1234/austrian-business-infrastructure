package api

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
)

// Context keys for request-scoped values
type contextKey string

const (
	RequestIDKey  contextKey = "request_id"
	UserIDKey     contextKey = "user_id"
	TenantIDKey   contextKey = "tenant_id"
	UserRoleKey   contextKey = "user_role"
	UserEmailKey  contextKey = "user_email"
)

// Middleware represents a middleware function
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares in order
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// RequestID adds a unique request ID to each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger logs request details
func Logger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			requestID, _ := r.Context().Value(RequestIDKey).(string)

			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
				"request_id", requestID,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

// Recovery recovers from panics and returns 500 error
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID, _ := r.Context().Value(RequestIDKey).(string)

					logger.Error("panic recovered",
						"error", err,
						"request_id", requestID,
						"stack", string(debug.Stack()),
					)

					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// CORS adds CORS headers
func CORS(allowedOrigins []string) Middleware {
	originsMap := make(map[string]bool)
	allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	for _, origin := range allowedOrigins {
		originsMap[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Set Vary header to prevent cache poisoning when origin varies
			w.Header().Add("Vary", "Origin")

			if origin != "" && (originsMap[origin] || allowAll) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				// Only allow credentials for explicitly listed origins, not wildcard
				if !allowAll {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID, X-API-Key")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders adds security headers
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// HSTS - enforce HTTPS for 1 year, include subdomains
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		next.ServeHTTP(w, r)
	})
}

// CSPConfig holds Content Security Policy configuration
type CSPConfig struct {
	// DefaultSrc sets default-src directive (fallback for other directives)
	DefaultSrc []string
	// ScriptSrc sets script-src directive
	ScriptSrc []string
	// StyleSrc sets style-src directive
	StyleSrc []string
	// ImgSrc sets img-src directive
	ImgSrc []string
	// ConnectSrc sets connect-src directive (for fetch/XHR/WebSocket)
	ConnectSrc []string
	// FontSrc sets font-src directive
	FontSrc []string
	// ObjectSrc sets object-src directive
	ObjectSrc []string
	// FrameAncestors sets frame-ancestors directive (prevents clickjacking)
	FrameAncestors []string
	// BaseURI sets base-uri directive
	BaseURI []string
	// FormAction sets form-action directive
	FormAction []string
	// ReportURI sets report-uri directive for CSP violation reports
	ReportURI string
	// ReportOnly if true, uses Content-Security-Policy-Report-Only header
	ReportOnly bool
}

// DefaultCSPConfig returns a strict default CSP configuration
func DefaultCSPConfig() *CSPConfig {
	return &CSPConfig{
		DefaultSrc:     []string{"'self'"},
		ScriptSrc:      []string{"'self'"},
		StyleSrc:       []string{"'self'", "'unsafe-inline'"}, // unsafe-inline often needed for styling
		ImgSrc:         []string{"'self'", "data:", "blob:"},
		ConnectSrc:     []string{"'self'"},
		FontSrc:        []string{"'self'"},
		ObjectSrc:      []string{"'none'"},
		FrameAncestors: []string{"'none'"}, // Prevent framing (clickjacking protection)
		BaseURI:        []string{"'self'"},
		FormAction:     []string{"'self'"},
		ReportOnly:     false,
	}
}

// ContentSecurityPolicy adds CSP headers for XSS defense in depth
func ContentSecurityPolicy(config *CSPConfig) Middleware {
	if config == nil {
		config = DefaultCSPConfig()
	}

	// Build CSP header value
	directives := []string{}

	if len(config.DefaultSrc) > 0 {
		directives = append(directives, "default-src "+joinCSPValues(config.DefaultSrc))
	}
	if len(config.ScriptSrc) > 0 {
		directives = append(directives, "script-src "+joinCSPValues(config.ScriptSrc))
	}
	if len(config.StyleSrc) > 0 {
		directives = append(directives, "style-src "+joinCSPValues(config.StyleSrc))
	}
	if len(config.ImgSrc) > 0 {
		directives = append(directives, "img-src "+joinCSPValues(config.ImgSrc))
	}
	if len(config.ConnectSrc) > 0 {
		directives = append(directives, "connect-src "+joinCSPValues(config.ConnectSrc))
	}
	if len(config.FontSrc) > 0 {
		directives = append(directives, "font-src "+joinCSPValues(config.FontSrc))
	}
	if len(config.ObjectSrc) > 0 {
		directives = append(directives, "object-src "+joinCSPValues(config.ObjectSrc))
	}
	if len(config.FrameAncestors) > 0 {
		directives = append(directives, "frame-ancestors "+joinCSPValues(config.FrameAncestors))
	}
	if len(config.BaseURI) > 0 {
		directives = append(directives, "base-uri "+joinCSPValues(config.BaseURI))
	}
	if len(config.FormAction) > 0 {
		directives = append(directives, "form-action "+joinCSPValues(config.FormAction))
	}
	if config.ReportURI != "" {
		directives = append(directives, "report-uri "+config.ReportURI)
	}

	cspValue := ""
	for i, d := range directives {
		if i > 0 {
			cspValue += "; "
		}
		cspValue += d
	}

	headerName := "Content-Security-Policy"
	if config.ReportOnly {
		headerName = "Content-Security-Policy-Report-Only"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerName, cspValue)
			next.ServeHTTP(w, r)
		})
	}
}

func joinCSPValues(values []string) string {
	result := ""
	for i, v := range values {
		if i > 0 {
			result += " "
		}
		result += v
	}
	return result
}

// ContentType sets the Content-Type header for JSON responses
func ContentType(contentType string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Helper functions for context values

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// GetTenantID retrieves tenant ID from context
func GetTenantID(ctx context.Context) string {
	if id, ok := ctx.Value(TenantIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserRole retrieves user role from context
func GetUserRole(ctx context.Context) string {
	if role, ok := ctx.Value(UserRoleKey).(string); ok {
		return role
	}
	return ""
}

// GetUserEmail retrieves user email from context
func GetUserEmail(ctx context.Context) string {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}
