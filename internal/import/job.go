package imports

import (
	"context"
	"sync"

	"austrian-business-infrastructure/internal/account"
	"austrian-business-infrastructure/internal/account/types"
	"github.com/google/uuid"
)

// JobRunner executes import jobs
type JobRunner struct {
	repo           *Repository
	accountService *account.Service
	concurrency    int
}

// NewJobRunner creates a new job runner
func NewJobRunner(repo *Repository, accountService *account.Service, concurrency int) *JobRunner {
	if concurrency <= 0 {
		concurrency = 5 // Default max concurrent
	}
	return &JobRunner{
		repo:           repo,
		accountService: accountService,
		concurrency:    concurrency,
	}
}

// Run executes an import job with the parsed rows
func (jr *JobRunner) Run(ctx context.Context, job *ImportJob, rows []*ParsedRow) error {
	// Update status to processing
	if err := jr.repo.UpdateStatus(ctx, job.ID, "processing"); err != nil {
		return err
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, jr.concurrency)

	processed := 0
	successCount := 0
	var importErrors []ImportError

	for _, row := range rows {
		if !row.Valid {
			mu.Lock()
			processed++
			importErrors = append(importErrors, ImportError{
				RowNumber: row.RowNumber,
				Message:   joinErrors(row.Errors),
			})
			jr.repo.UpdateProgress(ctx, job.ID, processed, successCount, len(importErrors))
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(r *ParsedRow) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Create account
			err := jr.createAccount(ctx, job.TenantID, r)

			mu.Lock()
			processed++
			if err != nil {
				importErrors = append(importErrors, ImportError{
					RowNumber: r.RowNumber,
					Message:   err.Error(),
				})
			} else {
				successCount++
			}
			// Update progress periodically
			if processed%10 == 0 || processed == len(rows) {
				jr.repo.UpdateProgress(ctx, job.ID, processed, successCount, len(importErrors))
			}
			mu.Unlock()
		}(row)
	}

	wg.Wait()

	// Final progress update
	jr.repo.UpdateProgress(ctx, job.ID, processed, successCount, len(importErrors))

	// Complete the job
	return jr.repo.Complete(ctx, job.ID, importErrors)
}

func (jr *JobRunner) createAccount(ctx context.Context, tenantID uuid.UUID, row *ParsedRow) error {
	var creds interface{}

	switch row.Type {
	case account.AccountTypeFinanzOnline:
		creds = &types.FinanzOnlineCredentials{
			TID:   row.TID,
			BenID: row.BenID,
			PIN:   row.PIN,
		}

	case account.AccountTypeELDA:
		creds = &types.ELDACredentials{
			DienstgeberNr:       row.DienstgeberNr,
			PIN:                 row.PIN,
			CertificatePath:     row.CertPath,
			CertificatePassword: row.CertPassword,
		}

	case account.AccountTypeFirmenbuch:
		creds = &types.FirmenbuchCredentials{
			Username: row.Username,
			Password: row.Password,
		}
	}

	input := &account.CreateAccountInput{
		TenantID:    tenantID,
		Name:        row.Name,
		Type:        row.Type,
		Credentials: creds,
	}

	_, err := jr.accountService.CreateAccount(ctx, input)
	return err
}

func joinErrors(errors []string) string {
	if len(errors) == 0 {
		return ""
	}
	result := errors[0]
	for i := 1; i < len(errors); i++ {
		result += "; " + errors[i]
	}
	return result
}
