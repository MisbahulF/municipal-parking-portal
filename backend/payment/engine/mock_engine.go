// Package engine provides the mock payment gateway used during development
// and testing. Replace with a real gateway client in production.
package engine

import (
	"crypto/rand"
	"fmt"
	"log"
)

// ChargeResult holds the outcome of a mock payment gateway call.
type ChargeResult struct {
	Status        string // "paid" | "failed"
	TransactionID string // "TX-" + uuid v4
}

// MockPaymentEngine simulates an external payment gateway.
// It is safe for concurrent use.
type MockPaymentEngine struct{}

// New returns a new MockPaymentEngine instance.
func New() *MockPaymentEngine {
	return &MockPaymentEngine{}
}

// Charge simulates a payment gateway call.
//
// Signature matches the assignment specification:
//
//	PaymentService.charge(invoice_id string, amount float64, scenario string)
//	    → (status string, transaction_id string)
//
// Behaviour:
//   - scenario "success" → status "paid",   transaction_id "TX-<uuid>"
//   - scenario "failed"  → status "failed", transaction_id "TX-<uuid>"
//   - any other value    → treated as "success"
//
// A unique Transaction ID is always generated regardless of outcome,
// so failed attempts are still traceable.
func (e *MockPaymentEngine) Charge(invoiceID string, amount float64, scenario string) (status string, transactionID string) {
	transactionID = "TX-" + newUUIDv4()

	log.Printf("[mock-engine] charge | invoice_id=%s amount=%.0f scenario=%s tx_id=%s",
		invoiceID, amount, scenario, transactionID)

	if scenario == "failed" {
		return "failed", transactionID
	}
	return "paid", transactionID
}

// newUUIDv4 generates a RFC 4122 UUID v4 using crypto/rand.
// Falls back to a hex-encoded random string if the OS entropy pool is unavailable.
func newUUIDv4() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// This should never happen on a healthy system.
		log.Printf("[mock-engine] uuid rand.Read error: %v — using fallback", err)
		return fmt.Sprintf("fallback-%x", b)
	}
	// Set version 4 and RFC 4122 variant bits.
	b[6] = (b[6] & 0x0f) | 0x40 // version = 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant = 10xx

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
