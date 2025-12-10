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
	// Signature-related emails
	SendSignatureRequest(ctx context.Context, to string, params SignatureRequestParams) error
	SendSignatureReminder(ctx context.Context, to string, params SignatureReminderParams) error
	SendSignatureCompleted(ctx context.Context, to string, params SignatureCompletedParams) error
	SendSignatureExpired(ctx context.Context, to string, params SignatureExpiredParams) error
}

// SignatureRequestParams contains parameters for signature request emails
type SignatureRequestParams struct {
	SignerName      string
	RequesterName   string
	CompanyName     string
	DocumentTitle   string
	SigningURL      string
	ExpiresAt       string
	Message         string
	SignerPosition  int
	TotalSigners    int
}

// SignatureReminderParams contains parameters for signature reminder emails
type SignatureReminderParams struct {
	SignerName    string
	DocumentTitle string
	SigningURL    string
	ExpiresAt     string
	DaysLeft      int
	ReminderCount int
}

// SignatureCompletedParams contains parameters for signature completion emails
type SignatureCompletedParams struct {
	RequesterName string
	DocumentTitle string
	SignerName    string
	SignedAt      string
	AllSigned     bool
	DownloadURL   string
}

// SignatureExpiredParams contains parameters for signature expiry emails
type SignatureExpiredParams struct {
	RecipientName string
	DocumentTitle string
	ExpiredAt     string
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

// SendSignatureRequest sends a signature request email
func (s *SMTPService) SendSignatureRequest(ctx context.Context, to string, params SignatureRequestParams) error {
	subject := fmt.Sprintf("Signaturanfrage: %s", params.DocumentTitle)

	positionInfo := ""
	if params.TotalSigners > 1 {
		positionInfo = fmt.Sprintf("\n\nSie sind Unterzeichner %d von %d.", params.SignerPosition, params.TotalSigners)
	}

	messageSection := ""
	if params.Message != "" {
		messageSection = fmt.Sprintf("\n\nNachricht von %s:\n%s", params.RequesterName, params.Message)
	}

	body := fmt.Sprintf(`Guten Tag %s,

%s von %s bittet Sie, das folgende Dokument digital zu signieren:

Dokument: %s%s%s

Bitte klicken Sie auf den folgenden Link, um das Dokument zu signieren:

%s

Die Signatur erfolgt mit ID Austria (qualifizierte elektronische Signatur).

Dieser Link ist gueltig bis: %s

Bei Fragen wenden Sie sich bitte an %s.

Mit freundlichen Gruessen,
Austrian Business Platform
`, params.SignerName, params.RequesterName, params.CompanyName, params.DocumentTitle, positionInfo, messageSection, params.SigningURL, params.ExpiresAt, params.RequesterName)

	return s.send(to, subject, body)
}

// SendSignatureReminder sends a signature reminder email
func (s *SMTPService) SendSignatureReminder(ctx context.Context, to string, params SignatureReminderParams) error {
	subject := fmt.Sprintf("Erinnerung: Signatur ausstehend - %s", params.DocumentTitle)

	urgencyNote := ""
	if params.DaysLeft <= 3 {
		urgencyNote = "\n\n*** DRINGEND: Nur noch wenige Tage Zeit! ***\n"
	}

	body := fmt.Sprintf(`Guten Tag %s,

dies ist eine Erinnerung, dass Ihre Signatur fuer das folgende Dokument noch aussteht:

Dokument: %s%s

Bitte signieren Sie das Dokument bis zum %s (%d Tage verbleibend):

%s

Mit freundlichen Gruessen,
Austrian Business Platform
`, params.SignerName, params.DocumentTitle, urgencyNote, params.ExpiresAt, params.DaysLeft, params.SigningURL)

	return s.send(to, subject, body)
}

// SendSignatureCompleted sends a signature completion notification
func (s *SMTPService) SendSignatureCompleted(ctx context.Context, to string, params SignatureCompletedParams) error {
	var subject, body string

	if params.AllSigned {
		subject = fmt.Sprintf("Signatur abgeschlossen: %s", params.DocumentTitle)
		body = fmt.Sprintf(`Guten Tag %s,

alle Signaturen fuer das folgende Dokument wurden abgeschlossen:

Dokument: %s
Letzte Signatur von: %s
Zeitpunkt: %s

Sie koennen das signierte Dokument hier herunterladen:

%s

Mit freundlichen Gruessen,
Austrian Business Platform
`, params.RequesterName, params.DocumentTitle, params.SignerName, params.SignedAt, params.DownloadURL)
	} else {
		subject = fmt.Sprintf("Signatur erhalten: %s", params.DocumentTitle)
		body = fmt.Sprintf(`Guten Tag %s,

eine Signatur wurde fuer das folgende Dokument hinzugefuegt:

Dokument: %s
Signiert von: %s
Zeitpunkt: %s

Das Dokument wird nun an den naechsten Unterzeichner weitergeleitet.

Mit freundlichen Gruessen,
Austrian Business Platform
`, params.RequesterName, params.DocumentTitle, params.SignerName, params.SignedAt)
	}

	return s.send(to, subject, body)
}

// SendSignatureExpired sends a signature expiry notification
func (s *SMTPService) SendSignatureExpired(ctx context.Context, to string, params SignatureExpiredParams) error {
	subject := fmt.Sprintf("Signaturanfrage abgelaufen: %s", params.DocumentTitle)

	body := fmt.Sprintf(`Guten Tag %s,

die folgende Signaturanfrage ist abgelaufen:

Dokument: %s
Abgelaufen am: %s

Die ausstehenden Signaturen koennen nicht mehr abgeschlossen werden.
Bitte erstellen Sie bei Bedarf eine neue Signaturanfrage.

Mit freundlichen Gruessen,
Austrian Business Platform
`, params.RecipientName, params.DocumentTitle, params.ExpiredAt)

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

// SendSignatureRequest does nothing (no-op)
func (s *NoopService) SendSignatureRequest(ctx context.Context, to string, params SignatureRequestParams) error {
	return nil
}

// SendSignatureReminder does nothing (no-op)
func (s *NoopService) SendSignatureReminder(ctx context.Context, to string, params SignatureReminderParams) error {
	return nil
}

// SendSignatureCompleted does nothing (no-op)
func (s *NoopService) SendSignatureCompleted(ctx context.Context, to string, params SignatureCompletedParams) error {
	return nil
}

// SendSignatureExpired does nothing (no-op)
func (s *NoopService) SendSignatureExpired(ctx context.Context, to string, params SignatureExpiredParams) error {
	return nil
}
