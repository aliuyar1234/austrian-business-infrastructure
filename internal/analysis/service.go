package analysis

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"austrian-business-infrastructure/internal/ai"
	"austrian-business-infrastructure/internal/document"
	"austrian-business-infrastructure/internal/ocr"
)

// Service orchestrates document analysis
type Service struct {
	repo        *Repository
	docService  *document.Service
	ocrService  *ocr.Service
	classifier  *Classifier
	extractor   *Extractor
	aiClient    *ai.Client
	maxCost     float64
	enabled     bool
}

// ServiceConfig holds analysis service configuration
type ServiceConfig struct {
	AIClient      *ai.Client
	PromptLoader  *ai.PromptLoader
	OCRService    *ocr.Service
	DocService    *document.Service
	MaxCostPerDoc float64
	Enabled       bool
}

// NewService creates a new analysis service
func NewService(repo *Repository, cfg ServiceConfig) *Service {
	return &Service{
		repo:       repo,
		docService: cfg.DocService,
		ocrService: cfg.OCRService,
		classifier: NewClassifier(cfg.AIClient, cfg.PromptLoader),
		extractor:  NewExtractor(cfg.AIClient, cfg.PromptLoader),
		aiClient:   cfg.AIClient,
		maxCost:    cfg.MaxCostPerDoc,
		enabled:    cfg.Enabled,
	}
}

// AnalysisOptions configures what analysis to perform
type AnalysisOptions struct {
	IncludeOCR         bool `json:"include_ocr"`
	IncludeClassify    bool `json:"include_classify"`
	IncludeSummary     bool `json:"include_summary"`
	IncludeDeadlines   bool `json:"include_deadlines"`
	IncludeAmounts     bool `json:"include_amounts"`
	IncludeActionItems bool `json:"include_action_items"`
	IncludeSuggestions bool `json:"include_suggestions"`
}

// DefaultOptions returns the default analysis options
func DefaultOptions() AnalysisOptions {
	return AnalysisOptions{
		IncludeOCR:         true,
		IncludeClassify:    true,
		IncludeSummary:     true,
		IncludeDeadlines:   true,
		IncludeAmounts:     true,
		IncludeActionItems: true,
		IncludeSuggestions: true,
	}
}

// FullAnalysisResult contains all analysis results
type FullAnalysisResult struct {
	Analysis    *Analysis             `json:"analysis"`
	Deadlines   []*Deadline           `json:"deadlines,omitempty"`
	Amounts     []*Amount             `json:"amounts,omitempty"`
	ActionItems []*ActionItem         `json:"action_items,omitempty"`
	Suggestions []*Suggestion         `json:"suggestions,omitempty"`
	Warnings    []ConfidenceWarning   `json:"warnings,omitempty"`
}

// ConfidenceWarning represents a low confidence warning
type ConfidenceWarning struct {
	Type        string  `json:"type"`        // classification, deadline, amount, action_item
	ItemID      string  `json:"item_id"`     // ID of the item with low confidence
	Field       string  `json:"field"`       // Which field has low confidence
	Confidence  float64 `json:"confidence"`
	Message     string  `json:"message"`
	Severity    string  `json:"severity"`    // low, medium, high
}

// Confidence thresholds
const (
	ConfidenceHigh   = 0.8
	ConfidenceMedium = 0.5
	ConfidenceLow    = 0.3
)

