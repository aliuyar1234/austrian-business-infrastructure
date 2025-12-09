package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageLog represents an AI usage log entry
type UsageLog struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	PromptType   string     `json:"prompt_type"`
	Model        string     `json:"model"`
	DocumentID   *uuid.UUID `json:"document_id,omitempty"`
	InputTokens  int        `json:"input_tokens"`
	OutputTokens int        `json:"output_tokens"`
	TotalTokens  int        `json:"total_tokens"`
	CostCents    int        `json:"cost_cents"`
	LatencyMs    int        `json:"latency_ms"`
	Success      bool       `json:"success"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// UsageStats represents aggregated usage statistics
type UsageStats struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	Period          string    `json:"period"` // day, week, month
	TotalRequests   int       `json:"total_requests"`
	SuccessfulReqs  int       `json:"successful_requests"`
	FailedRequests  int       `json:"failed_requests"`
	TotalInputTokens  int     `json:"total_input_tokens"`
	TotalOutputTokens int     `json:"total_output_tokens"`
	TotalTokens     int       `json:"total_tokens"`
	TotalCostCents  int       `json:"total_cost_cents"`
	AvgLatencyMs    int       `json:"avg_latency_ms"`
	ByPromptType    map[string]PromptTypeStats `json:"by_prompt_type"`
}

// PromptTypeStats holds stats per prompt type
type PromptTypeStats struct {
	PromptType   string `json:"prompt_type"`
	Requests     int    `json:"requests"`
	Tokens       int    `json:"tokens"`
	CostCents    int    `json:"cost_cents"`
	AvgLatencyMs int    `json:"avg_latency_ms"`
}

// UsageLogger handles AI usage logging
type UsageLogger struct {
	db *pgxpool.Pool
}

// NewUsageLogger creates a new usage logger
func NewUsageLogger(db *pgxpool.Pool) *UsageLogger {
	return &UsageLogger{db: db}
}

// LogUsage logs an AI API usage event
func (l *UsageLogger) LogUsage(ctx context.Context, log *UsageLog) error {
	query := `
		INSERT INTO ai_usage_logs (
			tenant_id, prompt_type, model, document_id,
			input_tokens, output_tokens, total_tokens,
			cost_cents, latency_ms, success, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`

	err := l.db.QueryRow(ctx, query,
		log.TenantID, log.PromptType, log.Model, log.DocumentID,
		log.InputTokens, log.OutputTokens, log.TotalTokens,
		log.CostCents, log.LatencyMs, log.Success, log.ErrorMessage,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert usage log: %w", err)
	}

	return nil
}

// LogFromResponse logs usage from a Claude API response
func (l *UsageLogger) LogFromResponse(
	ctx context.Context,
	tenantID uuid.UUID,
	promptType string,
	resp *Response,
	documentID *uuid.UUID,
	latencyMs int,
	err error,
) error {
	log := &UsageLog{
		TenantID:   tenantID,
		PromptType: promptType,
		DocumentID: documentID,
		LatencyMs:  latencyMs,
	}

	if resp != nil {
		log.Model = resp.Model
		log.InputTokens = resp.Usage.InputTokens
		log.OutputTokens = resp.Usage.OutputTokens
		log.TotalTokens = resp.Usage.InputTokens + resp.Usage.OutputTokens
		log.CostCents = EstimateCost(resp.Usage.InputTokens, resp.Usage.OutputTokens)
		log.Success = true
	}

	if err != nil {
		log.Success = false
		log.ErrorMessage = err.Error()
	}

	return l.LogUsage(ctx, log)
}

// GetUsageStats retrieves aggregated usage statistics for a tenant
func (l *UsageLogger) GetUsageStats(ctx context.Context, tenantID uuid.UUID, period string) (*UsageStats, error) {
	var interval string
	switch period {
	case "day":
		interval = "1 day"
	case "week":
		interval = "7 days"
	case "month":
		interval = "30 days"
	default:
		interval = "30 days"
		period = "month"
	}

	stats := &UsageStats{
		TenantID:     tenantID,
		Period:       period,
		ByPromptType: make(map[string]PromptTypeStats),
	}

	// Get overall stats
	overallQuery := `
		SELECT
			COUNT(*) as total_requests,
			COUNT(*) FILTER (WHERE success = true) as successful,
			COUNT(*) FILTER (WHERE success = false) as failed,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cost_cents), 0) as total_cost,
			COALESCE(AVG(latency_ms)::int, 0) as avg_latency
		FROM ai_usage_logs
		WHERE tenant_id = $1
		AND created_at >= NOW() - $2::interval
	`

	err := l.db.QueryRow(ctx, overallQuery, tenantID, interval).Scan(
		&stats.TotalRequests,
		&stats.SuccessfulReqs,
		&stats.FailedRequests,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalTokens,
		&stats.TotalCostCents,
		&stats.AvgLatencyMs,
	)
	if err != nil {
		return nil, fmt.Errorf("get overall stats: %w", err)
	}

	// Get stats by prompt type
	byTypeQuery := `
		SELECT
			prompt_type,
			COUNT(*) as requests,
			COALESCE(SUM(total_tokens), 0) as tokens,
			COALESCE(SUM(cost_cents), 0) as cost,
			COALESCE(AVG(latency_ms)::int, 0) as avg_latency
		FROM ai_usage_logs
		WHERE tenant_id = $1
		AND created_at >= NOW() - $2::interval
		GROUP BY prompt_type
		ORDER BY requests DESC
	`

	rows, err := l.db.Query(ctx, byTypeQuery, tenantID, interval)
	if err != nil {
		return nil, fmt.Errorf("get stats by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pt PromptTypeStats
		if err := rows.Scan(&pt.PromptType, &pt.Requests, &pt.Tokens, &pt.CostCents, &pt.AvgLatencyMs); err != nil {
			continue
		}
		stats.ByPromptType[pt.PromptType] = pt
	}

	return stats, nil
}

// GetRecentLogs retrieves recent usage logs for a tenant
func (l *UsageLogger) GetRecentLogs(ctx context.Context, tenantID uuid.UUID, limit int) ([]*UsageLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, tenant_id, prompt_type, model, document_id,
			input_tokens, output_tokens, total_tokens,
			cost_cents, latency_ms, success, error_message, created_at
		FROM ai_usage_logs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := l.db.Query(ctx, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent logs: %w", err)
	}
	defer rows.Close()

	var logs []*UsageLog
	for rows.Next() {
		log := &UsageLog{}
		err := rows.Scan(
			&log.ID, &log.TenantID, &log.PromptType, &log.Model, &log.DocumentID,
			&log.InputTokens, &log.OutputTokens, &log.TotalTokens,
			&log.CostCents, &log.LatencyMs, &log.Success, &log.ErrorMessage, &log.CreatedAt,
		)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetTenantCostSummary returns cost summary for billing
func (l *UsageLogger) GetTenantCostSummary(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*CostSummary, error) {
	query := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cost_cents), 0) as total_cost_cents
		FROM ai_usage_logs
		WHERE tenant_id = $1
		AND created_at >= $2
		AND created_at < $3
		AND success = true
	`

	summary := &CostSummary{
		TenantID:  tenantID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	err := l.db.QueryRow(ctx, query, tenantID, startDate, endDate).Scan(
		&summary.TotalRequests,
		&summary.TotalTokens,
		&summary.TotalCostCents,
	)
	if err != nil {
		return nil, fmt.Errorf("get cost summary: %w", err)
	}

	// Convert cents to euros
	summary.TotalCostEuros = float64(summary.TotalCostCents) / 100.0

	return summary, nil
}

// CostSummary represents a billing cost summary
type CostSummary struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	TotalRequests   int       `json:"total_requests"`
	TotalTokens     int       `json:"total_tokens"`
	TotalCostCents  int       `json:"total_cost_cents"`
	TotalCostEuros  float64   `json:"total_cost_euros"`
}

// CleanupOldLogs removes logs older than the specified retention period
func (l *UsageLogger) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays < 30 {
		retentionDays = 30 // Minimum 30 days retention
	}

	query := `
		DELETE FROM ai_usage_logs
		WHERE created_at < NOW() - $1::interval
	`

	result, err := l.db.Exec(ctx, query, fmt.Sprintf("%d days", retentionDays))
	if err != nil {
		return 0, fmt.Errorf("cleanup old logs: %w", err)
	}

	return result.RowsAffected(), nil
}
