package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Event types
const (
	EventNewDocument     = "new_document"
	EventDeadlineWarning = "deadline_warning"
	EventFBChange        = "fb_change"
	EventSyncComplete    = "sync_complete"
	EventDocumentRead    = "document_read"
)

// Event represents a webhook event payload
type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	TenantID  string      `json:"tenant_id"`
	Data      interface{} `json:"data"`
}

// Service handles webhook delivery logic
type Service struct {
	repo       *Repository
	httpClient *http.Client
	logger     *slog.Logger
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	Logger        *slog.Logger
	DefaultTimeout time.Duration
}

// NewService creates a new webhook service
func NewService(repo *Repository, cfg *ServiceConfig) *Service {
	timeout := 30 * time.Second
	logger := slog.Default()

	if cfg != nil {
		if cfg.DefaultTimeout > 0 {
			timeout = cfg.DefaultTimeout
		}
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
	}

	return &Service{
		repo: repo,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// TriggerEvent triggers webhooks for a specific event
func (s *Service) TriggerEvent(ctx context.Context, tenantID uuid.UUID, eventType string, data interface{}) error {
	// Get webhooks subscribed to this event
	webhooks, err := s.repo.ListByEvent(ctx, tenantID, eventType)
	if err != nil {
		return fmt.Errorf("list webhooks: %w", err)
	}

	if len(webhooks) == 0 {
		return nil // No webhooks subscribed
	}

	// Create event payload
	event := &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now(),
		TenantID:  tenantID.String(),
		Data:      data,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Queue deliveries for each webhook
	for _, wh := range webhooks {
		delivery := &Delivery{
			WebhookID:    wh.ID,
			TenantID:     tenantID,
			EventType:    eventType,
			Payload:      eventJSON,
			Status:       "pending",
			AttemptCount: 0,
		}

		if err := s.repo.CreateDelivery(ctx, delivery); err != nil {
			s.logger.Error("failed to create delivery",
				"webhook_id", wh.ID,
				"event_type", eventType,
				"error", err)
			continue
		}

		s.logger.Debug("delivery queued",
			"webhook_id", wh.ID,
			"delivery_id", delivery.ID,
			"event_type", eventType)
	}

	return nil
}

// ProcessPendingDeliveries processes pending webhook deliveries
func (s *Service) ProcessPendingDeliveries(ctx context.Context, batchSize int) (int, error) {
	deliveries, err := s.repo.GetPendingDeliveries(ctx, batchSize)
	if err != nil {
		return 0, fmt.Errorf("get pending deliveries: %w", err)
	}

	processed := 0
	for _, d := range deliveries {
		if err := s.deliverWebhook(ctx, d); err != nil {
			s.logger.Error("delivery failed", "delivery_id", d.ID, "error", err)
		}
		processed++
	}

	return processed, nil
}

// deliverWebhook attempts to deliver a webhook
func (s *Service) deliverWebhook(ctx context.Context, d *Delivery) error {
	// Get webhook configuration
	wh, err := s.repo.GetByID(ctx, d.WebhookID)
	if err != nil {
		return fmt.Errorf("get webhook: %w", err)
	}

	// Prepare payload
	payloadBytes, ok := d.Payload.([]byte)
	if !ok {
		payloadBytes, _ = json.Marshal(d.Payload)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.handleDeliveryError(ctx, d, wh, err, nil)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AustrianBusinessPlatform-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", wh.ID.String())
	req.Header.Set("X-Delivery-ID", d.ID.String())
	req.Header.Set("X-Event-Type", d.EventType)

	// Add signature
	signature := s.generateSignature(payloadBytes, wh.Secret)
	req.Header.Set("X-Webhook-Signature", signature)

	// Add custom headers
	for k, v := range wh.Headers {
		req.Header.Set(k, v)
	}

	// Create client with webhook-specific timeout
	client := &http.Client{
		Timeout: time.Duration(wh.TimeoutSeconds) * time.Second,
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return s.handleDeliveryError(ctx, d, wh, err, nil)
	}
	defer resp.Body.Close()

	// Read response body (limited)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024)) // Max 64KB

	// Check for success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success
		headers := make(map[string]string)
		for k, v := range resp.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		if err := s.repo.UpdateDeliverySuccess(ctx, d.ID, resp.StatusCode, string(body), headers); err != nil {
			s.logger.Error("failed to update delivery success", "delivery_id", d.ID, "error", err)
		}

		s.logger.Info("webhook delivered",
			"webhook_id", wh.ID,
			"delivery_id", d.ID,
			"status", resp.StatusCode)

		return nil
	}

	// Non-2xx response
	return s.handleDeliveryError(ctx, d, wh, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)), &resp.StatusCode)
}

// handleDeliveryError handles a failed delivery attempt
func (s *Service) handleDeliveryError(ctx context.Context, d *Delivery, wh *Webhook, err error, statusCode *int) error {
	d.AttemptCount++

	var nextRetryAt *time.Time
	if d.AttemptCount < wh.MaxRetries {
		// Calculate exponential backoff
		delay := time.Duration(1<<uint(d.AttemptCount)) * time.Second // 1s, 2s, 4s, 8s, ...
		retryAt := time.Now().Add(delay)
		nextRetryAt = &retryAt
	}

	if err := s.repo.UpdateDeliveryFailure(ctx, d.ID, err.Error(), statusCode, nextRetryAt); err != nil {
		s.logger.Error("failed to update delivery failure", "delivery_id", d.ID, "error", err)
	}

	if nextRetryAt != nil {
		s.logger.Info("webhook delivery scheduled for retry",
			"webhook_id", wh.ID,
			"delivery_id", d.ID,
			"attempt", d.AttemptCount,
			"retry_at", nextRetryAt)
	} else {
		s.logger.Warn("webhook delivery failed permanently",
			"webhook_id", wh.ID,
			"delivery_id", d.ID,
			"attempts", d.AttemptCount,
			"error", err)
	}

	return err
}

// generateSignature generates HMAC-SHA256 signature for the payload
func (s *Service) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies a webhook signature
func VerifySignature(payload []byte, signature, secret string) bool {
	expected := "sha256=" + hex.EncodeToString(hmacSHA256(payload, secret))
	return hmac.Equal([]byte(signature), []byte(expected))
}

func hmacSHA256(data []byte, key string) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return mac.Sum(nil)
}