// GenerateConfidenceWarnings analyzes the result and generates warnings for low confidence items
func (r *FullAnalysisResult) GenerateConfidenceWarnings() {
	r.Warnings = nil

	// Check classification confidence
	if r.Analysis != nil && r.Analysis.ClassificationConfidence > 0 {
		if r.Analysis.ClassificationConfidence < ConfidenceMedium {
			r.Warnings = append(r.Warnings, ConfidenceWarning{
				Type:       "classification",
				ItemID:     r.Analysis.ID.String(),
				Field:      "document_type",
				Confidence: r.Analysis.ClassificationConfidence,
				Message:    "Dokumentenklassifizierung unsicher - manuelle Überprüfung empfohlen",
				Severity:   getSeverity(r.Analysis.ClassificationConfidence),
			})
		}
	}

	// Check OCR confidence
	if r.Analysis != nil && r.Analysis.IsScanned && r.Analysis.OCRConfidence > 0 {
		if r.Analysis.OCRConfidence < ConfidenceMedium {
			r.Warnings = append(r.Warnings, ConfidenceWarning{
				Type:       "ocr",
				ItemID:     r.Analysis.ID.String(),
				Field:      "extracted_text",
				Confidence: r.Analysis.OCRConfidence,
				Message:    "OCR-Qualität niedrig - Text könnte unvollständig oder fehlerhaft sein",
				Severity:   getSeverity(r.Analysis.OCRConfidence),
			})
		}
	}

	// Check deadline confidence
	for _, d := range r.Deadlines {
		if d.Confidence < ConfidenceHigh {
			severity := getSeverity(d.Confidence)
			msg := "Frist könnte ungenau sein"
			if d.Confidence < ConfidenceMedium {
				msg = "ACHTUNG: Frist sehr unsicher - unbedingt manuell prüfen!"
			}
			r.Warnings = append(r.Warnings, ConfidenceWarning{
				Type:       "deadline",
				ItemID:     d.ID.String(),
				Field:      "date",
				Confidence: d.Confidence,
				Message:    msg,
				Severity:   severity,
			})
		}
	}

	// Check amount confidence
	for _, a := range r.Amounts {
		if a.Confidence < ConfidenceHigh {
			r.Warnings = append(r.Warnings, ConfidenceWarning{
				Type:       "amount",
				ItemID:     a.ID.String(),
				Field:      "amount",
				Confidence: a.Confidence,
				Message:    "Betrag könnte ungenau extrahiert worden sein",
				Severity:   getSeverity(a.Confidence),
			})
		}
	}

	// Check action item confidence
	for _, ai := range r.ActionItems {
		if ai.Confidence < ConfidenceHigh {
			r.Warnings = append(r.Warnings, ConfidenceWarning{
				Type:       "action_item",
				ItemID:     ai.ID.String(),
				Field:      "priority",
				Confidence: ai.Confidence,
				Message:    "Priorität der Aufgabe könnte ungenau sein",
				Severity:   getSeverity(ai.Confidence),
			})
		}
	}
}

func getSeverity(confidence float64) string {
	if confidence < ConfidenceLow {
		return "high"
	}
	if confidence < ConfidenceMedium {
		return "medium"
	}
	return "low"
}

