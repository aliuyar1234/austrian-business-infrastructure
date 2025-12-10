package signature

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRequestNotFound      = errors.New("signature request not found")
	ErrSignerNotFound       = errors.New("signer not found")
	ErrBatchNotFound        = errors.New("batch not found")
	ErrBatchItemNotFound    = errors.New("batch item not found")
	ErrTemplateNotFound     = errors.New("template not found")
	ErrVerificationNotFound = errors.New("verification not found")
	ErrInvalidToken         = errors.New("invalid or expired signing token")
	ErrAlreadySigned        = errors.New("document already signed by this signer")
)

// Repository provides signature data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new signature repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ===== Signature Request Operations =====

// CreateRequest creates a new signature request
func (r *Repository) CreateRequest(ctx context.Context, req *SignatureRequest) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	query := `
		INSERT INTO signature_requests (
			id, tenant_id, document_id, name, message, expires_at, is_sequential, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING status, current_signer_index, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		req.ID, req.TenantID, req.DocumentID, req.Name, req.Message,
		req.ExpiresAt, req.IsSequential, req.CreatedBy,
	).Scan(&req.Status, &req.CurrentSignerIdx, &req.CreatedAt, &req.UpdatedAt)

	return err
}

// GetRequestByID retrieves a signature request by ID
func (r *Repository) GetRequestByID(ctx context.Context, id uuid.UUID) (*SignatureRequest, error) {
	query := `
		SELECT sr.id, sr.tenant_id, sr.document_id, sr.name, sr.message, sr.expires_at,
			sr.status, sr.completed_at, sr.is_sequential, sr.current_signer_index,
			sr.signed_document_id, sr.created_by, sr.created_at, sr.updated_at,
			d.title as document_title
		FROM signature_requests sr
		LEFT JOIN documents d ON sr.document_id = d.id
		WHERE sr.id = $1
	`

	req := &SignatureRequest{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&req.ID, &req.TenantID, &req.DocumentID, &req.Name, &req.Message, &req.ExpiresAt,
		&req.Status, &req.CompletedAt, &req.IsSequential, &req.CurrentSignerIdx,
		&req.SignedDocumentID, &req.CreatedBy, &req.CreatedAt, &req.UpdatedAt,
		&req.DocumentTitle,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}

	return req, nil
}

// GetRequestWithSigners retrieves a request with all its signers
func (r *Repository) GetRequestWithSigners(ctx context.Context, id uuid.UUID) (*SignatureRequest, error) {
	req, err := r.GetRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}

	signers, err := r.ListSignersByRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	req.Signers = signers

	fields, err := r.ListFieldsByRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	req.Fields = fields

	return req, nil
}

// ListRequestsByTenant lists signature requests for a tenant
func (r *Repository) ListRequestsByTenant(ctx context.Context, tenantID uuid.UUID, status *RequestStatus, limit, offset int) ([]*SignatureRequest, int, error) {
	countQuery := `SELECT COUNT(*) FROM signature_requests WHERE tenant_id = $1`
	listQuery := `
		SELECT sr.id, sr.tenant_id, sr.document_id, sr.name, sr.message, sr.expires_at,
			sr.status, sr.completed_at, sr.is_sequential, sr.current_signer_index,
			sr.signed_document_id, sr.created_by, sr.created_at, sr.updated_at,
			d.title as document_title
		FROM signature_requests sr
		LEFT JOIN documents d ON sr.document_id = d.id
		WHERE sr.tenant_id = $1
	`

	args := []interface{}{tenantID}
	argNum := 2

	if status != nil {
		countQuery += ` AND status = $2`
		listQuery += ` AND sr.status = $2`
		args = append(args, *status)
		argNum++
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += ` ORDER BY sr.created_at DESC`
	if limit > 0 {
		listQuery += ` LIMIT $` + itoa(argNum)
		args = append(args, limit)
		argNum++
	}
	if offset > 0 {
		listQuery += ` OFFSET $` + itoa(argNum)
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var requests []*SignatureRequest
	for rows.Next() {
		req := &SignatureRequest{}
		err := rows.Scan(
			&req.ID, &req.TenantID, &req.DocumentID, &req.Name, &req.Message, &req.ExpiresAt,
			&req.Status, &req.CompletedAt, &req.IsSequential, &req.CurrentSignerIdx,
			&req.SignedDocumentID, &req.CreatedBy, &req.CreatedAt, &req.UpdatedAt,
			&req.DocumentTitle,
		)
		if err != nil {
			return nil, 0, err
		}
		requests = append(requests, req)
	}

	return requests, total, rows.Err()
}

// UpdateRequestStatus updates the status of a signature request
func (r *Repository) UpdateRequestStatus(ctx context.Context, id uuid.UUID, status RequestStatus) error {
	query := `UPDATE signature_requests SET status = $2, updated_at = NOW() WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrRequestNotFound
	}
	return nil
}

