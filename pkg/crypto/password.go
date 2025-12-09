package crypto

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the cost factor for bcrypt hashing
	BcryptCost = 12
	// MinPasswordLength is the minimum required password length
	MinPasswordLength = 12
)

var (
	ErrPasswordTooShort    = errors.New("password must be at least 12 characters")
	ErrPasswordNoUppercase = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLowercase = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit     = errors.New("password must contain at least one digit")
	ErrPasswordInvalid     = errors.New("invalid password")
)

// PasswordPolicy defines password requirements
type PasswordPolicy struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireDigit     bool
	RequireSpecial   bool
}

// DefaultPasswordPolicy returns the default password policy
func DefaultPasswordPolicy() *PasswordPolicy {
	return &PasswordPolicy{
		MinLength:        MinPasswordLength,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   false,
	}
}

// ValidatePassword checks if a password meets the policy requirements
func ValidatePassword(password string, policy *PasswordPolicy) error {
	if policy == nil {
		policy = DefaultPasswordPolicy()
	}

	if len(password) < policy.MinLength {
		return ErrPasswordTooShort
	}

	var hasUppercase, hasLowercase, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUppercase = true
		case unicode.IsLower(char):
			hasLowercase = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if policy.RequireUppercase && !hasUppercase {
		return ErrPasswordNoUppercase
	}
	if policy.RequireLowercase && !hasLowercase {
		return ErrPasswordNoLowercase
	}
	if policy.RequireDigit && !hasDigit {
		return ErrPasswordNoDigit
	}
	if policy.RequireSpecial && !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword checks if a password matches a hash
func VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordInvalid
		}
		return err
	}
	return nil
}

// HashAndValidatePassword validates and hashes a password
func HashAndValidatePassword(password string, policy *PasswordPolicy) (string, error) {
	if err := ValidatePassword(password, policy); err != nil {
		return "", err
	}
	return HashPassword(password)
}