// AnalyzeDocument performs full document analysis
func (s *Service) AnalyzeDocument(ctx context.Context, documentID, tenantID uuid.UUID, opts AnalysisOptions) (*FullAnalysisResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("AI analysis is disabled")
	}

	startTime := time.Now()

	// Create analysis record
	analysis := &Analysis{
		DocumentID: documentID,
		TenantID:   tenantID,
		Status:     StatusProcessing,
	}

	if err := s.repo.CreateAnalysis(ctx, analysis); err != nil {
		return nil, fmt.Errorf("create analysis: %w", err)
	}

	// Get document content with tenant isolation
	content, storageInfo, err := s.docService.GetContent(ctx, tenantID, documentID)
	if err != nil {
		s.failAnalysis(ctx, analysis, "document_not_found", err.Error())
		return nil, fmt.Errorf("get document: %w", err)
	}
	defer content.Close()

	// Read content
	data, err := io.ReadAll(content)
	if err != nil {
		s.failAnalysis(ctx, analysis, "read_error", err.Error())
		return nil, fmt.Errorf("read content: %w", err)
	}

	// Get document metadata with tenant isolation
	doc, err := s.docService.GetByID(ctx, tenantID, documentID)
	if err != nil {
		s.failAnalysis(ctx, analysis, "document_not_found", err.Error())
		return nil, fmt.Errorf("get document metadata: %w", err)
	}

	var text string

	// Step 1: OCR/Text Extraction
	if opts.IncludeOCR && storageInfo.ContentType == "application/pdf" {
		ocrResult, err := s.ocrService.ProcessBytes(ctx, data)
		if err != nil {
			// Log OCR error but continue with what we have
			analysis.ErrorMessage = fmt.Sprintf("OCR warning: %v", err)
		} else {
			text = ocrResult.Text
			analysis.IsScanned = ocrResult.Provider != ocr.ProviderNone
			analysis.OCRProvider = string(ocrResult.Provider)
			analysis.OCRConfidence = ocrResult.Confidence
			analysis.PageCount = len(ocrResult.PageTexts)
		}
	}

	// If no text from OCR, try direct extraction
	if text == "" {
		extracted, err := ocr.ExtractPDFTextFromBytes(data)
		if err == nil {
			text = extracted
		}
	}

	analysis.ExtractedText = text
	analysis.TextLength = len(text)

	// If we still have no text, fail
	if text == "" {
		s.failAnalysis(ctx, analysis, "no_text", "Could not extract text from document")
		return nil, fmt.Errorf("no text could be extracted from document")
	}

	result := &FullAnalysisResult{
		Analysis: analysis,
	}

	// Step 2: Classification
	var classification *ClassificationResult
	if opts.IncludeClassify {
		classification, err = s.classifier.ClassifyWithFallback(ctx, text, doc.Title)
		if err != nil {
			// Non-fatal, continue with default
			classification = &ClassificationResult{
				DocumentType: DocTypeSonstige,
				Confidence:   0.5,
			}
		}

		analysis.DocumentType = string(classification.DocumentType)
		analysis.DocumentSubtype = string(classification.DocumentSubtype)
		analysis.ClassificationConfidence = classification.Confidence
	}

	// Step 3: Summary
	if opts.IncludeSummary {
		summary, err := s.extractor.Summarize(ctx, text)
		if err == nil {
			analysis.Summary = summary.Summary
			analysis.KeyPoints = summary.KeyPoints
			analysis.Language = summary.Language
		}
	}

	// Step 4: Deadlines
	if opts.IncludeDeadlines {
		extractedDeadlines, err := s.extractor.ExtractDeadlines(ctx, text)
		if err == nil && len(extractedDeadlines) > 0 {
			for _, ed := range extractedDeadlines {
				deadline := &Deadline{
					AnalysisID:   analysis.ID,
					DocumentID:   documentID,
					TenantID:     tenantID,
					DeadlineType: ed.Type,
					Date:         ed.Date,
					Description:  ed.Description,
					SourceText:   ed.SourceText,
					Confidence:   ed.Confidence,
					IsHard:       ed.IsHard,
				}
				if err := s.repo.CreateDeadline(ctx, deadline); err == nil {
					result.Deadlines = append(result.Deadlines, deadline)
				}
			}
		}
	}

	// Step 5: Amounts
	if opts.IncludeAmounts {
		extractedAmounts, err := s.extractor.ExtractAmounts(ctx, text)
		if err == nil && len(extractedAmounts) > 0 {
			for _, ea := range extractedAmounts {
				amount := &Amount{
					AnalysisID:  analysis.ID,
					DocumentID:  documentID,
					TenantID:    tenantID,
					AmountType:  ea.Type,
					Amount:      ea.Amount,
					Currency:    ea.Currency,
					Description: ea.Description,
					SourceText:  ea.SourceText,
					Confidence:  ea.Confidence,
					DueDate:     ea.DueDate,
				}
				if err := s.repo.CreateAmount(ctx, amount); err == nil {
					result.Amounts = append(result.Amounts, amount)
				}
			}
		}
	}

	// Step 6: Action Items
	if opts.IncludeActionItems && classification != nil {
		var deadlinesForItems []ExtractedDeadline
		for _, d := range result.Deadlines {
			deadlinesForItems = append(deadlinesForItems, ExtractedDeadline{
				Type:        d.DeadlineType,
				Date:        d.Date,
				Description: d.Description,
				SourceText:  d.SourceText,
				Confidence:  d.Confidence,
				IsHard:      d.IsHard,
			})
		}

		actionItems, err := s.extractor.ExtractActionItems(ctx, text, classification, deadlinesForItems)
		if err == nil && len(actionItems) > 0 {
			for _, ai := range actionItems {
				item := &ActionItem{
					AnalysisID:  analysis.ID,
					DocumentID:  documentID,
					TenantID:    tenantID,
					Title:       ai.Title,
					Description: ai.Description,
					Priority:    Priority(ai.Priority),
					Category:    ai.Category,
					Status:      ActionStatusPending,
					DueDate:     ai.DueDate,
					SourceText:  ai.SourceText,
					Confidence:  ai.Confidence,
				}
				if err := s.repo.CreateActionItem(ctx, item); err == nil {
					result.ActionItems = append(result.ActionItems, item)
				}
			}
		}
	}

	// Step 7: Response Suggestions
	if opts.IncludeSuggestions && classification != nil {
		suggestions, err := s.extractor.GenerateSuggestions(ctx, text, classification)
		if err == nil && len(suggestions) > 0 {
			for _, sg := range suggestions {
				sugg := &Suggestion{
					AnalysisID:     analysis.ID,
					DocumentID:     documentID,
					TenantID:       tenantID,
					SuggestionType: sg.Type,
					Title:          sg.Title,
					Content:        sg.Content,
					Reasoning:      sg.Reasoning,
					Confidence:     sg.Confidence,
				}
				if err := s.repo.CreateSuggestion(ctx, sugg); err == nil {
					result.Suggestions = append(result.Suggestions, sugg)
				}
			}
		}
	}

	// Finalize analysis
	analysis.Status = StatusCompleted
	analysis.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
	now := time.Now()
	analysis.CompletedAt = &now

	if err := s.repo.UpdateAnalysis(ctx, analysis); err != nil {
		return nil, fmt.Errorf("update analysis: %w", err)
	}

	// Generate confidence warnings for low-confidence items
	result.GenerateConfidenceWarnings()

	return result, nil
}

