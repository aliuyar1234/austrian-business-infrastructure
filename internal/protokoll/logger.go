package protokoll

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Logger provides protocol logging for ELDA operations
type Logger struct {
	repo *Repository
}

// NewLogger creates a new protokoll logger
func NewLogger(db *pgxpool.Pool) *Logger {
	return &Logger{repo: NewRepository(db)}
}

// LogRequest starts logging an ELDA request
func (l *Logger) LogRequest(ctx context.Context, req *LogRequestParams) *LogEntry {
	return &LogEntry{
		logger:     l,
		ctx:        ctx,
		startTime:  time.Now(),
		protokoll:  &Protokoll{
			ID:            uuid.New(),
			ELDAAccountID: req.ELDAAccountID,
			Type:          req.Type,
			Status:        ProtokollStatusPending,
			RelatedID:     req.RelatedID,
			SVNummer:      req.SVNummer,
			Description:   req.Description,
			RequestXML:    req.RequestXML,
			CreatedBy:     req.CreatedBy,
		},
	}
}

// LogRequestParams contains parameters for starting a log entry
type LogRequestParams struct {
	ELDAAccountID uuid.UUID
	Type          ProtokollType
	RelatedID     *uuid.UUID
	SVNummer      string
	Description   string
	RequestXML    string
	CreatedBy     *uuid.UUID
}

// LogEntry represents an in-progress log entry
type LogEntry struct {
	logger    *Logger
	ctx       context.Context
	startTime time.Time
	protokoll *Protokoll
}

// Success marks the request as successful and saves
func (e *LogEntry) Success(protokollnummer string, responseXML string) error {
	e.protokoll.Status = ProtokollStatusSuccess
	e.protokoll.Protokollnummer = protokollnummer
	e.protokoll.ResponseXML = responseXML
	e.protokoll.DurationMS = time.Since(e.startTime).Milliseconds()
	e.protokoll.CreatedAt = time.Now()

	return e.logger.repo.Create(e.ctx, e.protokoll)
}

// Error marks the request as failed and saves
func (e *LogEntry) Error(errorCode string, errorMessage string, responseXML string) error {
	e.protokoll.Status = ProtokollStatusError
	e.protokoll.ErrorCode = errorCode
	e.protokoll.ErrorMessage = errorMessage
	e.protokoll.ResponseXML = responseXML
	e.protokoll.DurationMS = time.Since(e.startTime).Milliseconds()
	e.protokoll.CreatedAt = time.Now()

	return e.logger.repo.Create(e.ctx, e.protokoll)
}

// LogSimple logs a simple operation (non-request based)
func (l *Logger) LogSimple(ctx context.Context, params *LogSimpleParams) error {
	p := &Protokoll{
		ID:            uuid.New(),
		ELDAAccountID: params.ELDAAccountID,
		Type:          params.Type,
		Status:        params.Status,
		Protokollnummer: params.Protokollnummer,
		RelatedID:     params.RelatedID,
		SVNummer:      params.SVNummer,
		Description:   params.Description,
		ErrorCode:     params.ErrorCode,
		ErrorMessage:  params.ErrorMessage,
		DurationMS:    params.DurationMS,
		CreatedAt:     time.Now(),
		CreatedBy:     params.CreatedBy,
	}

	return l.repo.Create(ctx, p)
}

// LogSimpleParams contains parameters for a simple log entry
type LogSimpleParams struct {
	ELDAAccountID   uuid.UUID
	Type            ProtokollType
	Status          ProtokollStatus
	Protokollnummer string
	RelatedID       *uuid.UUID
	SVNummer        string
	Description     string
	ErrorCode       string
	ErrorMessage    string
	DurationMS      int64
	CreatedBy       *uuid.UUID
}

