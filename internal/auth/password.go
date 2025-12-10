package auth

import (
	"austrian-business-infrastructure/pkg/crypto"
)

// Re-export password functions from crypto package for backwards compatibility
var (
	ErrPasswordTooShort    = crypto.ErrPasswordTooShort
	ErrPasswordNoUppercase = crypto.ErrPasswordNoUppercase
	ErrPasswordNoLowercase = crypto.ErrPasswordNoLowercase
	ErrPasswordNoDigit     = crypto.ErrPasswordNoDigit
	ErrPasswordInvalid     = crypto.ErrPasswordInvalid
)

// PasswordPolicy is an alias for crypto.PasswordPolicy
type PasswordPolicy = crypto.PasswordPolicy

// DefaultPasswordPolicy returns the default password policy
func DefaultPasswordPolicy() *PasswordPolicy {
	return crypto.DefaultPasswordPolicy()
}

// ValidatePassword checks if a password meets the policy requirements
func ValidatePassword(password string, policy *PasswordPolicy) error {
	return crypto.ValidatePassword(password, policy)
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	return crypto.HashPassword(password)
}

// VerifyPassword checks if a password matches a hash
func VerifyPassword(password, hash string) error {
	return crypto.VerifyPassword(password, hash)
}

// HashAndValidatePassword validates and hashes a password
func HashAndValidatePassword(password string, policy *PasswordPolicy) (string, error) {
	return crypto.HashAndValidatePassword(password, policy)
}
