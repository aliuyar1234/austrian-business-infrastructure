package fonws

import (
	"errors"
	"fmt"
)

// Error codes from FinanzOnline WebService
const (
	ErrCodeNone              = 0
	ErrCodeSessionExpired    = -1
	ErrCodeMaintenance       = -2
	ErrCodeTechnical         = -3
	ErrCodeInvalidCredentials = -4
	ErrCodeUserLockedTemp    = -5
	ErrCodeUserLockedPerm    = -6
	ErrCodeNotWebServiceUser = -7
	ErrCodeParticipantLocked = -8
)

// FOError represents a FinanzOnline WebService error
type FOError struct {
	Code    int
	Message string
}

func (e *FOError) Error() string {
	return e.Message
}

// Common errors
var (
	ErrSessionExpired    = errors.New("session expired")
	ErrMaintenance       = errors.New("service under maintenance")
	ErrTechnical         = errors.New("technical error")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserLockedTemp    = errors.New("user temporarily locked")
	ErrUserLockedPerm    = errors.New("user permanently locked")
	ErrNotWebServiceUser = errors.New("not a WebService user")
	ErrParticipantLocked = errors.New("participant locked")
	ErrNoActiveSession   = errors.New("no active session")
)

// errorMessages maps error codes to human-readable messages
var errorMessages = map[int]string{
	ErrCodeNone:              "",
	ErrCodeSessionExpired:    "Session expired. Please log in again.",
	ErrCodeMaintenance:       "FinanzOnline is under maintenance. Try again later.",
	ErrCodeTechnical:         "Technical error. Please try again later.",
	ErrCodeInvalidCredentials: "Invalid credentials. Check Teilnehmer-ID, Benutzer-ID, and PIN.",
	ErrCodeUserLockedTemp:    "User temporarily locked. Too many failed attempts.",
	ErrCodeUserLockedPerm:    "User permanently locked. Contact FinanzOnline support.",
	ErrCodeNotWebServiceUser: "Not a WebService user. Enable WebService access in FinanzOnline.",
	ErrCodeParticipantLocked: "Participant locked. Contact FinanzOnline support.",
}

// GetErrorMessage returns a human-readable error message for a FinanzOnline error code
func GetErrorMessage(code int) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return fmt.Sprintf("Unknown error (code %d)", code)
}

// NewFOError creates a FOError from a response code
func NewFOError(code int, serverMsg string) error {
	if code == ErrCodeNone {
		return nil
	}

	msg := GetErrorMessage(code)
	if serverMsg != "" {
		msg = fmt.Sprintf("%s (%s)", msg, serverMsg)
	}

	return &FOError{
		Code:    code,
		Message: msg,
	}
}

// CheckResponse checks a response code and returns an error if non-zero
func CheckResponse(code int, serverMsg string) error {
	return NewFOError(code, serverMsg)
}

// IsSessionExpired returns true if the error indicates session expiration
func IsSessionExpired(err error) bool {
	if err == nil {
		return false
	}
	var foErr *FOError
	if errors.As(err, &foErr) {
		return foErr.Code == ErrCodeSessionExpired
	}
	return errors.Is(err, ErrSessionExpired)
}

// IsRetryable returns true if the error is temporary and the operation can be retried
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var foErr *FOError
	if errors.As(err, &foErr) {
		switch foErr.Code {
		case ErrCodeSessionExpired, ErrCodeMaintenance, ErrCodeTechnical:
			return true
		}
	}
	return false
}
