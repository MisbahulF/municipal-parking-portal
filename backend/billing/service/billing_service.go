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

type GenerateInvoiceInput struct {
	ViolationID  uint   `json:"violation_id"`
	LicensePlate string `json:"license_plate"`
}

type FineRuleDetailInput struct {
	ViolationType models.ViolationType `json:"violation_type"`
	BaseAmount    float64              `json:"base_amount"`
	LogicType     models.LogicType     `json:"logic_type"`
	TimeWindow    int                  `json:"time_window"`
	Multiplier    float64              `json:"multiplier"`
}

type PublishFineRuleInput struct {
	Details []FineRuleDetailInput `json:"details"`
}

var (
	ErrInvoiceAlreadyExists   = errors.New("invoice already exists for this violation")
	ErrNoFineRuleDetails      = errors.New("at least one fine rule detail is required")
	ErrDuplicateViolationType = errors.New("duplicate violation_type in rule details")
)

type BillingService interface {
	GenerateInvoice(input GenerateInvoiceInput) (*models.Invoice, error)
	GetInvoiceByID(id uint) (*models.Invoice, error)
	PublishFineRule(input PublishFineRuleInput) (*models.FineRule, error)
	GetActiveFineRule() (*models.FineRule, error)
	GetAllInvoices() ([]models.Invoice, error)
}

type billingService struct {
	repo repository.BillingRepository
}

func New(repo repository.BillingRepository) BillingService {
	return &billingService{repo: repo}
}

func (s *billingService) GetActiveFineRule() (*models.FineRule, error) {
	return s.repo.FindActiveFineRule()
}

func (s *billingService) GetAllInvoices() ([]models.Invoice, error) {
	return s.repo.FindAllInvoices()
}

func (s *billingService) GenerateInvoice(input GenerateInvoiceInput) (*models.Invoice, error) {
	exists, err := s.repo.InvoiceExistsByViolationID(input.ViolationID)
	if err != nil {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		return nil, ErrInvoiceAlreadyExists
	}

	violation, err := s.repo.FindViolationByID(input.ViolationID)
	if err != nil {
		return nil, fmt.Errorf("violation %d not found: %w", input.ViolationID, err)
	}

	activeRule, err := s.repo.FindActiveFineRule()
	if err != nil {
		return nil, fmt.Errorf("no active FineRule found: %w", err)
	}

	detail, err := s.repo.FindDetailByViolationType(activeRule.ID, violation.ViolationType)
	if err != nil {
		return nil, fmt.Errorf("no fine detail for violation type %q in rule v%d: %w",
			violation.ViolationType, activeRule.Version, err)
	}

	timeMult := timeMultiplier(violation.Timestamp)

	priorUnpaid, err := s.repo.CountPriorUnpaid(
		violation.LicensePlate,
		violation.Timestamp,
		violation.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("repeat-offender query failed: %w", err)
	}
	repeatMult := repeatMultiplier(priorUnpaid)

	calculatedAmount := math.Round(detail.BaseAmount*timeMult*repeatMult*100) / 100

	log.Printf(
		"[billing-service] fine calc | violation=%d type=%s base=%.0f time=%.1fx repeat=%.1fx (prior_unpaid=%d) → %.0f",
		violation.ID, violation.ViolationType,
		detail.BaseAmount, timeMult, repeatMult, priorUnpaid,
		calculatedAmount,
	)

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

func (s *billingService) GetInvoiceByID(id uint) (*models.Invoice, error) {
	inv, err := s.repo.FindInvoiceByID(id)
	if err != nil {
		return nil, fmt.Errorf("invoice %d not found: %w", id, err)
	}
	return inv, nil
}

func (s *billingService) PublishFineRule(input PublishFineRuleInput) (*models.FineRule, error) {
	if len(input.Details) == 0 {
		return nil, ErrNoFineRuleDetails
	}

	seen := make(map[models.ViolationType]struct{}, len(input.Details))
	for _, d := range input.Details {
		if _, exists := seen[d.ViolationType]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateViolationType, d.ViolationType)
		}
		seen[d.ViolationType] = struct{}{}
	}

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

func timeMultiplier(t time.Time) float64 {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.UTC
	}
	hour := t.In(loc).Hour()

	if hour >= 22 || hour < 6 {
		return 1.5
	}
	return 1.0
}

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

func generateInvoiceNo(violationID uint, ts time.Time) string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return fmt.Sprintf("INV-%s-%06d", ts.In(loc).Format("20060102"), violationID)
}
