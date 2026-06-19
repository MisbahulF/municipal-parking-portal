package service

import (
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/dhill/parking-violation-portal/backend/billing/repository"
	"github.com/dhill/parking-violation-portal/shared/models"
)

// GenerateInvoiceInput is the DTO received from the violation service trigger.
type GenerateInvoiceInput struct {
	ViolationID  uint   `json:"violation_id"`
	LicensePlate string `json:"license_plate"`
}

// FineRuleDetailInput is a single violation-type rule supplied in a publish request.
type FineRuleDetailInput struct {
	ViolationType models.ViolationType `json:"violation_type"`
	BaseAmount    float64              `json:"base_amount"`
	LogicType     models.LogicType     `json:"logic_type"`
	TimeWindow    int                  `json:"time_window"`
	Multiplier    float64              `json:"multiplier"`
}

// PublishFineRuleInput carries the new ruleset submitted by an officer.
type PublishFineRuleInput struct {
	Details []FineRuleDetailInput `json:"details"`
}

// Sentinel errors
var (
	// ErrInvoiceAlreadyExists is returned when the violation already has an invoice.
	// Callers should treat this as a 409 Conflict, not a 500.
	ErrInvoiceAlreadyExists = errors.New("invoice already exists for this violation")

	// ErrNoFineRuleDetails is returned when a publish request contains no details.
	ErrNoFineRuleDetails = errors.New("at least one fine rule detail is required")

	// ErrDuplicateViolationType is returned when the same violation type appears
	// more than once in a single publish request.
	ErrDuplicateViolationType = errors.New("duplicate violation_type in rule details")
)

// BillingService defines the calculation engine interface.
type BillingService interface {
	GenerateInvoice(input GenerateInvoiceInput) (*models.Invoice, error)
	GetInvoiceByID(id uint) (*models.Invoice, error)
	// PublishFineRule implements Flow 3: creates a new versioned FineRule,
	// deactivates the previous one, and never mutates existing invoices.
	PublishFineRule(input PublishFineRuleInput) (*models.FineRule, error)
	// GetActiveFineRule returns the currently active ruleset config.
	GetActiveFineRule() (*models.FineRule, error)
	// GetAllInvoices returns all invoices.
	GetAllInvoices() ([]models.Invoice, error)
}

type billingService struct {
	repo repository.BillingRepository
}

// New returns a BillingService wired to the given repository.
func New(repo repository.BillingRepository) BillingService {
	return &billingService{repo: repo}
}

// GetActiveFineRule fetches the currently active version of rules from DB.
func (s *billingService) GetActiveFineRule() (*models.FineRule, error) {
	return s.repo.FindActiveFineRule()
}

// GetAllInvoices fetches all invoices.
func (s *billingService) GetAllInvoices() ([]models.Invoice, error) {
	return s.repo.FindAllInvoices()
}

// ─── Public Methods ───────────────────────────────────────────────────────────

// GenerateInvoice implements Flow 2 & 5: the full fine calculation pipeline.
//
// Steps:
//  1. Idempotency guard (one invoice per violation).
//  2. Fetch the violation record (source of truth for type & timestamp).
//  3. Fetch the currently active FineRule version.
//  4. Look up base_amount for the violation's type.
//  5. Compute time_multiplier (day/night window).
//  6. Count prior UNPAID violations within 90 days → repeat_multiplier.
//  7. Calculate: fine = base_amount × time_multiplier × repeat_multiplier.
//  8. Persist and return the immutable Invoice snapshot.
func (s *billingService) GenerateInvoice(input GenerateInvoiceInput) (*models.Invoice, error) {
	// ── Step 1: Idempotency ──────────────────────────────────────────────────
	exists, err := s.repo.InvoiceExistsByViolationID(input.ViolationID)
	if err != nil {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		return nil, ErrInvoiceAlreadyExists
	}

	// ── Step 2: Fetch Violation ──────────────────────────────────────────────
	violation, err := s.repo.FindViolationByID(input.ViolationID)
	if err != nil {
		return nil, fmt.Errorf("violation %d not found: %w", input.ViolationID, err)
	}

	// ── Step 3: Fetch Active FineRule ────────────────────────────────────────
	activeRule, err := s.repo.FindActiveFineRule()
	if err != nil {
		return nil, fmt.Errorf("no active FineRule found: %w", err)
	}

	// ── Step 4: Base Amount ──────────────────────────────────────────────────
	detail, err := s.repo.FindDetailByViolationType(activeRule.ID, violation.ViolationType)
	if err != nil {
		return nil, fmt.Errorf("no fine detail for violation type %q in rule v%d: %w",
			violation.ViolationType, activeRule.Version, err)
	}

	// ── Step 5: Time Multiplier ──────────────────────────────────────────────
	timeMult := timeMultiplier(violation.Timestamp)

	// ── Step 6: Repeat Multiplier (90-day unpaid window) ────────────────────
	priorUnpaid, err := s.repo.CountPriorUnpaid(
		violation.LicensePlate,
		violation.Timestamp,
		violation.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("repeat-offender query failed: %w", err)
	}
	repeatMult := repeatMultiplier(priorUnpaid)

	// ── Step 7: Final Calculation ────────────────────────────────────────────
	// Round to 2 decimal places to avoid floating-point noise in storage.
	calculatedAmount := math.Round(detail.BaseAmount*timeMult*repeatMult*100) / 100

	log.Printf(
		"[billing-service] fine calc | violation=%d type=%s base=%.0f time=%.1fx repeat=%.1fx (prior_unpaid=%d) → %.0f",
		violation.ID, violation.ViolationType,
		detail.BaseAmount, timeMult, repeatMult, priorUnpaid,
		calculatedAmount,
	)

	// ── Step 8: Persist Invoice ──────────────────────────────────────────────
	invoice := &models.Invoice{
		ViolationID:       violation.ID,
		InvoiceNo:         generateInvoiceNo(violation.ID, violation.Timestamp),
		AppliedFineRuleID: activeRule.ID,
		CalculatedAmount:  calculatedAmount,
		Status:            models.InvoiceStatusUnpaid,
	}
	if err := s.repo.CreateInvoice(invoice); err != nil {
		return nil, fmt.Errorf("failed to persist invoice: %w", err)
	}

	log.Printf("[billing-service] invoice created | id=%d no=%s amount=%.0f status=%s",
		invoice.ID, invoice.InvoiceNo, invoice.CalculatedAmount, invoice.Status)

	return invoice, nil
}

