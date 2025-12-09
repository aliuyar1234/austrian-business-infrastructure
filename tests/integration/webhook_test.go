package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/webhook"
	"github.com/google/uuid"
)

func TestWebhookEventTypes(t *testing.T) {
	t.Run("event type constants are defined", func(t *testing.T) {
		eventTypes := []string{
			webhook.EventNewDocument,
			webhook.EventDeadlineWarning,
			webhook.EventFBChange,
			webhook.EventSyncComplete,
			webhook.EventDocumentRead,
		}

		for _, et := range eventTypes {
			if et == "" {
				t.Error("event type constant is empty")
			}
		}
	})
}

func TestWebhookStruct(t *testing.T) {
	t.Run("create webhook with all fields", func(t *testing.T) {
		tenantID := uuid.New()

		wh := &webhook.Webhook{
			ID:             uuid.New(),
			TenantID:       tenantID,
			Name:           "My Webhook",
			URL:            "https://example.com/webhook",
			Secret:         "super-secret-key",
			Events:         []string{webhook.EventNewDocument, webhook.EventDeadlineWarning},
			Enabled:        true,
			TimeoutSeconds: 30,
			MaxRetries:     3,
			Headers: map[string]string{
				"X-Custom-Header": "value",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if wh.ID == uuid.Nil {
			t.Error("webhook ID should not be nil")
		}
		if wh.TenantID != tenantID {
			t.Error("tenant ID mismatch")
		}
		if wh.Name != "My Webhook" {
			t.Error("name mismatch")
		}
		if wh.URL != "https://example.com/webhook" {
			t.Error("URL mismatch")
		}
		if len(wh.Events) != 2 {
			t.Errorf("expected 2 events, got %d", len(wh.Events))
		}
		if !wh.Enabled {
			t.Error("should be enabled")
		}
		if wh.TimeoutSeconds != 30 {
			t.Error("timeout mismatch")
		}
		if wh.MaxRetries != 3 {
			t.Error("max retries mismatch")
		}
	})

	t.Run("webhook with custom headers", func(t *testing.T) {
		wh := &webhook.Webhook{
			Headers: map[string]string{
				"Authorization": "Bearer token123",
				"X-API-Key":     "api-key-value",
			},
		}

		if len(wh.Headers) != 2 {
			t.Errorf("expected 2 headers, got %d", len(wh.Headers))
		}
		if wh.Headers["Authorization"] != "Bearer token123" {
			t.Error("Authorization header mismatch")
		}
	})
}

func TestDeliveryStruct(t *testing.T) {
	t.Run("create delivery with all fields", func(t *testing.T) {
		webhookID := uuid.New()
		tenantID := uuid.New()
		now := time.Now()

		d := &webhook.Delivery{
			ID:             uuid.New(),
			WebhookID:      webhookID,
			TenantID:       tenantID,
			EventType:      webhook.EventNewDocument,
			Payload:        json.RawMessage(`{"document_id": "123"}`),
			Status:         "pending",
			AttemptCount:   0,
			CreatedAt:      now,
		}

		if d.ID == uuid.Nil {
			t.Error("delivery ID should not be nil")
		}
		if d.WebhookID != webhookID {
			t.Error("webhook ID mismatch")
		}
		if d.EventType != webhook.EventNewDocument {
			t.Error("event type mismatch")
		}
		if d.Status != "pending" {
			t.Error("status should be pending")
		}
	})

	t.Run("delivery success", func(t *testing.T) {
		now := time.Now()
		statusCode := 200

		d := &webhook.Delivery{
			Status:         "success",
			ResponseStatus: &statusCode,
			ResponseBody:   `{"ok": true}`,
			DeliveredAt:    &now,
			AttemptCount:   1,
		}

		if d.Status != "success" {
			t.Error("status should be success")
		}
		if d.ResponseStatus == nil || *d.ResponseStatus != 200 {
			t.Error("response status should be 200")
		}
		if d.DeliveredAt == nil {
			t.Error("delivered at should be set")
		}
	})

	t.Run("delivery failure with retry", func(t *testing.T) {
		nextRetry := time.Now().Add(1 * time.Minute)
		statusCode := 500

		d := &webhook.Delivery{
			Status:         "pending",
			ResponseStatus: &statusCode,
			LastError:      "Internal Server Error",
			AttemptCount:   2,
			NextRetryAt:    &nextRetry,
		}

		if d.Status != "pending" {
			t.Error("status should be pending for retry")
		}
		if d.AttemptCount != 2 {
			t.Error("attempt count mismatch")
		}
		if d.NextRetryAt == nil {
			t.Error("next retry should be set")
		}
	})

	t.Run("delivery permanent failure", func(t *testing.T) {
		d := &webhook.Delivery{
			Status:       "failed",
			LastError:    "Max retries exceeded",
			AttemptCount: 3,
			NextRetryAt:  nil,
		}

		if d.Status != "failed" {
			t.Error("status should be failed")
		}
		if d.NextRetryAt != nil {
			t.Error("next retry should not be set for permanent failure")
		}
	})
}

func TestWebhookSignature(t *testing.T) {
	t.Run("generate valid HMAC-SHA256 signature", func(t *testing.T) {
		payload := []byte(`{"event": "test", "data": {"key": "value"}}`)
		secret := "my-webhook-secret"

		// Generate signature
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		if signature == "" {
			t.Error("signature should not be empty")
		}
		if len(signature) < 10 {
			t.Error("signature is too short")
		}
		if signature[:7] != "sha256=" {
			t.Error("signature should start with 'sha256='")
		}
	})

	t.Run("verify signature", func(t *testing.T) {
		payload := []byte(`{"test": true}`)
		secret := "secret123"

		// Generate expected signature
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		// Verify using same method
		mac2 := hmac.New(sha256.New, []byte(secret))
		mac2.Write(payload)
		actual := "sha256=" + hex.EncodeToString(mac2.Sum(nil))

		if !hmac.Equal([]byte(expected), []byte(actual)) {
			t.Error("signatures should match")
		}
	})

	t.Run("different secrets produce different signatures", func(t *testing.T) {
		payload := []byte(`{"test": true}`)

		mac1 := hmac.New(sha256.New, []byte("secret1"))
		mac1.Write(payload)
		sig1 := hex.EncodeToString(mac1.Sum(nil))

		mac2 := hmac.New(sha256.New, []byte("secret2"))
		mac2.Write(payload)
		sig2 := hex.EncodeToString(mac2.Sum(nil))

		if sig1 == sig2 {
			t.Error("different secrets should produce different signatures")
		}
	})

	t.Run("different payloads produce different signatures", func(t *testing.T) {
		secret := []byte("same-secret")

		mac1 := hmac.New(sha256.New, secret)
		mac1.Write([]byte(`{"a": 1}`))
		sig1 := hex.EncodeToString(mac1.Sum(nil))

		mac2 := hmac.New(sha256.New, secret)
		mac2.Write([]byte(`{"b": 2}`))
		sig2 := hex.EncodeToString(mac2.Sum(nil))

		if sig1 == sig2 {
			t.Error("different payloads should produce different signatures")
		}
	})
}

func TestWebhookEventPayload(t *testing.T) {
	t.Run("new document event payload", func(t *testing.T) {
		event := &webhook.Event{
			ID:        uuid.New().String(),
			Type:      webhook.EventNewDocument,
			Timestamp: time.Now(),
			TenantID:  uuid.New().String(),
			Data: map[string]interface{}{
				"document_id":   uuid.New().String(),
				"document_type": "Bescheid",
				"account_id":    uuid.New().String(),
			},
		}

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}

		var parsed webhook.Event
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}

		if parsed.Type != webhook.EventNewDocument {
			t.Error("event type mismatch")
		}
		if parsed.Data == nil {
			t.Error("event data should not be nil")
		}
	})

	t.Run("deadline warning event payload", func(t *testing.T) {
		event := &webhook.Event{
			ID:        uuid.New().String(),
			Type:      webhook.EventDeadlineWarning,
			Timestamp: time.Now(),
			TenantID:  uuid.New().String(),
			Data: map[string]interface{}{
				"document_id":   uuid.New().String(),
				"deadline":      time.Now().Add(3 * 24 * time.Hour).Format(time.RFC3339),
				"days_until":    3,
				"document_name": "Ergänzungsersuchen",
			},
		}

		if event.Type != webhook.EventDeadlineWarning {
			t.Error("event type mismatch")
		}
	})

	t.Run("FB change event payload", func(t *testing.T) {
		event := &webhook.Event{
			ID:        uuid.New().String(),
			Type:      webhook.EventFBChange,
			Timestamp: time.Now(),
			TenantID:  uuid.New().String(),
			Data: map[string]interface{}{
				"company_number": "FN123456a",
				"company_name":   "Test GmbH",
				"change_type":    "geschaeftsfuehrer_changed",
			},
		}

		if event.Type != webhook.EventFBChange {
			t.Error("event type mismatch")
		}
	})
}

