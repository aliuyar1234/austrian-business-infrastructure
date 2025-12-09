package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/approval"
	"github.com/austrian-business-infrastructure/fo/internal/branding"
	"github.com/austrian-business-infrastructure/fo/internal/client"
	"github.com/austrian-business-infrastructure/fo/internal/clientgroup"
	"github.com/austrian-business-infrastructure/fo/internal/message"
	"github.com/austrian-business-infrastructure/fo/internal/share"
	"github.com/austrian-business-infrastructure/fo/internal/task"
	"github.com/austrian-business-infrastructure/fo/internal/upload"
)

// TestClientTypes verifies client package types compile correctly
func TestClientTypes(t *testing.T) {
	c := &client.Client{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "Test Client",
		Email:    "test@example.com",
		Status:   client.StatusActive,
	}

	if c.Name != "Test Client" {
		t.Error("Client name mismatch")
	}

	if c.Status != client.StatusActive {
		t.Error("Client status mismatch")
	}
}

// TestUploadTypes verifies upload package types compile correctly
func TestUploadTypes(t *testing.T) {
	cat := upload.CategoryRechnung
	u := &upload.Upload{
		ID:       uuid.New(),
		ClientID: uuid.New(),
		Filename: "test.pdf",
		FileSize: 1024,
		Category: &cat,
		Status:   upload.StatusNew,
	}

	if u.Filename != "test.pdf" {
		t.Error("Upload filename mismatch")
	}

	if u.Category == nil || *u.Category != upload.CategoryRechnung {
		t.Error("Upload category mismatch")
	}
}

// TestShareTypes verifies share package types compile correctly
func TestShareTypes(t *testing.T) {
	now := time.Now()
	s := &share.DocumentShare{
		ID:          uuid.New(),
		DocumentID:  uuid.New(),
		ClientID:    uuid.New(),
		SharedBy:    uuid.New(),
		CanDownload: true,
		SharedAt:    now,
	}

	if !s.CanDownload {
		t.Error("Share download flag mismatch")
	}
}

// TestApprovalTypes verifies approval package types compile correctly
func TestApprovalTypes(t *testing.T) {
	a := &approval.ApprovalRequest{
		ID:          uuid.New(),
		DocumentID:  uuid.New(),
		ClientID:    uuid.New(),
		RequestedBy: uuid.New(),
		Status:      approval.StatusPending,
	}

	if a.Status != approval.StatusPending {
		t.Error("Approval status mismatch")
	}

	// Test status validation
	if !approval.IsValidStatus("pending") {
		t.Error("Valid status not recognized")
	}

	if approval.IsValidStatus("invalid") {
		t.Error("Invalid status recognized")
	}
}

// TestMessageTypes verifies message package types compile correctly
func TestMessageTypes(t *testing.T) {
	thread := &message.Thread{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		ClientID: uuid.New(),
		Subject:  "Test Thread",
	}

	if thread.Subject != "Test Thread" {
		t.Error("Thread subject mismatch")
	}

	msg := &message.Message{
		ID:         uuid.New(),
		ThreadID:   thread.ID,
		SenderType: "client",
		SenderID:   uuid.New(),
		Content:    "Test message",
	}

	if msg.Content != "Test message" {
		t.Error("Message content mismatch")
	}
}

// TestTaskTypes verifies task package types compile correctly
func TestTaskTypes(t *testing.T) {
	dueDate := time.Now().Add(24 * time.Hour)
	tk := &task.ClientTask{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		ClientID: uuid.New(),
		Title:    "Test Task",
		Status:   task.StatusOpen,
		Priority: task.PriorityHigh,
		DueDate:  &dueDate,
	}

	if tk.Title != "Test Task" {
		t.Error("Task title mismatch")
	}

	if tk.Status != task.StatusOpen {
		t.Error("Task status mismatch")
	}

	if tk.Priority != task.PriorityHigh {
		t.Error("Task priority mismatch")
	}
}

// TestClientGroupTypes verifies clientgroup package types compile correctly
func TestClientGroupTypes(t *testing.T) {
	g := &clientgroup.ClientGroup{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "Test Group",
	}

	if g.Name != "Test Group" {
		t.Error("Group name mismatch")
	}

	m := &clientgroup.GroupMember{
		GroupID:  g.ID,
		ClientID: uuid.New(),
	}

	if m.GroupID != g.ID {
		t.Error("Member group ID mismatch")
	}
}

// TestBrandingTypes verifies branding package types compile correctly
func TestBrandingTypes(t *testing.T) {
	b := &branding.TenantBranding{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		CompanyName:  "Test Company",
		PrimaryColor: "#3B82F6",
	}

	if b.CompanyName != "Test Company" {
		t.Error("Branding company name mismatch")
	}

	if b.PrimaryColor != "#3B82F6" {
		t.Error("Branding primary color mismatch")
	}

	// Test default branding
	if branding.DefaultBranding.PrimaryColor == "" {
		t.Error("Default branding missing primary color")
	}
}

// TestClientAuth verifies client auth types
func TestClientAuth(t *testing.T) {
	claims := &client.ClientClaims{
		ClientID: uuid.New(),
		TenantID: uuid.New(),
		Email:    "test@example.com",
		IsClient: true,
	}

	if !claims.IsClient {
		t.Error("Claims should be marked as client")
	}

	if claims.Email != "test@example.com" {
		t.Error("Claims email mismatch")
	}
}
