package unit

import (
	"testing"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/fonws"
)

// T139: Test ZM XML generation
func TestZMXMLGeneration(t *testing.T) {
	zm := &fonws.ZM{
		Year:    2025,
		Quarter: 1,
		Entries: []fonws.ZMEntry{
			{
				PartnerUID:   "DE123456789",
				CountryCode:  "DE",
				DeliveryType: fonws.ZMDeliveryTypeGoods,
				Amount:       1500000, // 15,000.00 EUR
			},
			{
				PartnerUID:   "FR12345678901",
				CountryCode:  "FR",
				DeliveryType: fonws.ZMDeliveryTypeServices,
				Amount:       500000, // 5,000.00 EUR
			},
		},
		CreatedAt: time.Date(2025, 4, 1, 10, 0, 0, 0, time.UTC),
		Status:    fonws.ZMStatusDraft,
	}

	xmlData, err := fonws.GenerateZMXML(zm)
	if err != nil {
		t.Fatalf("Failed to generate ZM XML: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify XML structure
	if !contains(xmlStr, "<?xml") {
		t.Error("Missing XML declaration")
	}
	if !contains(xmlStr, "<ZM") {
		t.Error("Missing ZM root element")
	}
	if !contains(xmlStr, "<Jahr>2025</Jahr>") {
		t.Error("Missing or incorrect Jahr")
	}
	if !contains(xmlStr, "<Quartal>1</Quartal>") {
		t.Error("Missing or incorrect Quartal")
	}
	if !contains(xmlStr, "DE123456789") {
		t.Error("Missing partner UID DE123456789")
	}
	if !contains(xmlStr, "FR12345678901") {
		t.Error("Missing partner UID FR12345678901")
	}
	if !contains(xmlStr, "<Lieferart>L</Lieferart>") {
		t.Error("Missing delivery type L (Goods)")
	}
	if !contains(xmlStr, "<Lieferart>S</Lieferart>") {
		t.Error("Missing delivery type S (Services)")
	}
}

// T140: Test ZM entry validation
func TestZMEntryValidation(t *testing.T) {
	testCases := []struct {
		name    string
		entry   fonws.ZMEntry
		valid   bool
		errType string
	}{
		{
			name: "Valid goods entry",
			entry: fonws.ZMEntry{
				PartnerUID:   "DE123456789",
				CountryCode:  "DE",
				DeliveryType: fonws.ZMDeliveryTypeGoods,
				Amount:       1000000,
			},
			valid: true,
		},
		{
			name: "Valid services entry",
			entry: fonws.ZMEntry{
				PartnerUID:   "FR12345678901",
				CountryCode:  "FR",
				DeliveryType: fonws.ZMDeliveryTypeServices,
				Amount:       500000,
			},
			valid: true,
		},
		{
			name: "Valid triangular entry",
			entry: fonws.ZMEntry{
				PartnerUID:   "IT12345678901",
				CountryCode:  "IT",
				DeliveryType: fonws.ZMDeliveryTypeTriangular,
				Amount:       250000,
			},
			valid: true,
		},
		{
			name: "Invalid - missing UID",
			entry: fonws.ZMEntry{
				PartnerUID:   "",
				CountryCode:  "DE",
				DeliveryType: fonws.ZMDeliveryTypeGoods,
				Amount:       1000000,
			},
			valid:   false,
			errType: "partner_uid",
		},
		{
			name: "Invalid - missing country",
			entry: fonws.ZMEntry{
				PartnerUID:   "DE123456789",
				CountryCode:  "",
				DeliveryType: fonws.ZMDeliveryTypeGoods,
				Amount:       1000000,
			},
			valid:   false,
			errType: "country_code",
		},
		{
			name: "Invalid - Austrian UID (not allowed in ZM)",
			entry: fonws.ZMEntry{
				PartnerUID:   "ATU12345678",
				CountryCode:  "AT",
				DeliveryType: fonws.ZMDeliveryTypeGoods,
				Amount:       1000000,
			},
			valid:   false,
			errType: "country_code",
		},
		{
			name: "Invalid - zero amount",
			entry: fonws.ZMEntry{
				PartnerUID:   "DE123456789",
				CountryCode:  "DE",
				DeliveryType: fonws.ZMDeliveryTypeGoods,
				Amount:       0,
			},
			valid:   false,
			errType: "amount",
		},
		{
			name: "Invalid - negative amount",
			entry: fonws.ZMEntry{
				PartnerUID:   "DE123456789",
				CountryCode:  "DE",
				DeliveryType: fonws.ZMDeliveryTypeGoods,
				Amount:       -1000,
			},
			valid:   false,
			errType: "amount",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.entry.Validate()
			if tc.valid && err != nil {
				t.Errorf("Expected valid entry, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Error("Expected error for invalid entry, got nil")
			}
		})
	}
}

// Test ZM struct
func TestZMStruct(t *testing.T) {
	zm := &fonws.ZM{
		Year:      2025,
		Quarter:   4,
		Entries:   []fonws.ZMEntry{},
		CreatedAt: time.Now(),
		Status:    fonws.ZMStatusDraft,
	}

	if zm.Year != 2025 {
		t.Errorf("Expected year 2025, got %d", zm.Year)
	}
	if zm.Quarter != 4 {
		t.Errorf("Expected quarter 4, got %d", zm.Quarter)
	}
	if zm.Status != fonws.ZMStatusDraft {
		t.Errorf("Expected status draft, got %s", zm.Status)
	}
}

// Test ZM status constants
func TestZMStatusConstants(t *testing.T) {
	statuses := []fonws.ZMStatus{
		fonws.ZMStatusDraft,
		fonws.ZMStatusSubmitted,
		fonws.ZMStatusAccepted,
		fonws.ZMStatusRejected,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("ZM status constant is empty")
		}
	}
}

// Test ZM delivery type constants
func TestZMDeliveryTypeConstants(t *testing.T) {
	if fonws.ZMDeliveryTypeGoods != "L" {
		t.Errorf("Expected L for goods, got %s", fonws.ZMDeliveryTypeGoods)
	}
	if fonws.ZMDeliveryTypeTriangular != "D" {
		t.Errorf("Expected D for triangular, got %s", fonws.ZMDeliveryTypeTriangular)
	}
	if fonws.ZMDeliveryTypeServices != "S" {
		t.Errorf("Expected S for services, got %s", fonws.ZMDeliveryTypeServices)
	}
}

// Test ZM period string
func TestZMPeriodString(t *testing.T) {
	zm := &fonws.ZM{
		Year:    2025,
		Quarter: 1,
	}

	period := zm.PeriodString()
	if period != "Q1/2025" {
		t.Errorf("Expected Q1/2025, got %s", period)
	}
}

// Test ZM total amount
func TestZMTotalAmount(t *testing.T) {
	zm := &fonws.ZM{
		Year:    2025,
		Quarter: 1,
		Entries: []fonws.ZMEntry{
			{Amount: 1000000},
			{Amount: 500000},
			{Amount: 250000},
		},
	}

	total := zm.TotalAmount()
	if total != 1750000 {
		t.Errorf("Expected total 1750000, got %d", total)
	}

	// Test EUR conversion
	totalEUR := zm.TotalAmountEUR()
	if totalEUR != 17500.00 {
		t.Errorf("Expected totalEUR 17500.00, got %.2f", totalEUR)
	}
}

// Test ZM validate function
func TestZMValidation(t *testing.T) {
	// Valid ZM
	validZM := &fonws.ZM{
		Year:    2025,
		Quarter: 1,
		Entries: []fonws.ZMEntry{
			{
				PartnerUID:   "DE123456789",
				CountryCode:  "DE",
				DeliveryType: fonws.ZMDeliveryTypeGoods,
				Amount:       1000000,
			},
		},
	}

	if err := validZM.Validate(); err != nil {
		t.Errorf("Expected valid ZM, got error: %v", err)
	}

	// Invalid - year out of range
	invalidYear := &fonws.ZM{
		Year:    1999,
		Quarter: 1,
	}
	if err := invalidYear.Validate(); err == nil {
		t.Error("Expected error for invalid year")
	}

	// Invalid - quarter out of range
	invalidQuarter := &fonws.ZM{
		Year:    2025,
		Quarter: 5,
	}
	if err := invalidQuarter.Validate(); err == nil {
		t.Error("Expected error for invalid quarter")
	}

	// Invalid - empty entries
	emptyEntries := &fonws.ZM{
		Year:    2025,
		Quarter: 1,
		Entries: []fonws.ZMEntry{},
	}
	if err := emptyEntries.Validate(); err == nil {
		t.Error("Expected error for empty entries")
	}
}

// Test NewZM constructor
func TestNewZM(t *testing.T) {
	zm := fonws.NewZM(2025, 1)

	if zm.Year != 2025 {
		t.Errorf("Expected year 2025, got %d", zm.Year)
	}
	if zm.Quarter != 1 {
		t.Errorf("Expected quarter 1, got %d", zm.Quarter)
	}
	if zm.Status != fonws.ZMStatusDraft {
		t.Errorf("Expected status draft, got %s", zm.Status)
	}
	if zm.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}
