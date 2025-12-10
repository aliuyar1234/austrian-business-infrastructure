package signature

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/atrust"
	"austrian-business-infrastructure/internal/config"
	"austrian-business-infrastructure/internal/idaustria"
)

// EmailSender interface for sending emails
type EmailSender interface {
	SendSignatureRequest(ctx context.Context, to, name, signingURL, documentTitle, message string, expiresAt time.Time) error
	SendSignatureReminder(ctx context.Context, to, name, signingURL, documentTitle string, daysLeft int) error
	SendSignatureComplete(ctx context.Context, to, name, documentTitle string, signerCount int) error
}

// DocumentStore interface for document operations
type DocumentStore interface {
	GetDocumentContent(ctx context.Context, documentID uuid.UUID) ([]byte, error)
	StoreSignedDocument(ctx context.Context, tenantID, originalDocID uuid.UUID, content []byte, title string) (uuid.UUID, error)
}

// Service provides signature business logic
type Service struct {
	repo       *Repository
	config     *config.SignatureConfig
	atrust     atrust.Signer
	idaustria  *idaustria.Client
	email      EmailSender
	documents  DocumentStore
}

// NewService creates a new signature service
func NewService(
	repo *Repository,
	cfg *config.SignatureConfig,
	atrustClient atrust.Signer,
	idaustriaClient *idaustria.Client,
	emailSender EmailSender,
	docStore DocumentStore,
) *Service {
	return &Service{
		repo:      repo,
		config:    cfg,
		atrust:    atrustClient,
		idaustria: idaustriaClient,
		email:     emailSender,
		documents: docStore,
	}
}

// CreateRequestInput contains the input for creating a signature request
type CreateRequestInput struct {
	TenantID     uuid.UUID
	DocumentID   uuid.UUID
	Name         string
	Message      string
	IsSequential bool
	ExpiryDays   int
	Signers      []SignerInput
	Fields       []FieldInput
	CreatedBy    uuid.UUID
}

// SignerInput contains input for a signer
type SignerInput struct {
	Email      string
	Name       string
	OrderIndex int
}

// FieldInput contains input for a signature field
type FieldInput struct {
	SignerIndex int
	Page        int
	X           float64
	Y           float64
	Width       float64
	Height      float64
	ShowName    bool
	ShowDate    bool
	ShowReason  bool
	Reason      string
}

// CreateRequest creates a new signature request
func (s *Service) CreateRequest(ctx context.Context, input *CreateRequestInput) (*SignatureRequest, error) {
	if len(input.Signers) == 0 {
		return nil, fmt.Errorf("at least one signer is required")
	}

	// Default expiry
	expiryDays := input.ExpiryDays
	if expiryDays <= 0 {
		expiryDays = s.config.SignatureLinkExpiryDays
	}

	// Create request
	req := &SignatureRequest{
		TenantID:     input.TenantID,
		DocumentID:   input.DocumentID,
		ExpiresAt:    time.Now().AddDate(0, 0, expiryDays),
		IsSequential: input.IsSequential,
		CreatedBy:    input.CreatedBy,
	}
	if input.Name != "" {
		req.Name = &input.Name
	}
	if input.Message != "" {
		req.Message = &input.Message
	}

	// Create request in database
	if err := s.repo.CreateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Create signers
	signers := make([]*Signer, len(input.Signers))
	for i, signerInput := range input.Signers {
		signer := &Signer{
			SignatureRequestID: req.ID,
			Email:              signerInput.Email,
			Name:               signerInput.Name,
			OrderIndex:         signerInput.OrderIndex,
		}
		if err := s.repo.CreateSigner(ctx, signer); err != nil {
			return nil, fmt.Errorf("failed to create signer: %w", err)
		}
		signers[i] = signer
	}
	req.Signers = signers

	// Create fields
	if len(input.Fields) > 0 {
		fields := make([]*Field, len(input.Fields))
		for i, fieldInput := range input.Fields {
			var signerID *uuid.UUID
			if fieldInput.SignerIndex >= 0 && fieldInput.SignerIndex < len(signers) {
				signerID = &signers[fieldInput.SignerIndex].ID
			}

			field := &Field{
				SignatureRequestID: req.ID,
				SignerID:           signerID,
				Page:               fieldInput.Page,
				X:                  fieldInput.X,
				Y:                  fieldInput.Y,
				Width:              fieldInput.Width,
				Height:             fieldInput.Height,
				ShowName:           fieldInput.ShowName,
				ShowDate:           fieldInput.ShowDate,
				ShowReason:         fieldInput.ShowReason,
			}
			if fieldInput.Reason != "" {
				field.Reason = &fieldInput.Reason
			}

			if err := s.repo.CreateField(ctx, field); err != nil {
				return nil, fmt.Errorf("failed to create field: %w", err)
			}
			fields[i] = field
		}
		req.Fields = fields
	}

	// Create audit event
	s.createAuditEvent(ctx, req.TenantID, &req.ID, nil, nil, nil, AuditEventRequestCreated,
		map[string]interface{}{
			"signer_count": len(signers),
			"field_count":  len(input.Fields),
		}, "user", input.CreatedBy.String(), "", "")

	return req, nil
}

