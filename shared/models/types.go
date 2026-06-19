package models

// ViolationType enumerates the categories of parking violations
// recognised by the portal. Used in both FineRuleDetail and Violation.
type ViolationType string

const (
	// Core Day-One violation types (seeded on first boot).
	ViolationExpiredMeter    ViolationType = "EXPIRED_METER"    // base: 50,000
	ViolationNoParkingZone   ViolationType = "NO_PARKING_ZONE"  // base: 150,000
	ViolationBlockingHydrant ViolationType = "BLOCKING_HYDRANT" // base: 250,000
	ViolationDisabledSpot    ViolationType = "DISABLED_SPOT"    // base: 500,000

	// Extended violation types (available for future rule versions).
	ViolationDoubleParking ViolationType = "DOUBLE_PARKING"
	ViolationTowAwayZone   ViolationType = "TOW_AWAY_ZONE"
)

// LogicType defines how the fine amount is calculated for a given rule detail.
//
//   - FLAT        : fine is always base_amount, regardless of duration.
//   - PROGRESSIVE : fine increases by multiplier × base_amount for every
//                   time_window (minutes) the vehicle remains in violation.
type LogicType string

const (
	LogicFlat        LogicType = "FLAT"
	LogicProgressive LogicType = "PROGRESSIVE"
)

// InvoiceStatus tracks the payment lifecycle of an Invoice.
type InvoiceStatus string

const (
	InvoiceStatusUnpaid  InvoiceStatus = "UNPAID"
	InvoiceStatusPaid    InvoiceStatus = "PAID"
	InvoiceStatusVoided  InvoiceStatus = "VOIDED"
)
