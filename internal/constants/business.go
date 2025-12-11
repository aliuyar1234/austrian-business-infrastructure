// Package constants provides shared business constants and thresholds
// used across the application.
package constants

// Company Size Thresholds (EU KMU Definition)
// Based on the European Commission SME definition.
const (
	// SMEEmployeeThreshold is the maximum number of employees for SME classification.
	// Companies with 250 or more employees are considered large enterprises.
	SMEEmployeeThreshold = 250

	// MicroEmployeeThreshold is the maximum employees for micro-enterprises.
	// Companies with fewer than 10 employees and revenue below MicroRevenueThreshold.
	MicroEmployeeThreshold = 10

	// SmallEmployeeThreshold is the maximum employees for small enterprises.
	// Companies with fewer than 50 employees and revenue below SmallRevenueThreshold.
	SmallEmployeeThreshold = 50

	// MediumEmployeeThreshold is the maximum employees for medium enterprises.
	// Same as SMEEmployeeThreshold - companies with fewer than 250 employees.
	MediumEmployeeThreshold = SMEEmployeeThreshold
)

// Revenue Thresholds (in EUR)
const (
	// MicroRevenueThreshold is the maximum annual revenue for micro-enterprises.
	// EUR 2 million
	MicroRevenueThreshold = 2_000_000

	// SmallRevenueThreshold is the maximum annual revenue for small enterprises.
	// EUR 10 million
	SmallRevenueThreshold = 10_000_000

	// MediumRevenueThreshold is the maximum annual revenue for medium enterprises.
	// EUR 50 million - same as SMERevenueThreshold
	MediumRevenueThreshold = 50_000_000

	// SMERevenueThreshold is the maximum annual revenue for SME classification.
	// Companies with EUR 50 million or more in annual revenue may be classified
	// as large enterprises (depending on balance sheet total).
	SMERevenueThreshold = 50_000_000
)

// Balance Sheet Thresholds (in EUR)
const (
	// MicroBalanceThreshold is the maximum balance sheet total for micro-enterprises.
	// EUR 2 million
	MicroBalanceThreshold = 2_000_000

	// SmallBalanceThreshold is the maximum balance sheet total for small enterprises.
	// EUR 10 million
	SmallBalanceThreshold = 10_000_000

	// MediumBalanceThreshold is the maximum balance sheet total for medium enterprises.
	// EUR 43 million
	MediumBalanceThreshold = 43_000_000

	// SMEBalanceThreshold is the maximum balance sheet total for SME classification.
	// EUR 43 million
	SMEBalanceThreshold = 43_000_000
)
