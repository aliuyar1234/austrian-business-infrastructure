package integration

import (
	"strings"
	"testing"

	imports "github.com/austrian-business-infrastructure/fo/internal/import"
)

// T065: Integration tests for CSV import

func TestCSVParser(t *testing.T) {
	t.Run("Parse valid CSV", func(t *testing.T) {
		csv := `name,type,tid,ben_id,pin
Account 1,finanzonline,123456782,USER1,pin1
Account 2,finanzonline,234567890,USER2,pin2`

		parser := imports.NewParser(500)
		result, err := parser.Parse(strings.NewReader(csv))
		if err != nil {
			t.Fatalf("Failed to parse CSV: %v", err)
		}

		if result.TotalRows != 2 {
			t.Errorf("Expected 2 rows, got %d", result.TotalRows)
		}

		// First row should be valid (123456782 has valid checksum)
		if len(result.Rows) < 1 {
			t.Fatal("No rows parsed")
		}

		row1 := result.Rows[0]
		if row1.Name != "Account 1" {
			t.Errorf("Name mismatch: got %s", row1.Name)
		}
		if row1.Type != "finanzonline" {
			t.Errorf("Type mismatch: got %s", row1.Type)
		}
		if row1.TID != "123456782" {
			t.Errorf("TID mismatch: got %s", row1.TID)
		}
	})

	t.Run("Parse CSV with validation errors", func(t *testing.T) {
		csv := `name,type,tid,ben_id,pin
,finanzonline,123456789,USER1,pin1
Account 2,invalid_type,234567890,USER2,pin2
Account 3,finanzonline,12345,USER3,pin3`

		parser := imports.NewParser(500)
		result, err := parser.Parse(strings.NewReader(csv))
		if err != nil {
			t.Fatalf("Failed to parse CSV: %v", err)
		}

		if result.TotalRows != 3 {
			t.Errorf("Expected 3 rows, got %d", result.TotalRows)
		}

		// All rows should have errors
		for i, row := range result.Rows {
			if row.Valid {
				t.Errorf("Row %d should have validation errors", i+1)
			}
			if len(row.Errors) == 0 {
				t.Errorf("Row %d should have error messages", i+1)
			}
		}
	})

	t.Run("Parse empty CSV", func(t *testing.T) {
		csv := ``

		parser := imports.NewParser(500)
		_, err := parser.Parse(strings.NewReader(csv))
		if err != imports.ErrEmptyFile {
			t.Errorf("Expected ErrEmptyFile, got %v", err)
		}
	})

	t.Run("Parse CSV missing headers", func(t *testing.T) {
		csv := `name,type
Account 1,finanzonline`

		parser := imports.NewParser(500)
		_, err := parser.Parse(strings.NewReader(csv))
		if err != imports.ErrMissingHeaders {
			t.Errorf("Expected ErrMissingHeaders, got %v", err)
		}
	})

	t.Run("Parse CSV exceeding max rows", func(t *testing.T) {
		// Create CSV with 3 rows but max 2
		csv := `name,type,tid,ben_id,pin
Account 1,finanzonline,123456782,USER1,pin1
Account 2,finanzonline,234567890,USER2,pin2
Account 3,finanzonline,345678901,USER3,pin3`

		parser := imports.NewParser(2)
		_, err := parser.Parse(strings.NewReader(csv))
		if err != imports.ErrTooManyRows {
			t.Errorf("Expected ErrTooManyRows, got %v", err)
		}
	})

	t.Run("Parse CSV with ELDA accounts", func(t *testing.T) {
		csv := `name,type,tid,ben_id,pin,dienstgeber_nr,cert_path
ELDA Account,elda,,,mypin,123456,/path/to/cert.p12`

		parser := imports.NewParser(500)
		result, err := parser.Parse(strings.NewReader(csv))
		if err != nil {
			t.Fatalf("Failed to parse CSV: %v", err)
		}

		if result.TotalRows != 1 {
			t.Errorf("Expected 1 row, got %d", result.TotalRows)
		}

		row := result.Rows[0]
		if row.Type != "elda" {
			t.Errorf("Type mismatch: got %s", row.Type)
		}
		if row.DienstgeberNr != "123456" {
			t.Errorf("DienstgeberNr mismatch: got %s", row.DienstgeberNr)
		}
	})

	t.Run("Parse CSV with Firmenbuch accounts", func(t *testing.T) {
		csv := `name,type,tid,ben_id,pin,username,password
FB Account,firmenbuch,,,,fbuser,fbpass`

		parser := imports.NewParser(500)
		result, err := parser.Parse(strings.NewReader(csv))
		if err != nil {
			t.Fatalf("Failed to parse CSV: %v", err)
		}

		if result.TotalRows != 1 {
			t.Errorf("Expected 1 row, got %d", result.TotalRows)
		}

		row := result.Rows[0]
		if row.Type != "firmenbuch" {
			t.Errorf("Type mismatch: got %s", row.Type)
		}
		if row.Username != "fbuser" {
			t.Errorf("Username mismatch: got %s", row.Username)
		}
	})

	t.Run("CSV header case insensitivity", func(t *testing.T) {
		csv := `NAME,TYPE,TID,BEN_ID,PIN
Account 1,finanzonline,123456782,USER1,pin1`

		parser := imports.NewParser(500)
		result, err := parser.Parse(strings.NewReader(csv))
		if err != nil {
			t.Fatalf("Failed to parse CSV: %v", err)
		}

		if result.TotalRows != 1 {
			t.Errorf("Expected 1 row, got %d", result.TotalRows)
		}

		row := result.Rows[0]
		if row.Name != "Account 1" {
			t.Errorf("Name not parsed correctly: got %s", row.Name)
		}
	})

	t.Run("CSV with extra whitespace", func(t *testing.T) {
		csv := `name,type,tid,ben_id,pin
  Account 1  ,  finanzonline  ,  123456782  ,  USER1  ,  pin1  `

		parser := imports.NewParser(500)
		result, err := parser.Parse(strings.NewReader(csv))
		if err != nil {
			t.Fatalf("Failed to parse CSV: %v", err)
		}

		row := result.Rows[0]
		if row.Name != "Account 1" {
			t.Errorf("Whitespace not trimmed from name: got '%s'", row.Name)
		}
		if row.TID != "123456782" {
			t.Errorf("Whitespace not trimmed from TID: got '%s'", row.TID)
		}
	})
}