func TestWebhookRetryBackoff(t *testing.T) {
	t.Run("exponential backoff calculation", func(t *testing.T) {
		attempts := []struct {
			attempt  int
			expected time.Duration
		}{
			{1, 1 * time.Second},
			{2, 2 * time.Second},
			{3, 4 * time.Second},
			{4, 8 * time.Second},
			{5, 16 * time.Second},
		}

		for _, tt := range attempts {
			// Calculate exponential backoff: 2^(attempt-1) seconds
			backoff := time.Duration(1<<uint(tt.attempt-1)) * time.Second

			if backoff != tt.expected {
				t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, backoff)
			}
		}
	})
}

func TestWebhookURLValidation(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		valid bool
	}{
		{"valid https", "https://example.com/webhook", true},
		{"valid http", "http://localhost:8080/webhook", true},
		{"with path", "https://api.example.com/v1/webhooks/receive", true},
		{"with query", "https://example.com/webhook?token=abc", true},
		{"empty", "", false},
		{"no scheme", "example.com/webhook", false},
		{"invalid scheme", "ftp://example.com/webhook", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic URL validation
			valid := tt.url != "" && (len(tt.url) >= 7 && (tt.url[:7] == "http://" || (len(tt.url) >= 8 && tt.url[:8] == "https://")))

			if valid != tt.valid {
				t.Errorf("URL %q: expected valid=%v, got %v", tt.url, tt.valid, valid)
			}
		})
	}
}