// NotifySigners sends notification emails to signers
func (s *Service) NotifySigners(ctx context.Context, requestID uuid.UUID) error {
	req, err := s.repo.GetRequestWithSigners(ctx, requestID)
	if err != nil {
		return err
	}

	if req.Status != RequestStatusPending {
		return fmt.Errorf("request is not pending")
	}

	// For sequential signing, only notify the first pending signer
	// For parallel signing, notify all pending signers
	for _, signer := range req.Signers {
		if signer.Status != SignerStatusPending {
			continue
		}

		signingURL := fmt.Sprintf("%s/%s", s.config.PortalSigningBasePath, signer.SigningToken)

		message := ""
		if req.Message != nil {
			message = *req.Message
		}

		docTitle := req.DocumentTitle
		if req.Name != nil && *req.Name != "" {
			docTitle = *req.Name
		}

		if s.email != nil {
			if err := s.email.SendSignatureRequest(
				ctx,
				signer.Email,
				signer.Name,
				signingURL,
				docTitle,
				message,
				req.ExpiresAt,
			); err != nil {
				return fmt.Errorf("failed to send notification to %s: %w", signer.Email, err)
			}
		}

		if err := s.repo.MarkSignerNotified(ctx, signer.ID); err != nil {
			return fmt.Errorf("failed to mark signer as notified: %w", err)
		}

		s.createAuditEvent(ctx, req.TenantID, &req.ID, &signer.ID, nil, nil, AuditEventSignerNotified,
			map[string]interface{}{"email": signer.Email}, "system", "", "", "")

		// For sequential signing, only notify one signer
		if req.IsSequential {
			break
		}
	}

	return nil
}

// SendReminder sends a reminder to a signer
func (s *Service) SendReminder(ctx context.Context, signerID uuid.UUID) error {
	signer, err := s.repo.GetSignerByID(ctx, signerID)
	if err != nil {
		return err
	}

	if signer.Status != SignerStatusNotified {
		return fmt.Errorf("signer has not been notified or has already signed")
	}

	req, err := s.repo.GetRequestByID(ctx, signer.SignatureRequestID)
	if err != nil {
		return err
	}

	daysLeft := int(time.Until(req.ExpiresAt).Hours() / 24)
	signingURL := fmt.Sprintf("%s/%s", s.config.PortalSigningBasePath, signer.SigningToken)

	docTitle := req.DocumentTitle
	if req.Name != nil && *req.Name != "" {
		docTitle = *req.Name
	}

	if s.email != nil {
		if err := s.email.SendSignatureReminder(ctx, signer.Email, signer.Name, signingURL, docTitle, daysLeft); err != nil {
			return fmt.Errorf("failed to send reminder: %w", err)
		}
	}

	if err := s.repo.UpdateReminderSent(ctx, signerID); err != nil {
		return err
	}

	s.createAuditEvent(ctx, req.TenantID, &req.ID, &signerID, nil, nil, AuditEventSignerReminded,
		map[string]interface{}{"reminder_count": signer.ReminderCount + 1}, "user", "", "", "")

	return nil
}

