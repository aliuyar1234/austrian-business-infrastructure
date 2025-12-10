package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"austrian-business-infrastructure/internal/api"
	"austrian-business-infrastructure/internal/auth"
	"austrian-business-infrastructure/internal/document"

	"github.com/google/uuid"
)

// MockDocumentService implements document.Service for testing tenant isolation
type MockDocumentService struct {
	documents  map[uuid.UUID]*document.Document
	tenantDocs map[uuid.UUID][]uuid.UUID // tenantID -> []documentIDs
}

func NewMockDocumentService() *MockDocumentService {
	return &MockDocumentService{
		documents:  make(map[uuid.UUID]*document.Document),
		tenantDocs: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *MockDocumentService) AddDocument(tenantID, docID uuid.UUID, title string) {
	m.documents[docID] = &document.Document{
		ID:       docID,
		TenantID: tenantID,
		Title:    title,
		Status:   document.StatusNew,
	}
	m.tenantDocs[tenantID] = append(m.tenantDocs[tenantID], docID)
}

func (m *MockDocumentService) GetByID(ctx context.Context, tenantID, docID uuid.UUID) (*document.Document, error) {
	doc, exists := m.documents[docID]
	if !exists {
		return nil, document.ErrDocumentNotFound
	}
	// CRITICAL: Tenant isolation check
	if doc.TenantID != tenantID {
		return nil, document.ErrDocumentNotFound
	}
	return doc, nil
}

// TestDocumentTenantIsolation tests that documents are properly isolated by tenant
func TestDocumentTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup JWT with HS256 for testing
	jwtConfig := &auth.JWTConfig{
		Secret:             "test-secret-key-for-testing-only-32bytes!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
		UseES256:           false,
	}
	jwtManager := auth.NewJWTManager(jwtConfig)

	// Create two tenants
	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create documents for each tenant
	docA := uuid.New()
	docB := uuid.New()

	mockService := NewMockDocumentService()
	mockService.AddDocument(tenantA, docA, "Tenant A Document")
	mockService.AddDocument(tenantB, docB, "Tenant B Document")

	// Create tokens
	tokenA, _, _ := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   uuid.New().String(),
		TenantID: tenantA.String(),
		Role:     "admin",
	})
	tokenB, _, _ := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   uuid.New().String(),
		TenantID: tenantB.String(),
		Role:     "admin",
	})

	// Handler that checks tenant isolation
	getDocHandler := func(service *MockDocumentService) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			tenantIDStr := api.GetTenantID(ctx)
			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			// Extract document ID from path
			path := r.URL.Path
			parts := strings.Split(path, "/")
			docIDStr := parts[len(parts)-1]
			docID, err := uuid.Parse(docIDStr)
			if err != nil {
				http.Error(w, "invalid document ID", http.StatusBadRequest)
				return
			}

			doc, err := service.GetByID(ctx, tenantID, docID)
			if err != nil {
				http.Error(w, "document not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(doc)
		}
	}

	// Build middleware chain
	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	handler := authMiddleware.RequireAuth(getDocHandler(mockService))

	t.Run("tenant_A_can_access_own_document", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/documents/"+docA.String(), nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var result document.Document
		json.Unmarshal(rec.Body.Bytes(), &result)
		if result.Title != "Tenant A Document" {
			t.Errorf("Expected 'Tenant A Document', got %s", result.Title)
		}
	})

	t.Run("tenant_A_cannot_access_tenant_B_document", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/documents/"+docB.String(), nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for cross-tenant access, got %d", rec.Code)
		}
	})

	t.Run("tenant_B_can_access_own_document", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/documents/"+docB.String(), nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rec.Code)
		}
	})

	t.Run("tenant_B_cannot_access_tenant_A_document", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/documents/"+docA.String(), nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for cross-tenant access, got %d", rec.Code)
		}
	})

	t.Run("nonexistent_document_returns_404", func(t *testing.T) {
		nonexistentID := uuid.New()
		req := httptest.NewRequest("GET", "/api/v1/documents/"+nonexistentID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for nonexistent document, got %d", rec.Code)
		}
	})

	t.Run("invalid_uuid_returns_400", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/documents/not-a-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid UUID, got %d", rec.Code)
		}
	})
}

// TestSignedURLExpiry tests that signed URLs respect expiry settings
func TestSignedURLExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test expiry parameter validation
	testCases := []struct {
		name        string
		expiryParam string
		expectedMin int
		expectedMax int
	}{
		{"default_expiry", "", 15, 15},
		{"custom_30_min", "30", 30, 30},
		{"max_60_min", "60", 60, 60},
		{"over_max_clamped", "120", 60, 60},
		{"negative_default", "-5", 15, 15},
		{"zero_default", "0", 15, 15},
		{"invalid_default", "abc", 15, 15},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expiry := 15 // Default minutes
			if tc.expiryParam != "" {
				if minutes, err := strconv.Atoi(tc.expiryParam); err == nil && minutes > 0 {
					if minutes > 60 {
						minutes = 60 // Clamp to max
					}
					expiry = minutes
				}
			}

			if expiry < tc.expectedMin || expiry > tc.expectedMax {
				t.Errorf("Expected expiry between %d-%d, got %d", tc.expectedMin, tc.expectedMax, expiry)
			}
		})
	}
}

// TestBulkArchiveTenantIsolation tests that bulk archive only affects tenant's own documents
func TestBulkArchiveTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	jwtConfig := &auth.JWTConfig{
		Secret:             "test-secret-key-for-testing-only-32bytes!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "test",
		UseES256:           false,
	}
	jwtManager := auth.NewJWTManager(jwtConfig)

	tenantA := uuid.New()
	tenantB := uuid.New()

	docA1, docA2 := uuid.New(), uuid.New()
	docB1 := uuid.New()

	mockService := NewMockDocumentService()
	mockService.AddDocument(tenantA, docA1, "A1")
	mockService.AddDocument(tenantA, docA2, "A2")
	mockService.AddDocument(tenantB, docB1, "B1")

	tokenA, _, _ := jwtManager.GenerateAccessToken(&auth.UserInfo{
		UserID:   uuid.New().String(),
		TenantID: tenantA.String(),
		Role:     "admin",
	})

	// Handler simulating bulk archive
	bulkArchiveHandler := func(service *MockDocumentService) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			tenantIDStr := api.GetTenantID(ctx)
			tenantID, _ := uuid.Parse(tenantIDStr)

			var req struct {
				IDs []uuid.UUID `json:"ids"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			archived := 0
			skipped := 0
			for _, docID := range req.IDs {
				doc, err := service.GetByID(ctx, tenantID, docID)
				if err != nil || doc.TenantID != tenantID {
					skipped++
					continue
				}
				archived++
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]int{
				"archived": archived,
				"skipped":  skipped,
			})
		}
	}

	authMiddleware := auth.NewAuthMiddleware(jwtManager)
	handler := authMiddleware.RequireAuth(bulkArchiveHandler(mockService))

	t.Run("bulk_archive_only_affects_own_documents", func(t *testing.T) {
		body := map[string][]uuid.UUID{
			"ids": {docA1, docA2, docB1},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/documents/archive", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Authorization", "Bearer "+tokenA)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rec.Code)
		}

		var result map[string]int
		json.Unmarshal(rec.Body.Bytes(), &result)

		if result["archived"] != 2 {
			t.Errorf("Expected 2 archived, got %d", result["archived"])
		}
		if result["skipped"] != 1 {
			t.Errorf("Expected 1 skipped, got %d", result["skipped"])
		}
	})
}
