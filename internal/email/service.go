package email

import (
	"context"
	"fmt"
	"net/smtp"
)

// Service provides email sending functionality
type Service interface {
	SendInvitation(ctx context.Context, to, inviterName, tenantName, token, appURL string) error
	SendPasswordReset(ctx context.Context, to, token, appURL string) error
	SendEmailVerification(ctx context.Context, to, token, appURL string) error
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// SMTPService implements email sending via SMTP
type SMTPService struct {
	config *SMTPConfig
}

// NewSMTPService creates a new SMTP email service
func NewSMTPService(config *SMTPConfig) *SMTPService {
	return &SMTPService{config: config}
}

// SendInvitation sends an invitation email
func (s *SMTPService) SendInvitation(ctx context.Context, to, inviterName, tenantName, token, appURL string) error {
	subject := fmt.Sprintf("You've been invited to join %s", tenantName)
	inviteURL := fmt.Sprintf("%s/invitations/accept?token=%s", appURL, token)

	body := fmt.Sprintf(`Hello,

%s has invited you to join %s on Austrian Business Platform.

Click the link below to accept the invitation and create your account:

%s

This invitation will expire in 7 days.

If you did not expect this invitation, you can safely ignore this email.

Best regards,
Austrian Business Platform Team
`, inviterName, tenantName, inviteURL)

	return s.send(to, subject, body)
}

// SendPasswordReset sends a password reset email
func (s *SMTPService) SendPasswordReset(ctx context.Context, to, token, appURL string) error {
	subject := "Reset your password"
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", appURL, token)

	body := fmt.Sprintf(`Hello,

A password reset was requested for your account.

Click the link below to reset your password:

%s

This link will expire in 1 hour.

If you did not request a password reset, you can safely ignore this email.

Best regards,
Austrian Business Platform Team
`, resetURL)

	return s.send(to, subject, body)
}

// SendEmailVerification sends an email verification email
func (s *SMTPService) SendEmailVerification(ctx context.Context, to, token, appURL string) error {
	subject := "Verify your email address"
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", appURL, token)

	body := fmt.Sprintf(`Hello,

Please verify your email address by clicking the link below:

%s

If you did not create an account, you can safely ignore this email.

Best regards,
Austrian Business Platform Team
`, verifyURL)

	return s.send(to, subject, body)
}

func (s *SMTPService) send(to, subject, body string) error {
	if s.config.Host == "" {
		// SMTP not configured - log and skip
		return nil
	}

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=utf-8\r\n"+
		"\r\n"+
		"%s", s.config.From, to, subject, body)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	var auth smtp.Auth
	if s.config.User != "" && s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.User, s.config.Password, s.config.Host)
	}

	return smtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(msg))
}

// NoopService is a no-op email service for testing/development
type NoopService struct{}

// NewNoopService creates a no-op email service
func NewNoopService() *NoopService {
	return &NoopService{}
}

// SendInvitation does nothing (no-op)
func (s *NoopService) SendInvitation(ctx context.Context, to, inviterName, tenantName, token, appURL string) error {
	return nil
}

// SendPasswordReset does nothing (no-op)
func (s *NoopService) SendPasswordReset(ctx context.Context, to, token, appURL string) error {
	return nil
}

// SendEmailVerification does nothing (no-op)
func (s *NoopService) SendEmailVerification(ctx context.Context, to, token, appURL string) error {
	return nil
}
