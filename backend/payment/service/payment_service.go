package service

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/dhill/parking-violation-portal/backend/payment/engine"
	"github.com/dhill/parking-violation-portal/backend/payment/repository"
	"github.com/dhill/parking-violation-portal/shared/models"
)

// Sentinel errors — callers use errors.Is() to map these to HTTP status codes.
var (
	// ErrInvoiceNotFound is returned when the invoice ID does not exist.
	ErrInvoiceNotFound = errors.New("invoice not found")

	// ErrInvoiceAlreadyPaid is returned when the invoice is already in PAID status.
	// Callers should respond with 409 Conflict.
	ErrInvoiceAlreadyPaid = errors.New("invoice has already been paid")

	// ErrInvoiceVoided is returned when attempting to pay a voided invoice.
	// Callers should respond with 422 Unprocessable Entity.
	ErrInvoiceVoided = errors.New("invoice has been voided and cannot be paid")

	// ErrPaymentFailed is returned when the mock engine returns "failed".
	// Callers should respond with 402 Payment Required.
	ErrPaymentFailed = errors.New("payment gateway declined the transaction")
)

// PaymentResult bundles the updated invoice with the mock engine's response
// so the handler can return the full picture in a single JSON body.
type PaymentResult struct {
	Invoice       *models.Invoice
	TransactionID string
	ChargeStatus  string
}

// PaymentService defines the business logic interface for Flow 4.
type PaymentService interface {
	// ProcessPayment validates, runs the mock engine, and (on success) marks
	// the invoice as PAID. scenario must be "success" or "failed".
	ProcessPayment(invoiceID uint, scenario string) (*PaymentResult, error)

	// PayInvoice is the public member-facing endpoint (POST /payments/:id/pay).
	// It verifies the invoice amount, calls the mock engine, logs every attempt
	// to payment_attempts, and on success persists the transaction_id.
	PayInvoice(invoiceID uint, scenario string) (*PaymentResult, error)

	// GetInvoiceStatus returns the current state of an invoice.
	GetInvoiceStatus(invoiceID uint) (*models.Invoice, error)

	// GetInvoicesByPlate lists all invoices for a vehicle license plate.
	GetInvoicesByPlate(licensePlate string) ([]models.Invoice, error)
}

type paymentService struct {
	repo   repository.PaymentRepository
	engine *engine.MockPaymentEngine
}

// New returns a PaymentService wired to the given repository and mock engine.
func New(repo repository.PaymentRepository) PaymentService {
	return &paymentService{
		repo:   repo,
		engine: engine.New(),
	}
}

// ProcessPayment implements Flow 4: the full payment pipeline.
//
// Steps:
//  1. Fetch the invoice — return ErrInvoiceNotFound if missing.
//  2. Guard: reject if already PAID (409) or VOIDED (422).
//  3. Call mock engine: Charge(invoice_id, amount, scenario).
//  4a. Engine returns "failed" → return ErrPaymentFailed (invoice stays UNPAID).
//  4b. Engine returns "paid"   → atomically update: status=PAID, paid_at=now.
//  5. Return PaymentResult with invoice snapshot + engine transaction data.
func (s *paymentService) ProcessPayment(invoiceID uint, scenario string) (*PaymentResult, error) {
	// ── Step 1: Fetch Invoice ─────────────────────────────────────────────────
	invoice, err := s.repo.FindInvoiceByID(invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	// ── Step 2: Status Guard ──────────────────────────────────────────────────
	switch invoice.Status {
	case models.InvoiceStatusPaid:
		return nil, fmt.Errorf("%w (invoice_no: %s, paid_at: %v)",
			ErrInvoiceAlreadyPaid, invoice.InvoiceNo, invoice.PaidAt)
	case models.InvoiceStatusVoided:
		return nil, fmt.Errorf("%w (invoice_no: %s)",
			ErrInvoiceVoided, invoice.InvoiceNo)
	}

	// ── Step 3: Mock Payment Engine ───────────────────────────────────────────
	// Matches the assignment signature exactly:
	//   Charge(invoice_id string, amount float64, scenario string)
	//       → (status string, transaction_id string)
	chargeStatus, transactionID := s.engine.Charge(
		strconv.FormatUint(uint64(invoiceID), 10),
		invoice.CalculatedAmount,
		scenario,
	)

	// ── Step 4a: Engine declined ──────────────────────────────────────────────
	if chargeStatus == "failed" {
		log.Printf("[payment-service] charge declined | invoice_id=%d tx_id=%s",
			invoiceID, transactionID)
		return &PaymentResult{
			Invoice:       invoice, // still UNPAID
			TransactionID: transactionID,
			ChargeStatus:  chargeStatus,
		}, ErrPaymentFailed
	}

	// ── Step 4b: Engine approved → persist ───────────────────────────────────
	paidAt := time.Now().UTC()
	updated, err := s.repo.MarkAsPaid(invoiceID, paidAt)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize payment: %w", err)
	}

	log.Printf("[payment-service] payment confirmed | id=%d no=%s amount=%.0f tx_id=%s plate=%s",
		updated.ID,
		updated.InvoiceNo,
		updated.CalculatedAmount,
		transactionID,
		updated.Violation.LicensePlate,
	)

	return &PaymentResult{
		Invoice:       updated,
		TransactionID: transactionID,
		ChargeStatus:  chargeStatus,
	}, nil
}