// GetInvoiceByID returns an invoice with its Violation and AppliedFineRule preloaded.
func (s *billingService) GetInvoiceByID(id uint) (*models.Invoice, error) {
	inv, err := s.repo.FindInvoiceByID(id)
	if err != nil {
		return nil, fmt.Errorf("invoice %d not found: %w", id, err)
	}
	return inv, nil
}

// PublishFineRule implements Flow 3: version-locked rule update.
//
// Guarantees:
//   - At least one detail must be provided.
//   - No duplicate violation_type within the same submission.
//   - The previous active rule is atomically deactivated in the same transaction.
//   - Zero existing invoices are modified — they remain linked to their original rule version.
func (s *billingService) PublishFineRule(input PublishFineRuleInput) (*models.FineRule, error) {
	// Validate: non-empty
	if len(input.Details) == 0 {
		return nil, ErrNoFineRuleDetails
	}

	// Validate: no duplicate violation types
	seen := make(map[models.ViolationType]struct{}, len(input.Details))
	for _, d := range input.Details {
		if _, exists := seen[d.ViolationType]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateViolationType, d.ViolationType)
		}
		seen[d.ViolationType] = struct{}{}
	}

	// Map input DTOs → model structs
	details := make([]models.FineRuleDetail, len(input.Details))
	for i, d := range input.Details {
		details[i] = models.FineRuleDetail{
			ViolationType: d.ViolationType,
			BaseAmount:    d.BaseAmount,
			LogicType:     d.LogicType,
			TimeWindow:    d.TimeWindow,
			Multiplier:    d.Multiplier,
		}
	}

	rule, err := s.repo.PublishNewFineRule(details)
	if err != nil {
		return nil, fmt.Errorf("failed to publish new fine rule: %w", err)
	}

	log.Printf("[billing-service] new FineRule published | version=%d id=%d details=%d",
		rule.Version, rule.ID, len(rule.Details))

	return rule, nil
}

// ─── Calculation Helpers ──────────────────────────────────────────────────────

// timeMultiplier returns 1.5 if the violation occurred between 22:00–05:59 WIB,
// and 1.0 otherwise (06:00–21:59 WIB).
//
// The timestamp is converted to Asia/Jakarta before extracting the hour so that
// violations recorded in UTC are evaluated correctly.
func timeMultiplier(t time.Time) float64 {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		// Fallback: use UTC hour if timezone data is unavailable.
		loc = time.UTC
	}
	hour := t.In(loc).Hour()

	// Night window: 22:00 (inclusive) → 06:00 (exclusive)
	if hour >= 22 || hour < 6 {
		return 1.5 // MultiplierNight
	}
	return 1.0 // MultiplierDaytime
}

// repeatMultiplier maps the count of prior UNPAID violations (within 90 days)
// to the appropriate fine multiplier tier.
//
//	0 unpaid  → 1.0×
//	1 unpaid  → 1.5×
//	2+ unpaid → 2.0×
func repeatMultiplier(priorUnpaidCount int64) float64 {
	switch {
	case priorUnpaidCount >= 2:
		return 2.0
	case priorUnpaidCount == 1:
		return 1.5
	default:
		return 1.0
	}
}

// generateInvoiceNo produces a human-readable, unique invoice reference.
// Format: INV-YYYYMMDD-{violationID zero-padded to 6 digits}
// Example: INV-20240115-000042
func generateInvoiceNo(violationID uint, ts time.Time) string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return fmt.Sprintf("INV-%s-%06d", ts.In(loc).Format("20060102"), violationID)
}