// failAnalysis marks an analysis as failed
func (s *Service) failAnalysis(ctx context.Context, analysis *Analysis, code, message string) {
	analysis.Status = StatusFailed
	analysis.ErrorCode = code
	analysis.ErrorMessage = message
	s.repo.UpdateAnalysis(ctx, analysis)
}

// GetAnalysis retrieves an analysis by ID
func (s *Service) GetAnalysis(ctx context.Context, id uuid.UUID) (*Analysis, error) {
	return s.repo.GetAnalysisByID(ctx, id)
}

// GetAnalysisByDocument retrieves the latest analysis for a document
func (s *Service) GetAnalysisByDocument(ctx context.Context, documentID uuid.UUID) (*Analysis, error) {
	return s.repo.GetAnalysisByDocumentID(ctx, documentID)
}

// GetFullAnalysis retrieves full analysis results for a document
func (s *Service) GetFullAnalysis(ctx context.Context, documentID uuid.UUID) (*FullAnalysisResult, error) {
	analysis, err := s.repo.GetAnalysisByDocumentID(ctx, documentID)
	if err != nil {
		return nil, err
	}

	deadlines, _ := s.repo.GetDeadlinesByDocument(ctx, documentID)
	amounts, _ := s.repo.GetAmountsByDocument(ctx, documentID)
	actionItems, _ := s.repo.GetActionItemsByDocument(ctx, documentID)
	suggestions, _ := s.repo.GetSuggestionsByDocument(ctx, documentID)

	return &FullAnalysisResult{
		Analysis:    analysis,
		Deadlines:   deadlines,
		Amounts:     amounts,
		ActionItems: actionItems,
		Suggestions: suggestions,
	}, nil
}

// ListAnalyses returns analyses for a tenant
func (s *Service) ListAnalyses(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Analysis, int, error) {
	return s.repo.ListAnalyses(ctx, tenantID, limit, offset)
}

// GetUpcomingDeadlines returns upcoming deadlines
func (s *Service) GetUpcomingDeadlines(ctx context.Context, tenantID uuid.UUID, days int) ([]*Deadline, error) {
	return s.repo.GetUpcomingDeadlines(ctx, tenantID, days)
}

// AcknowledgeDeadline acknowledges a deadline
func (s *Service) AcknowledgeDeadline(ctx context.Context, id uuid.UUID) error {
	return s.repo.AcknowledgeDeadline(ctx, id)
}

// GetPendingActionItems returns pending action items
func (s *Service) GetPendingActionItems(ctx context.Context, tenantID uuid.UUID) ([]*ActionItem, error) {
	return s.repo.GetPendingActionItems(ctx, tenantID)
}

// CompleteActionItem marks an action item as completed
func (s *Service) CompleteActionItem(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateActionItemStatus(ctx, id, ActionStatusCompleted)
}

// CancelActionItem marks an action item as cancelled
func (s *Service) CancelActionItem(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateActionItemStatus(ctx, id, ActionStatusCancelled)
}

// UseSuggestion marks a suggestion as used
func (s *Service) UseSuggestion(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkSuggestionUsed(ctx, id)
}

// GetStats returns analysis statistics
func (s *Service) GetStats(ctx context.Context, tenantID uuid.UUID) (*AnalysisStats, error) {
	return s.repo.GetAnalysisStats(ctx, tenantID)
}

// QuickClassify performs only classification without full analysis
func (s *Service) QuickClassify(ctx context.Context, text string, title string) (*ClassificationResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("AI analysis is disabled")
	}
	return s.classifier.ClassifyWithFallback(ctx, text, title)
}

// QuickSummary performs only summarization
func (s *Service) QuickSummary(ctx context.Context, text string) (*SummaryResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("AI analysis is disabled")
	}
	return s.extractor.Summarize(ctx, text)
}