// CompleteRequest marks a request as completed
func (r *Repository) CompleteRequest(ctx context.Context, id uuid.UUID, signedDocID uuid.UUID) error {
	query := `
		UPDATE signature_requests
		SET status = 'completed', completed_at = NOW(), signed_document_id = $2, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, id, signedDocID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrRequestNotFound
	}
	return nil
}

// CancelRequest cancels a signature request
func (r *Repository) CancelRequest(ctx context.Context, id uuid.UUID) error {
	return r.UpdateRequestStatus(ctx, id, RequestStatusCancelled)
}

// ExpirePendingRequests marks overdue requests as expired
func (r *Repository) ExpirePendingRequests(ctx context.Context) (int64, error) {
	query := `
		UPDATE signature_requests
		SET status = 'expired', updated_at = NOW()
		WHERE status = 'pending' AND expires_at < NOW()
	`
	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ===== Signer Operations =====

// CreateSigner creates a new signer for a request
func (r *Repository) CreateSigner(ctx context.Context, signer *Signer) error {
	if signer.ID == uuid.Nil {
		signer.ID = uuid.New()
	}

	// Generate signing token
	token, err := generateSecureToken(32)
	if err != nil {
		return err
	}
	signer.SigningToken = token
	signer.TokenExpiresAt = time.Now().Add(14 * 24 * time.Hour) // 14 days

	query := `
		INSERT INTO signers (
			id, signature_request_id, email, name, order_index,
			signing_token, token_expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING status, created_at
	`

	err = r.pool.QueryRow(ctx, query,
		signer.ID, signer.SignatureRequestID, signer.Email, signer.Name,
		signer.OrderIndex, signer.SigningToken, signer.TokenExpiresAt,
	).Scan(&signer.Status, &signer.CreatedAt)

	return err
}

// GetSignerByID retrieves a signer by ID
func (r *Repository) GetSignerByID(ctx context.Context, id uuid.UUID) (*Signer, error) {
	query := `
		SELECT id, signature_request_id, email, name, order_index,
			signing_token, token_expires_at, token_used, status, notified_at,
			signed_at, certificate_subject, certificate_serial, certificate_issuer,
			signature_value, idaustria_subject, idaustria_bpk, reminder_count,
			last_reminder_at, created_at
		FROM signers WHERE id = $1
	`

	signer := &Signer{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&signer.ID, &signer.SignatureRequestID, &signer.Email, &signer.Name, &signer.OrderIndex,
		&signer.SigningToken, &signer.TokenExpiresAt, &signer.TokenUsed, &signer.Status,
		&signer.NotifiedAt, &signer.SignedAt, &signer.CertificateSubject, &signer.CertificateSerial,
		&signer.CertificateIssuer, &signer.SignatureValue, &signer.IDAustriaSubject,
		&signer.IDAustriaBPK, &signer.ReminderCount, &signer.LastReminderAt, &signer.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSignerNotFound
		}
		return nil, err
	}

	return signer, nil
}

// GetSignerByToken retrieves a signer by their signing token
func (r *Repository) GetSignerByToken(ctx context.Context, token string) (*Signer, error) {
	query := `
		SELECT id, signature_request_id, email, name, order_index,
			signing_token, token_expires_at, token_used, status, notified_at,
			signed_at, certificate_subject, certificate_serial, certificate_issuer,
			signature_value, idaustria_subject, idaustria_bpk, reminder_count,
			last_reminder_at, created_at
		FROM signers
		WHERE signing_token = $1 AND NOT token_used AND token_expires_at > NOW()
	`

	signer := &Signer{}
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&signer.ID, &signer.SignatureRequestID, &signer.Email, &signer.Name, &signer.OrderIndex,
		&signer.SigningToken, &signer.TokenExpiresAt, &signer.TokenUsed, &signer.Status,
		&signer.NotifiedAt, &signer.SignedAt, &signer.CertificateSubject, &signer.CertificateSerial,
		&signer.CertificateIssuer, &signer.SignatureValue, &signer.IDAustriaSubject,
		&signer.IDAustriaBPK, &signer.ReminderCount, &signer.LastReminderAt, &signer.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	return signer, nil
}

// ListSignersByRequest lists all signers for a request
func (r *Repository) ListSignersByRequest(ctx context.Context, requestID uuid.UUID) ([]*Signer, error) {
	query := `
		SELECT id, signature_request_id, email, name, order_index,
			status, notified_at, signed_at, certificate_subject, certificate_serial,
			certificate_issuer, reminder_count, last_reminder_at, created_at
		FROM signers
		WHERE signature_request_id = $1
		ORDER BY order_index ASC
	`

	rows, err := r.pool.Query(ctx, query, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signers []*Signer
	for rows.Next() {
		signer := &Signer{}
		err := rows.Scan(
			&signer.ID, &signer.SignatureRequestID, &signer.Email, &signer.Name,
			&signer.OrderIndex, &signer.Status, &signer.NotifiedAt, &signer.SignedAt,
			&signer.CertificateSubject, &signer.CertificateSerial, &signer.CertificateIssuer,
			&signer.ReminderCount, &signer.LastReminderAt, &signer.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}

	return signers, rows.Err()
}

// UpdateSignerStatus updates a signer's status
func (r *Repository) UpdateSignerStatus(ctx context.Context, id uuid.UUID, status SignerStatus) error {
	query := `UPDATE signers SET status = $2 WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrSignerNotFound
	}
	return nil
}

