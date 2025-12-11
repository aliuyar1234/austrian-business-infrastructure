package constants

import "time"

// HTTP Client Timeouts
const (
	// DefaultHTTPTimeout is the default timeout for HTTP requests.
	DefaultHTTPTimeout = 30 * time.Second

	// AIClientTimeout is the timeout for AI/LLM API calls.
	// Set to 60 seconds to allow for longer model responses.
	AIClientTimeout = 60 * time.Second

	// OAuthTimeout is the timeout for OAuth token exchange operations.
	OAuthTimeout = 10 * time.Second

	// HealthCheckTimeout is the timeout for health check operations.
	HealthCheckTimeout = 5 * time.Second
)

// WebSocket Timeouts
const (
	// WebSocketWriteWait is the time allowed to write a message to the peer.
	WebSocketWriteWait = 10 * time.Second

	// WebSocketPongWait is the time allowed to read the next pong message from the peer.
	WebSocketPongWait = 60 * time.Second

	// WebSocketPingPeriod is the send ping period (must be less than PongWait).
	WebSocketPingPeriod = 30 * time.Second
)

// Authentication Timeouts
const (
	// TwoFactorChallengeTTL is the time-to-live for 2FA challenge tokens.
	TwoFactorChallengeTTL = 5 * time.Minute

	// OAuthStateTTL is the time-to-live for OAuth state tokens.
	OAuthStateTTL = 10 * time.Minute

	// AccessTokenExpiry is the default expiry time for access tokens.
	AccessTokenExpiry = 15 * time.Minute

	// RefreshTokenExpiry is the default expiry time for refresh tokens.
	RefreshTokenExpiry = 7 * 24 * time.Hour
)

// Rate Limiting Windows
const (
	// LoginRateLimitWindow is the time window for login rate limiting.
	LoginRateLimitWindow = time.Minute
)

// Expiry Times in Seconds (for API responses)
const (
	// TwoFactorChallengeExpirySeconds is the 2FA challenge expiry in seconds.
	TwoFactorChallengeExpirySeconds = 300 // 5 minutes

	// AccessTokenExpirySeconds is the access token expiry in seconds.
	AccessTokenExpirySeconds = 900 // 15 minutes
)