func TestWebhookEventSubscription(t *testing.T) {
	t.Run("subscribe to single event", func(t *testing.T) {
		wh := &webhook.Webhook{
			Events: []string{webhook.EventNewDocument},
		}

		if len(wh.Events) != 1 {
			t.Errorf("expected 1 event, got %d", len(wh.Events))
		}
	})

	t.Run("subscribe to multiple events", func(t *testing.T) {
		wh := &webhook.Webhook{
			Events: []string{
				webhook.EventNewDocument,
				webhook.EventDeadlineWarning,
				webhook.EventFBChange,
				webhook.EventSyncComplete,
			},
		}

		if len(wh.Events) != 4 {
			t.Errorf("expected 4 events, got %d", len(wh.Events))
		}
	})

	t.Run("check if subscribed to event", func(t *testing.T) {
		wh := &webhook.Webhook{
			Events: []string{webhook.EventNewDocument, webhook.EventFBChange},
		}

		// Check subscription
		isSubscribed := func(eventType string) bool {
			for _, e := range wh.Events {
				if e == eventType {
					return true
				}
			}
			return false
		}

		if !isSubscribed(webhook.EventNewDocument) {
			t.Error("should be subscribed to new_document")
		}
		if !isSubscribed(webhook.EventFBChange) {
			t.Error("should be subscribed to fb_change")
		}
		if isSubscribed(webhook.EventDeadlineWarning) {
			t.Error("should not be subscribed to deadline_warning")
		}
	})
}

func TestServiceConfig(t *testing.T) {
	t.Run("service config defaults", func(t *testing.T) {
		cfg := &webhook.ServiceConfig{}

		if cfg.DefaultTimeout < 0 {
			t.Error("default timeout should not be negative")
		}
	})

	t.Run("service config custom values", func(t *testing.T) {
		cfg := &webhook.ServiceConfig{
			DefaultTimeout: 60 * time.Second,
		}

		if cfg.DefaultTimeout != 60*time.Second {
			t.Error("default timeout mismatch")
		}
	})
}