// GetSigningInfo retrieves information for the signing page
func (s *Service) GetSigningInfo(ctx context.Context, token string) (*SignatureRequest, *Signer, error) {
	signer, err := s.repo.GetSignerByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.repo.GetRequestWithSigners(ctx, signer.SignatureRequestID)
	if err != nil {
		return nil, nil, err
	}

	if req.Status == RequestStatusExpired || req.Status == RequestStatusCancelled {
		return nil, nil, fmt.Errorf("signature request is no longer available")
	}

	if signer.Status == SignerStatusSigned {
		return nil, nil, ErrAlreadySigned
	}

	// For sequential signing, check if it's this signer's turn
	if req.IsSequential {
		for _, s := range req.Signers {
			if s.OrderIndex < signer.OrderIndex && s.Status != SignerStatusSigned {
				return nil, nil, fmt.Errorf("waiting for previous signer")
			}
		}
	}

	return req, signer, nil
}

// StartSigning initiates the ID Austria authentication flow for signing
func (s *Service) StartSigning(ctx context.Context, token string) (string, error) {
	signer, err := s.repo.GetSignerByToken(ctx, token)
	if err != nil {
		return "", err
	}

	req, err := s.repo.GetRequestByID(ctx, signer.SignatureRequestID)
	if err != nil {
		return "", err
	}

	// Update signer status
	if err := s.repo.UpdateSignerStatus(ctx, signer.ID, SignerStatusSigning); err != nil {
		return "", err
	}

	// Create ID Austria auth request
	authReq, err := s.idaustria.CreateAuthorizationRequest(
		fmt.Sprintf("%s/%s/complete", s.config.SigningCallbackURL, token),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}

	// Store session
	if err := s.repo.SaveIDAustriaSession(ctx, authReq.State, authReq.Nonce,
		authReq.CodeVerifier, authReq.RedirectAfter, &signer.ID, nil); err != nil {
		return "", fmt.Errorf("failed to save session: %w", err)
	}

	// Get auth URL
	authURL, err := s.idaustria.AuthorizationURL(ctx, authReq)
	if err != nil {
		return "", fmt.Errorf("failed to generate auth URL: %w", err)
	}

	s.createAuditEvent(ctx, req.TenantID, &req.ID, &signer.ID, nil, nil, AuditEventSigningStarted,
		map[string]interface{}{}, "signer", signer.Email, "", "")

	return authURL, nil
}

// CompleteSigningInput contains the input for completing a signature
type CompleteSigningInput struct {
	Token       string
	State       string
	Code        string
	Error       string
	ErrorDesc   string
	IP          string
	UserAgent   string
}