// QuickExtractDeadlines extracts only deadlines
func (s *Service) QuickExtractDeadlines(ctx context.Context, text string) ([]ExtractedDeadline, error) {
	if !s.enabled {
		return nil, fmt.Errorf("AI analysis is disabled")
	}
	return s.extractor.ExtractDeadlines(ctx, text)
}

// ProcessPDFBytes analyzes a PDF from bytes (for direct API use)
func (s *Service) ProcessPDFBytes(ctx context.Context, tenantID uuid.UUID, pdfData []byte, opts AnalysisOptions) (*FullAnalysisResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("AI analysis is disabled")
	}

	// Extract text
	var text string
	if s.ocrService != nil {
		ocrResult, err := s.ocrService.ProcessBytes(ctx, pdfData)
		if err == nil {
			text = ocrResult.Text
		}
	}

	if text == "" {
		extracted, err := ocr.ExtractPDFTextFromBytes(pdfData)
		if err != nil {
			return nil, fmt.Errorf("extract text: %w", err)
		}
		text = extracted
	}

	// Create in-memory analysis result
	result := &FullAnalysisResult{
		Analysis: &Analysis{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Status:        StatusCompleted,
			ExtractedText: text,
			TextLength:    len(text),
		},
	}

	// Classification
	if opts.IncludeClassify {
		classification, err := s.classifier.ClassifyWithFallback(ctx, text, "")
		if err == nil {
			result.Analysis.DocumentType = string(classification.DocumentType)
			result.Analysis.DocumentSubtype = string(classification.DocumentSubtype)
			result.Analysis.ClassificationConfidence = classification.Confidence
		}
	}

	// Summary
	if opts.IncludeSummary {
		summary, err := s.extractor.Summarize(ctx, text)
		if err == nil {
			result.Analysis.Summary = summary.Summary
			result.Analysis.KeyPoints = summary.KeyPoints
		}
	}

	// Deadlines
	if opts.IncludeDeadlines {
		deadlines, err := s.extractor.ExtractDeadlines(ctx, text)
		if err == nil {
			for _, d := range deadlines {
				result.Deadlines = append(result.Deadlines, &Deadline{
					ID:           uuid.New(),
					TenantID:     tenantID,
					DeadlineType: d.Type,
					Date:         d.Date,
					Description:  d.Description,
					SourceText:   d.SourceText,
					Confidence:   d.Confidence,
					IsHard:       d.IsHard,
				})
			}
		}
	}

	return result, nil
}

// IsEnabled returns whether analysis is enabled
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// Helper to read content as bytes
func readContent(r io.ReadCloser) ([]byte, error) {
	defer r.Close()
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, r)
	return buf.Bytes(), err
}

// UpdateDeadline updates a deadline (T029 - manual correction)
func (s *Service) UpdateDeadline(ctx context.Context, id uuid.UUID, req *UpdateDeadlineRequest) (*Deadline, error) {
	deadline, err := s.repo.GetDeadlineByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Date != nil {
		parsedDate, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
		}
		deadline.Date = parsedDate
		deadline.CorrectedByUser = true
	}

	if req.Description != nil {
		deadline.Description = *req.Description
	}

	if req.IsAcknowledged != nil {
		deadline.IsAcknowledged = *req.IsAcknowledged
		if *req.IsAcknowledged {
			now := time.Now()
			deadline.AcknowledgedAt = &now
		}
	}

	if req.ManuallySet != nil {
		deadline.ManuallySet = *req.ManuallySet
	}

	if req.CorrectedByUser != nil {
		deadline.CorrectedByUser = *req.CorrectedByUser
	}

	if req.Notes != nil {
		deadline.Notes = *req.Notes
	}

	if err := s.repo.UpdateDeadline(ctx, deadline); err != nil {
		return nil, err
	}

	return deadline, nil
}

