package models

import (
	"time"

	"gorm.io/gorm"
)

// Invoice is the immutable billing snapshot generated when a Violation is
// processed. Once created, CalcualtedAmount and AppliedFineRuleID must
// never change — they form the legal record of what was charged and under
// which rule version.
//
// Table: invoices
type Invoice struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	ViolationID uint   `gorm:"not null;uniqueIndex"     json:"violation_id"` // 1-to-1 with Violation
	InvoiceNo   string `gorm:"type:varchar(50);not null;uniqueIndex" json:"invoice_no"`   // human-readable ref, e.g. INV-2024-00001

	// AppliedFineRuleID locks the exact FineRule version used to calculate
	// this invoice. Immutable after creation.
	AppliedFineRuleID uint `gorm:"not null" json:"applied_fine_rule_id"`

	// CalculatedAmount is the final, locked fine value in the system currency.
	// Immutable after creation.
	CalculatedAmount float64 `gorm:"type:numeric(15,2);not null" json:"calculated_amount"`

	Status InvoiceStatus `gorm:"type:varchar(20);not null;default:'UNPAID'" json:"status"`

	// TransactionID is the gateway-issued TX reference populated on successful payment.
	// Nil when the invoice is UNPAID or VOIDED.
	TransactionID *string `gorm:"type:varchar(100);index" json:"transaction_id,omitempty"`

	// PaidAt records the exact moment payment was confirmed. Nil if unpaid.
	PaidAt *time.Time `gorm:"index" json:"paid_at,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime"   json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"   json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"            json:"-"`

	// Associations (read-only, populated via Preload)
	Violation       Violation `gorm:"foreignKey:ViolationID"       json:"violation,omitempty"`
	AppliedFineRule FineRule  `gorm:"foreignKey:AppliedFineRuleID" json:"applied_fine_rule,omitempty"`
}