// CompleteSigning completes the signing process after ID Austria callback
func (s *Service) CompleteSigning(ctx context.Context, input *CompleteSigningInput) (*SignatureRequest, error) {
	// Get session
	_, codeVerifier, signerID, _, redirectAfter, err := s.repo.GetIDAustriaSessionByState(ctx, input.State)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	// Validate callback
	if input.Error != "" {
		return nil, fmt.Errorf("ID Austria error: %s - %s", input.Error, input.ErrorDesc)
	}

	if signerID == nil {
		return nil, fmt.Errorf("invalid session context")
	}

	// Exchange code for tokens
	token, err := s.idaustria.ExchangeCode(ctx, input.Code, codeVerifier)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info
	userInfo, err := s.idaustria.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Get signer and request
	signer, err := s.repo.GetSignerByID(ctx, *signerID)
	if err != nil {
		return nil, err
	}

	req, err := s.repo.GetRequestByID(ctx, signer.SignatureRequestID)
	if err != nil {
		return nil, err
	}

	// Get document content
	docContent, err := s.documents.GetDocumentContent(ctx, req.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Calculate document hash
	docHash := atrust.HashDocument(docContent)

	// Sign with A-Trust
	signReq := &atrust.SignRequest{
		DocumentHash:  docHash,
		HashAlgorithm: atrust.HashAlgoSHA256,
		SignerCertID:  userInfo.Subject, // Use subject as cert ID
	}

	signResp, err := s.atrust.Sign(ctx, signReq)
	if err != nil {
		s.createAuditEvent(ctx, req.TenantID, &req.ID, signerID, nil, nil, AuditEventSigningFailed,
			map[string]interface{}{"error": err.Error()}, "signer", signer.Email, input.IP, input.UserAgent)
		return nil, fmt.Errorf("failed to sign document: %w", err)
	}

	// Update session as authenticated
	bpkHash := ""
	if userInfo.BPK != "" {
		bpkHash = idaustria.HashBPK(userInfo.BPK)
	}
	if err := s.repo.UpdateIDAustriaSessionAuthenticated(ctx, input.State, userInfo.Subject, userInfo.Name, bpkHash); err != nil {
		// Non-fatal error, continue
	}

	// Parse certificate info
	certInfo, _ := s.atrust.GetCertificateInfo(ctx, userInfo.Subject)
	certSubject, certSerial, certIssuer := "", "", ""
	if certInfo != nil {
		certSubject = certInfo.Subject
		certSerial = certInfo.SerialNumber
		certIssuer = certInfo.Issuer
	}

	// Mark signer as signed
	if err := s.repo.MarkSignerSigned(ctx, *signerID,
		certSubject, certSerial, certIssuer, signResp.Signature,
		userInfo.Subject, bpkHash); err != nil {
		return nil, fmt.Errorf("failed to update signer: %w", err)
	}

	// Audit event
	s.createAuditEvent(ctx, req.TenantID, &req.ID, signerID, nil, nil, AuditEventSigningCompleted,
		map[string]interface{}{
			"certificate_subject": certSubject,
			"signed_at":           signResp.SignedAt,
		}, "signer", signer.Email, input.IP, input.UserAgent)

	// Check if all signers have signed
	signers, err := s.repo.ListSignersByRequest(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	allSigned := true
	nextSigner := (*Signer)(nil)
	for _, s := range signers {
		if s.Status != SignerStatusSigned {
			allSigned = false
			if req.IsSequential && nextSigner == nil {
				nextSigner = s
			}
		}
	}

	if allSigned {
		// TODO: Embed all signatures into the PDF
		// For now, we'll store the original document as "signed"
		signedDocID, err := s.documents.StoreSignedDocument(ctx, req.TenantID, req.DocumentID, docContent, req.DocumentTitle+" (signiert)")
		if err != nil {
			return nil, fmt.Errorf("failed to store signed document: %w", err)
		}

		if err := s.repo.CompleteRequest(ctx, req.ID, signedDocID); err != nil {
			return nil, fmt.Errorf("failed to complete request: %w", err)
		}

		// Record usage
		usage := &Usage{
			TenantID:           req.TenantID,
			SignatureRequestID: &req.ID,
			SignatureCount:     len(signers),
			UsageDate:          time.Now(),
		}
		if s.config.SignatureCostCents > 0 {
			cost := s.config.SignatureCostCents * len(signers)
			usage.CostCents = &cost
		}
		s.repo.RecordUsage(ctx, usage)

		// Send completion notification
		if s.email != nil {
			// Notify the requester
			// TODO: Get requester email
		}

		// Update request
		req.Status = RequestStatusCompleted
		req.SignedDocumentID = &signedDocID

		s.createAuditEvent(ctx, req.TenantID, &req.ID, nil, nil, nil, AuditEventRequestCompleted,
			map[string]interface{}{"signed_document_id": signedDocID}, "system", "", "", "")
	} else if req.IsSequential && nextSigner != nil {
		// Notify next signer
		s.NotifySigners(ctx, req.ID)
	}

	// Use redirectAfter from session
	_ = redirectAfter

	return req, nil
}

// CancelRequest cancels a signature request
func (s *Service) CancelRequest(ctx context.Context, requestID uuid.UUID, userID uuid.UUID) error {
	req, err := s.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return err
	}

	if req.Status == RequestStatusCompleted {
		return fmt.Errorf("cannot cancel completed request")
	}

	if err := s.repo.CancelRequest(ctx, requestID); err != nil {
		return err
	}

	s.createAuditEvent(ctx, req.TenantID, &requestID, nil, nil, nil, AuditEventRequestCancelled,
		map[string]interface{}{}, "user", userID.String(), "", "")

	return nil
}

// GetRequest retrieves a signature request by ID
func (s *Service) GetRequest(ctx context.Context, requestID uuid.UUID) (*SignatureRequest, error) {
	return s.repo.GetRequestWithSigners(ctx, requestID)
}

// ListRequests lists signature requests for a tenant
func (s *Service) ListRequests(ctx context.Context, tenantID uuid.UUID, status *RequestStatus, limit, offset int) ([]*SignatureRequest, int, error) {
	return s.repo.ListRequestsByTenant(ctx, tenantID, status, limit, offset)
}

// ExpireRequests expires overdue signature requests
func (s *Service) ExpireRequests(ctx context.Context) (int64, error) {
	count, err := s.repo.ExpirePendingRequests(ctx)
	if err != nil {
		return 0, err
	}

	// TODO: Send expiration notifications
	return count, nil
}

// CreateFromTemplate creates a signature request from a template
func (s *Service) CreateFromTemplate(ctx context.Context, templateID uuid.UUID, documentID uuid.UUID, signers []SignerInput, tenantID, userID uuid.UUID) (*SignatureRequest, error) {
	tpl, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Parse signer templates
	var signerTpls []SignerTemplate
	if err := json.Unmarshal(tpl.SignerTemplates, &signerTpls); err != nil {
		return nil, fmt.Errorf("failed to parse signer templates: %w", err)
	}

	// Merge template with provided signers
	// TODO: Implement template merging logic

	input := &CreateRequestInput{
		TenantID:     tenantID,
		DocumentID:   documentID,
		IsSequential: tpl.IsSequential,
		ExpiryDays:   tpl.DefaultExpiryDays,
		Signers:      signers,
		CreatedBy:    userID,
	}
	if tpl.DefaultMessage != nil {
		input.Message = *tpl.DefaultMessage
	}

	req, err := s.CreateRequest(ctx, input)
	if err != nil {
		return nil, err
	}

	// Increment template usage
	s.repo.IncrementTemplateUsage(ctx, templateID)

	return req, nil
}

// Helper to create audit events
func (s *Service) createAuditEvent(ctx context.Context, tenantID uuid.UUID, reqID, signerID, batchID, verifyID *uuid.UUID, eventType string, details map[string]interface{}, actorType, actorID, actorIP, actorUA string) {
	detailsJSON, _ := json.Marshal(details)

	event := &AuditEvent{
		TenantID:           &tenantID,
		SignatureRequestID: reqID,
		SignerID:           signerID,
		BatchID:            batchID,
		VerificationID:     verifyID,
		EventType:          eventType,
		Details:            detailsJSON,
	}
	if actorType != "" {
		event.ActorType = &actorType
	}
	if actorID != "" {
		event.ActorID = &actorID
	}
	if actorIP != "" {
		event.ActorIP = &actorIP
	}
	if actorUA != "" {
		event.ActorUserAgent = &actorUA
	}

	s.repo.CreateAuditEvent(ctx, event)
}