func TestImportJob(t *testing.T) {
	t.Run("ImportJob status values", func(t *testing.T) {
		validStatuses := []string{"pending", "processing", "completed", "failed"}
		for _, status := range validStatuses {
			t.Logf("Valid status: %s", status)
		}
	})

	t.Run("ImportError structure", func(t *testing.T) {
		err := imports.ImportError{
			RowNumber: 5,
			Message:   "Invalid TID format",
		}

		if err.RowNumber != 5 {
			t.Errorf("RowNumber mismatch")
		}
		if err.Message != "Invalid TID format" {
			t.Errorf("Message mismatch")
		}
	})
}

func TestParseResult(t *testing.T) {
	t.Run("ParseResult counts", func(t *testing.T) {
		csv := `name,type,tid,ben_id,pin
Account 1,finanzonline,123456782,USER1,pin1
,finanzonline,234567890,USER2,pin2`

		parser := imports.NewParser(500)
		result, err := parser.Parse(strings.NewReader(csv))
		if err != nil {
			t.Fatalf("Failed to parse CSV: %v", err)
		}

		// First row valid, second row missing name
		if result.ValidCount+result.ErrorCount != result.TotalRows {
			t.Errorf("Count mismatch: valid=%d, error=%d, total=%d",
				result.ValidCount, result.ErrorCount, result.TotalRows)
		}
	})
}
