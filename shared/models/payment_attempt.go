package models

import "time"

// PaymentAttempt is an append-only audit log of every charge call made to the
// payment gateway (mock or real). A new row is written for every attempt —
// both successful and failed — so the full payment history is always traceable.
//
// Table: payment_attempts
type PaymentAttempt struct {
	ID            uint      `gorm:"primaryKey;autoIncrement"              json:"id"`
	InvoiceID     uint      `gorm:"not null;index"                        json:"invoice_id"`
	TransactionID string    `gorm:"type:varchar(100);not null"            json:"transaction_id"`
	ChargeStatus  string    `gorm:"type:varchar(20);not null"             json:"charge_status"`  // "paid" | "failed"
	Scenario      string    `gorm:"type:varchar(20);not null"             json:"scenario"`       // "success" | "failed"
	Amount        float64   `gorm:"type:numeric(15,2);not null"           json:"amount"`
	AttemptedAt   time.Time `gorm:"not null;index;autoCreateTime"         json:"attempted_at"`

	// Invoice is populated via Preload only — not eagerly fetched.
	Invoice Invoice `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
}