// MarkSignerNotified marks a signer as notified
func (r *Repository) MarkSignerNotified(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE signers SET status = 'notified', notified_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// MarkSignerSigned marks a signer as having signed
func (r *Repository) MarkSignerSigned(ctx context.Context, id uuid.UUID, certSubject, certSerial, certIssuer, signatureValue, idSubject, bpkHash string) error {
	query := `
		UPDATE signers
		SET status = 'signed', signed_at = NOW(), token_used = TRUE,
			certificate_subject = $2, certificate_serial = $3, certificate_issuer = $4,
			signature_value = $5, idaustria_subject = $6, idaustria_bpk = $7
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, id, certSubject, certSerial, certIssuer, signatureValue, idSubject, bpkHash)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrSignerNotFound
	}
	return nil
}

// UpdateReminderSent updates reminder tracking for a signer
func (r *Repository) UpdateReminderSent(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE signers
		SET reminder_count = reminder_count + 1, last_reminder_at = NOW()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// ===== Signature Field Operations =====

// CreateField creates a new signature field
func (r *Repository) CreateField(ctx context.Context, field *Field) error {
	if field.ID == uuid.Nil {
		field.ID = uuid.New()
	}

	query := `
		INSERT INTO signature_fields (
			id, signature_request_id, signer_id, page, x, y, width, height,
			show_name, show_date, show_reason, reason, background_image_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at
	`

	return r.pool.QueryRow(ctx, query,
		field.ID, field.SignatureRequestID, field.SignerID, field.Page,
		field.X, field.Y, field.Width, field.Height, field.ShowName,
		field.ShowDate, field.ShowReason, field.Reason, field.BackgroundImageURL,
	).Scan(&field.CreatedAt)
}

// GetFieldByID retrieves a field by ID
func (r *Repository) GetFieldByID(ctx context.Context, id uuid.UUID) (*Field, error) {
	query := `
		SELECT id, signature_request_id, signer_id, page, x, y, width, height,
			show_name, show_date, show_reason, reason, background_image_url, created_at
		FROM signature_fields WHERE id = $1
	`

	field := &Field{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&field.ID, &field.SignatureRequestID, &field.SignerID, &field.Page,
		&field.X, &field.Y, &field.Width, &field.Height, &field.ShowName,
		&field.ShowDate, &field.ShowReason, &field.Reason, &field.BackgroundImageURL,
		&field.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("field not found")
		}
		return nil, err
	}

	return field, nil
}

// ListFieldsByRequest lists all fields for a request
func (r *Repository) ListFieldsByRequest(ctx context.Context, requestID uuid.UUID) ([]*Field, error) {
	query := `
		SELECT id, signature_request_id, signer_id, page, x, y, width, height,
			show_name, show_date, show_reason, reason, background_image_url, created_at
		FROM signature_fields
		WHERE signature_request_id = $1
		ORDER BY page ASC, y DESC
	`

	rows, err := r.pool.Query(ctx, query, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []*Field
	for rows.Next() {
		field := &Field{}
		err := rows.Scan(
			&field.ID, &field.SignatureRequestID, &field.SignerID, &field.Page,
			&field.X, &field.Y, &field.Width, &field.Height, &field.ShowName,
			&field.ShowDate, &field.ShowReason, &field.Reason, &field.BackgroundImageURL,
			&field.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	return fields, rows.Err()
}

// UpdateField updates a signature field
func (r *Repository) UpdateField(ctx context.Context, field *Field) error {
	query := `
		UPDATE signature_fields
		SET page = $2, x = $3, y = $4, width = $5, height = $6,
			show_name = $7, show_date = $8, show_reason = $9, reason = $10,
			background_image_url = $11
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		field.ID, field.Page, field.X, field.Y, field.Width, field.Height,
		field.ShowName, field.ShowDate, field.ShowReason, field.Reason,
		field.BackgroundImageURL,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("field not found")
	}
	return nil
}

// DeleteField deletes a signature field
func (r *Repository) DeleteField(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM signature_fields WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("field not found")
	}
	return nil
}