// LogAnmeldung logs an Anmeldung operation
func (l *Logger) LogAnmeldung(ctx context.Context, accountID uuid.UUID, meldungID uuid.UUID, svNummer string, success bool, protokollnummer string, errorCode string, errorMessage string, requestXML string, responseXML string, durationMS int64) error {
	status := ProtokollStatusSuccess
	if !success {
		status = ProtokollStatusError
	}

	return l.LogSimple(ctx, &LogSimpleParams{
		ELDAAccountID:   accountID,
		Type:            ProtokollTypeAnmeldung,
		Status:          status,
		Protokollnummer: protokollnummer,
		RelatedID:       &meldungID,
		SVNummer:        svNummer,
		Description:     "Anmeldung übermittelt",
		ErrorCode:       errorCode,
		ErrorMessage:    errorMessage,
		DurationMS:      durationMS,
	})
}

// LogAbmeldung logs an Abmeldung operation
func (l *Logger) LogAbmeldung(ctx context.Context, accountID uuid.UUID, meldungID uuid.UUID, svNummer string, success bool, protokollnummer string, errorCode string, errorMessage string, durationMS int64) error {
	status := ProtokollStatusSuccess
	if !success {
		status = ProtokollStatusError
	}

	return l.LogSimple(ctx, &LogSimpleParams{
		ELDAAccountID:   accountID,
		Type:            ProtokollTypeAbmeldung,
		Status:          status,
		Protokollnummer: protokollnummer,
		RelatedID:       &meldungID,
		SVNummer:        svNummer,
		Description:     "Abmeldung übermittelt",
		ErrorCode:       errorCode,
		ErrorMessage:    errorMessage,
		DurationMS:      durationMS,
	})
}

// LogMBGM logs an mBGM submission
func (l *Logger) LogMBGM(ctx context.Context, accountID uuid.UUID, mbgmID uuid.UUID, success bool, protokollnummer string, errorCode string, errorMessage string, durationMS int64) error {
	status := ProtokollStatusSuccess
	if !success {
		status = ProtokollStatusError
	}

	return l.LogSimple(ctx, &LogSimpleParams{
		ELDAAccountID:   accountID,
		Type:            ProtokollTypeMBGM,
		Status:          status,
		Protokollnummer: protokollnummer,
		RelatedID:       &mbgmID,
		Description:     "mBGM übermittelt",
		ErrorCode:       errorCode,
		ErrorMessage:    errorMessage,
		DurationMS:      durationMS,
	})
}

// LogL16 logs an L16 submission
func (l *Logger) LogL16(ctx context.Context, accountID uuid.UUID, l16ID uuid.UUID, svNummer string, success bool, protokollnummer string, errorCode string, errorMessage string, durationMS int64) error {
	status := ProtokollStatusSuccess
	if !success {
		status = ProtokollStatusError
	}

	return l.LogSimple(ctx, &LogSimpleParams{
		ELDAAccountID:   accountID,
		Type:            ProtokollTypeL16,
		Status:          status,
		Protokollnummer: protokollnummer,
		RelatedID:       &l16ID,
		SVNummer:        svNummer,
		Description:     "L16 übermittelt",
		ErrorCode:       errorCode,
		ErrorMessage:    errorMessage,
		DurationMS:      durationMS,
	})
}

// LogDataboxSync logs a databox sync operation
func (l *Logger) LogDataboxSync(ctx context.Context, accountID uuid.UUID, success bool, documentsFound int, errorMessage string, durationMS int64) error {
	status := ProtokollStatusSuccess
	description := "Databox-Synchronisierung erfolgreich"
	if !success {
		status = ProtokollStatusError
		description = "Databox-Synchronisierung fehlgeschlagen"
	}
	if documentsFound > 0 {
		description = description + " - " + string(rune(documentsFound)) + " Dokumente"
	}

	return l.LogSimple(ctx, &LogSimpleParams{
		ELDAAccountID:  accountID,
		Type:           ProtokollTypeDataboxSync,
		Status:         status,
		Description:    description,
		ErrorMessage:   errorMessage,
		DurationMS:     durationMS,
	})
}