// PayInvoice is the public member endpoint (POST /api/v1/payments/:invoice_id/pay).
//
// Steps:
//  1. Fetch invoice — verify it exists and is UNPAID.
//  2. Call mock engine: Charge(invoice_id, amount, scenario).
//  3. Log the attempt to payment_attempts (always, regardless of outcome).
//  4a. Engine "paid"   → persist status=PAID, paid_at, transaction_id.
//  4b. Engine "failed" → invoice stays UNPAID, return ErrPaymentFailed.
//  5. Return PaymentResult.
func (s *paymentService) PayInvoice(invoiceID uint, scenario string) (*PaymentResult, error) {
	// Step 1: Fetch and validate invoice
	invoice, err := s.repo.FindInvoiceByID(invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}
	switch invoice.Status {
	case models.InvoiceStatusPaid:
		return nil, fmt.Errorf("%w (invoice_no: %s)", ErrInvoiceAlreadyPaid, invoice.InvoiceNo)
	case models.InvoiceStatusVoided:
		return nil, fmt.Errorf("%w (invoice_no: %s)", ErrInvoiceVoided, invoice.InvoiceNo)
	}

	// Step 2: Call mock engine
	chargeStatus, transactionID := s.engine.Charge(
		strconv.FormatUint(uint64(invoiceID), 10),
		invoice.CalculatedAmount,
		scenario,
	)

	// Step 3: Log every attempt (fire-and-forget — do not block on log error)
	attempt := &models.PaymentAttempt{
		InvoiceID:     invoiceID,
		TransactionID: transactionID,
		ChargeStatus:  chargeStatus,
		Scenario:      scenario,
		Amount:        invoice.CalculatedAmount,
	}
	if logErr := s.repo.LogPaymentAttempt(attempt); logErr != nil {
		log.Printf("[payment-service] WARNING: failed to log attempt for invoice %d: %v", invoiceID, logErr)
	}

	// Step 4a: Engine declined
	if chargeStatus == "failed" {
		log.Printf("[payment-service] charge declined | invoice_id=%d tx_id=%s", invoiceID, transactionID)
		return &PaymentResult{
			Invoice:       invoice,
			TransactionID: transactionID,
			ChargeStatus:  chargeStatus,
		}, ErrPaymentFailed
	}

	// Step 4b: Persist payment — store transaction_id in invoices table
	paidAt := time.Now().UTC()
	updated, err := s.repo.MarkAsPaidWithTx(invoiceID, paidAt, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to finalise payment: %w", err)
	}

	log.Printf("[payment-service] payment success | id=%d no=%s amount=%.0f tx_id=%s plate=%s",
		updated.ID, updated.InvoiceNo, updated.CalculatedAmount, transactionID,
		updated.Violation.LicensePlate,
	)

	return &PaymentResult{
		Invoice:       updated,
		TransactionID: transactionID,
		ChargeStatus:  chargeStatus,
	}, nil
}

// GetInvoiceStatus returns the current state of an invoice with preloaded relations.
func (s *paymentService) GetInvoiceStatus(invoiceID uint) (*models.Invoice, error) {
	inv, err := s.repo.FindInvoiceByID(invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}
	return inv, nil
}

// GetInvoicesByPlate returns all invoices for a given license plate, newest first.
func (s *paymentService) GetInvoicesByPlate(licensePlate string) ([]models.Invoice, error) {
	invoices, err := s.repo.FindInvoicesByPlate(licensePlate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invoices for plate %s: %w", licensePlate, err)
	}
	return invoices, nil
}