// ===== Template Operations =====

// CreateTemplate creates a new signature template
func (r *Repository) CreateTemplate(ctx context.Context, tpl *Template) error {
	if tpl.ID == uuid.Nil {
		tpl.ID = uuid.New()
	}

	query := `
		INSERT INTO signature_templates (
			id, tenant_id, name, description, is_sequential, default_expiry_days,
			default_message, signer_templates, field_templates, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`

	return r.pool.QueryRow(ctx, query,
		tpl.ID, tpl.TenantID, tpl.Name, tpl.Description, tpl.IsSequential,
		tpl.DefaultExpiryDays, tpl.DefaultMessage, tpl.SignerTemplates,
		tpl.FieldTemplates, tpl.CreatedBy,
	).Scan(&tpl.CreatedAt, &tpl.UpdatedAt)
}

// GetTemplateByID retrieves a template by ID
func (r *Repository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*Template, error) {
	query := `
		SELECT id, tenant_id, name, description, is_sequential, default_expiry_days,
			default_message, signer_templates, field_templates, use_count,
			last_used_at, created_by, created_at, updated_at
		FROM signature_templates WHERE id = $1
	`

	tpl := &Template{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tpl.ID, &tpl.TenantID, &tpl.Name, &tpl.Description, &tpl.IsSequential,
		&tpl.DefaultExpiryDays, &tpl.DefaultMessage, &tpl.SignerTemplates,
		&tpl.FieldTemplates, &tpl.UseCount, &tpl.LastUsedAt, &tpl.CreatedBy,
		&tpl.CreatedAt, &tpl.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	return tpl, nil
}

// ListTemplatesByTenant lists templates for a tenant
func (r *Repository) ListTemplatesByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Template, int, error) {
	countQuery := `SELECT COUNT(*) FROM signature_templates WHERE tenant_id = $1`

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
		SELECT id, tenant_id, name, description, is_sequential, default_expiry_days,
			default_message, signer_templates, field_templates, use_count,
			last_used_at, created_by, created_at, updated_at
		FROM signature_templates
		WHERE tenant_id = $1
		ORDER BY name ASC
	`

	args := []interface{}{tenantID}
	if limit > 0 {
		listQuery += ` LIMIT $2`
		args = append(args, limit)
	}
	if offset > 0 {
		listQuery += ` OFFSET $3`
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var templates []*Template
	for rows.Next() {
		tpl := &Template{}
		err := rows.Scan(
			&tpl.ID, &tpl.TenantID, &tpl.Name, &tpl.Description, &tpl.IsSequential,
			&tpl.DefaultExpiryDays, &tpl.DefaultMessage, &tpl.SignerTemplates,
			&tpl.FieldTemplates, &tpl.UseCount, &tpl.LastUsedAt, &tpl.CreatedBy,
			&tpl.CreatedAt, &tpl.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		templates = append(templates, tpl)
	}

	return templates, total, rows.Err()
}

// IncrementTemplateUsage increments the use count for a template
func (r *Repository) IncrementTemplateUsage(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE signature_templates
		SET use_count = use_count + 1, last_used_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// DeleteTemplate deletes a template
func (r *Repository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM signature_templates WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

// ===== Verification Operations =====

// CreateVerification stores a verification result
func (r *Repository) CreateVerification(ctx context.Context, v *Verification) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}

	query := `
		INSERT INTO signature_verifications (
			id, tenant_id, document_id, document_hash, original_filename,
			is_valid, verification_status, signatures, signature_count, verified_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING verified_at, created_at
	`

	return r.pool.QueryRow(ctx, query,
		v.ID, v.TenantID, v.DocumentID, v.DocumentHash, v.OriginalFilename,
		v.IsValid, v.VerificationStatus, v.Signatures, v.SignatureCount, v.VerifiedBy,
	).Scan(&v.VerifiedAt, &v.CreatedAt)
}

// GetVerificationByID retrieves a verification by ID
func (r *Repository) GetVerificationByID(ctx context.Context, id uuid.UUID) (*Verification, error) {
	query := `
		SELECT id, tenant_id, document_id, document_hash, original_filename,
			is_valid, verification_status, signatures, signature_count,
			verified_at, verified_by, created_at
		FROM signature_verifications WHERE id = $1
	`

	v := &Verification{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.TenantID, &v.DocumentID, &v.DocumentHash, &v.OriginalFilename,
		&v.IsValid, &v.VerificationStatus, &v.Signatures, &v.SignatureCount,
		&v.VerifiedAt, &v.VerifiedBy, &v.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVerificationNotFound
		}
		return nil, err
	}

	return v, nil
}

// ===== Audit Operations =====

// CreateAuditEvent creates an audit log entry
func (r *Repository) CreateAuditEvent(ctx context.Context, event *AuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	query := `
		INSERT INTO signature_audit_logs (
			id, tenant_id, signature_request_id, signer_id, batch_id,
			verification_id, event_type, details, actor_type, actor_id,
			actor_ip, actor_user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at
	`

	return r.pool.QueryRow(ctx, query,
		event.ID, event.TenantID, event.SignatureRequestID, event.SignerID,
		event.BatchID, event.VerificationID, event.EventType, event.Details,
		event.ActorType, event.ActorID, event.ActorIP, event.ActorUserAgent,
	).Scan(&event.CreatedAt)
}

// ListAuditEventsByRequest lists audit events for a request
func (r *Repository) ListAuditEventsByRequest(ctx context.Context, requestID uuid.UUID) ([]*AuditEvent, error) {
	query := `
		SELECT id, tenant_id, signature_request_id, signer_id, batch_id,
			verification_id, event_type, details, actor_type, actor_id,
			actor_ip, actor_user_agent, created_at
		FROM signature_audit_logs
		WHERE signature_request_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*AuditEvent
	for rows.Next() {
		event := &AuditEvent{}
		err := rows.Scan(
			&event.ID, &event.TenantID, &event.SignatureRequestID, &event.SignerID,
			&event.BatchID, &event.VerificationID, &event.EventType, &event.Details,
			&event.ActorType, &event.ActorID, &event.ActorIP, &event.ActorUserAgent,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// ===== Usage Tracking =====

// RecordUsage records signature usage for billing
func (r *Repository) RecordUsage(ctx context.Context, usage *Usage) error {
	if usage.ID == uuid.Nil {
		usage.ID = uuid.New()
	}

	query := `
		INSERT INTO signature_usage (
			id, tenant_id, signature_request_id, batch_id, signature_count,
			cost_cents, usage_date
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`

	return r.pool.QueryRow(ctx, query,
		usage.ID, usage.TenantID, usage.SignatureRequestID, usage.BatchID,
		usage.SignatureCount, usage.CostCents, usage.UsageDate,
	).Scan(&usage.CreatedAt)
}

// GetUsageByTenantAndMonth gets usage for a tenant in a specific month
func (r *Repository) GetUsageByTenantAndMonth(ctx context.Context, tenantID uuid.UUID, year, month int) (int, int, error) {
	query := `
		SELECT COALESCE(SUM(signature_count), 0), COALESCE(SUM(cost_cents), 0)
		FROM signature_usage
		WHERE tenant_id = $1
		  AND EXTRACT(YEAR FROM usage_date) = $2
		  AND EXTRACT(MONTH FROM usage_date) = $3
	`

	var totalSignatures, totalCostCents int
	err := r.pool.QueryRow(ctx, query, tenantID, year, month).Scan(&totalSignatures, &totalCostCents)
	return totalSignatures, totalCostCents, err
}

// ===== ID Austria Session Operations =====

// SaveIDAustriaSession saves an ID Austria session to the database
func (r *Repository) SaveIDAustriaSession(ctx context.Context, state, nonce, codeVerifier, redirectAfter string, signerID, batchID *uuid.UUID) error {
	query := `
		INSERT INTO idaustria_sessions (
			id, state, nonce, code_verifier, signer_id, batch_id, redirect_after
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		uuid.New(), state, nonce, codeVerifier, signerID, batchID, redirectAfter,
	)
	return err
}

// GetIDAustriaSessionByState retrieves a session by state
func (r *Repository) GetIDAustriaSessionByState(ctx context.Context, state string) (string, string, *uuid.UUID, *uuid.UUID, string, error) {
	query := `
		SELECT nonce, code_verifier, signer_id, batch_id, redirect_after
		FROM idaustria_sessions
		WHERE state = $1 AND status = 'pending'
	`

	var nonce, codeVerifier, redirectAfter string
	var signerID, batchID *uuid.UUID

	err := r.pool.QueryRow(ctx, query, state).Scan(&nonce, &codeVerifier, &signerID, &batchID, &redirectAfter)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", nil, nil, "", errors.New("session not found")
		}
		return "", "", nil, nil, "", err
	}

	return nonce, codeVerifier, signerID, batchID, redirectAfter, nil
}

// UpdateIDAustriaSessionAuthenticated marks a session as authenticated
func (r *Repository) UpdateIDAustriaSessionAuthenticated(ctx context.Context, state, subject, name, bpkHash string) error {
	query := `
		UPDATE idaustria_sessions
		SET status = 'authenticated', subject = $2, name = $3, bpk_hash = $4, updated_at = NOW()
		WHERE state = $1
	`
	_, err := r.pool.Exec(ctx, query, state, subject, name, bpkHash)
	return err
}

// ===== Helper Functions =====

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func itoa(i int) string {
	if i < 0 {
		return "-" + uitoa(uint(-i))
	}
	return uitoa(uint(i))
}

func uitoa(val uint) string {
	if val == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for val != 0 {
		buf[i] = byte('0' + val%10)
		val /= 10
		i--
	}
	return string(buf[i+1:])
}

// MarshalSignatureDetails marshals signature details to JSON
func MarshalSignatureDetails(details []SignatureDetails) (json.RawMessage, error) {
	return json.Marshal(details)
}

// UnmarshalSignatureDetails unmarshals signature details from JSON
func UnmarshalSignatureDetails(data json.RawMessage) ([]SignatureDetails, error) {
	var details []SignatureDetails
	err := json.Unmarshal(data, &details)
	return details, err
}