// UpdateActionItem updates an action item (T045)
func (s *Service) UpdateActionItem(ctx context.Context, id uuid.UUID, req *UpdateActionItemRequest) (*ActionItem, error) {
	item, err := s.repo.GetActionItemByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		item.Title = *req.Title
	}

	if req.Description != nil {
		item.Description = *req.Description
	}

	if req.Priority != nil {
		item.Priority = Priority(*req.Priority)
	}

	if req.Status != nil {
		item.Status = ActionStatus(*req.Status)
	}

	if req.DueDate != nil {
		parsedDate, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
		}
		item.DueDate = &parsedDate
	}

	if req.AssignedTo != nil {
		item.AssignedTo = req.AssignedTo
	}

	if req.Notes != nil {
		item.Notes = *req.Notes
	}

	if err := s.repo.UpdateActionItem(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

// DeleteActionItem deletes an action item (T046)
func (s *Service) DeleteActionItem(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteActionItem(ctx, id)
}

// UpdateAmount updates an extracted amount (T074 - manual correction)
func (s *Service) UpdateAmount(ctx context.Context, id uuid.UUID, req *UpdateAmountRequest) (*Amount, error) {
	amount, err := s.repo.GetAmountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Amount != nil {
		amount.Amount = *req.Amount
		amount.CorrectedByUser = true
	}

	if req.Currency != nil {
		amount.Currency = *req.Currency
	}

	if req.AmountType != nil {
		amount.AmountType = *req.AmountType
	}

	if req.Description != nil {
		amount.Description = *req.Description
	}

	if req.DueDate != nil {
		if *req.DueDate == "" {
			amount.DueDate = nil
		} else {
			parsedDate, err := time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
			}
			amount.DueDate = &parsedDate
		}
	}

	if req.Notes != nil {
		amount.Notes = *req.Notes
	}

	if err := s.repo.UpdateAmount(ctx, amount); err != nil {
		return nil, err
	}

	return amount, nil
}

// DeleteAmount deletes an extracted amount
func (s *Service) DeleteAmount(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteAmount(ctx, id)
}

// GenerateSuggestion generates a response suggestion for a document (T059)
func (s *Service) GenerateSuggestion(ctx context.Context, documentID, tenantID uuid.UUID, additionalContext, style string) (*Suggestion, error) {
	if !s.enabled {
		return nil, fmt.Errorf("AI analysis is disabled")
	}

	// Get existing analysis
	analysis, err := s.repo.GetAnalysisByDocumentID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("document analysis not found: %w", err)
	}

	// Get classification
	classification := &ClassificationResult{
		DocumentType: DocumentType(analysis.DocumentType),
		DocumentSubtype: DocumentSubtype(analysis.DocumentSubtype),
		Confidence: analysis.ClassificationConfidence,
	}

	// Generate suggestions
	suggestions, err := s.extractor.GenerateSuggestions(ctx, analysis.ExtractedText, classification)
	if err != nil {
		return nil, fmt.Errorf("generate suggestion: %w", err)
	}

	if len(suggestions) == 0 {
		return nil, fmt.Errorf("no suggestions generated")
	}

	// Store the first/best suggestion
	suggestion := &Suggestion{
		AnalysisID:     analysis.ID,
		DocumentID:     documentID,
		TenantID:       tenantID,
		SuggestionType: suggestions[0].Type,
		Title:          suggestions[0].Title,
		Content:        suggestions[0].Content,
		Reasoning:      suggestions[0].Reasoning,
		Confidence:     suggestions[0].Confidence,
	}

	if err := s.repo.CreateSuggestion(ctx, suggestion); err != nil {
		return nil, fmt.Errorf("store suggestion: %w", err)
	}

	return suggestion, nil
}

// ListResponseTemplates lists response templates (T063)
func (s *Service) ListResponseTemplates(ctx context.Context, tenantID uuid.UUID, category string) ([]*ResponseTemplate, error) {
	return s.repo.ListResponseTemplates(ctx, tenantID, category)
}

// CreateResponseTemplate creates a response template (T062)
func (s *Service) CreateResponseTemplate(ctx context.Context, tenantID uuid.UUID, req *ResponseTemplateRequest) (*ResponseTemplate, error) {
	template := &ResponseTemplate{
		TenantID:    tenantID,
		Name:        req.Name,
		Category:    req.Category,
		Content:     req.Content,
		Description: req.Description,
		Variables:   req.Variables,
		IsActive:    true,
	}

	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	if err := s.repo.CreateResponseTemplate(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// GetResponseTemplate gets a response template
func (s *Service) GetResponseTemplate(ctx context.Context, id uuid.UUID) (*ResponseTemplate, error) {
	return s.repo.GetResponseTemplateByID(ctx, id)
}

// UpdateResponseTemplate updates a response template
func (s *Service) UpdateResponseTemplate(ctx context.Context, id uuid.UUID, req *ResponseTemplateRequest) (*ResponseTemplate, error) {
	template, err := s.repo.GetResponseTemplateByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Category != "" {
		template.Category = req.Category
	}
	if req.Content != "" {
		template.Content = req.Content
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.Variables != nil {
		template.Variables = req.Variables
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateResponseTemplate(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// DeleteResponseTemplate deletes a response template
func (s *Service) DeleteResponseTemplate(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteResponseTemplate(ctx, id)
}
