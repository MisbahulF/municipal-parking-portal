package repository

import (
	"fmt"
	"time"

	"github.com/dhill/parking-violation-portal/shared/models"
	"gorm.io/gorm"
)

// BillingRepository defines all DB operations needed by the billing service.
type BillingRepository interface {
	// FindViolationByID fetches the full violation record needed for calculation.
	FindViolationByID(id uint) (*models.Violation, error)

	// FindActiveFineRule returns the currently active FineRule with all its Details preloaded.
	FindActiveFineRule() (*models.FineRule, error)

	// FindDetailByViolationType returns the FineRuleDetail for a specific type within a rule version.
	FindDetailByViolationType(fineRuleID uint, vtype models.ViolationType) (*models.FineRuleDetail, error)

	// CountPriorUnpaid counts UNPAID invoices for the same license plate whose
	// associated violation timestamp falls within [refTime - 90 days, refTime).
	// The current violationID is excluded so it does not count itself.
	CountPriorUnpaid(licensePlate string, refTime time.Time, excludeViolationID uint) (int64, error)

	// InvoiceExistsByViolationID checks for an existing invoice to ensure idempotency.
	InvoiceExistsByViolationID(violationID uint) (bool, error)

	// CreateInvoice persists the generated invoice snapshot.
	CreateInvoice(inv *models.Invoice) error

	// FindInvoiceByID fetches an invoice with its related Violation and FineRule.
	FindInvoiceByID(id uint) (*models.Invoice, error)

	// FindAllInvoices returns all invoices in the database.
	FindAllInvoices() ([]models.Invoice, error)

	// PublishNewFineRule atomically deactivates the current active rule and
	// creates a new version with the provided details.
	// All writes happen inside a single transaction — either everything commits
	// or nothing does, so existing invoices are never touched.
	PublishNewFineRule(details []models.FineRuleDetail) (*models.FineRule, error)
}

type billingRepository struct {
	db *gorm.DB
}

// New returns a concrete BillingRepository backed by the given GORM DB.
func New(db *gorm.DB) BillingRepository {
	return &billingRepository{db: db}
}

func (r *billingRepository) FindViolationByID(id uint) (*models.Violation, error) {
	var v models.Violation
	if err := r.db.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *billingRepository) FindActiveFineRule() (*models.FineRule, error) {
	var rule models.FineRule
	err := r.db.
		Preload("Details").
		Where("is_active = ?", true).
		Order("version DESC"). // newest active version wins
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *billingRepository) FindDetailByViolationType(fineRuleID uint, vtype models.ViolationType) (*models.FineRuleDetail, error) {
	var detail models.FineRuleDetail
	err := r.db.
		Where("fine_rule_id = ? AND violation_type = ?", fineRuleID, vtype).
		First(&detail).Error
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

// CountPriorUnpaid implements the 90-day repeat-offender window:
//
//	SELECT COUNT(*) FROM invoices
//	  JOIN violations ON violations.id = invoices.violation_id
//	  WHERE violations.license_plate = ?
//	    AND invoices.status         = 'UNPAID'
//	    AND violations.timestamp    BETWEEN (refTime - 90 days) AND refTime
//	    AND violations.id           != excludeViolationID
func (r *billingRepository) CountPriorUnpaid(licensePlate string, refTime time.Time, excludeViolationID uint) (int64, error) {
	cutoff := refTime.AddDate(0, 0, -90)
	var count int64
	err := r.db.Model(&models.Invoice{}).
		Joins("JOIN violations ON violations.id = invoices.violation_id").
		Where("violations.license_plate = ?", licensePlate).
		Where("invoices.status = ?", models.InvoiceStatusUnpaid).
		Where("violations.timestamp >= ? AND violations.timestamp < ?", cutoff, refTime).
		Where("violations.id != ?", excludeViolationID).
		Count(&count).Error
	return count, err
}

func (r *billingRepository) InvoiceExistsByViolationID(violationID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Invoice{}).
		Where("violation_id = ?", violationID).
		Count(&count).Error
	return count > 0, err
}

func (r *billingRepository) CreateInvoice(inv *models.Invoice) error {
	return r.db.Create(inv).Error
}

func (r *billingRepository) FindInvoiceByID(id uint) (*models.Invoice, error) {
	var inv models.Invoice
	err := r.db.
		Preload("Violation").
		Preload("AppliedFineRule").
		Preload("AppliedFineRule.Details").
		First(&inv, id).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *billingRepository) FindAllInvoices() ([]models.Invoice, error) {
	var invoices []models.Invoice
	err := r.db.
		Preload("Violation").
		Preload("AppliedFineRule").
		Order("created_at DESC").
		Find(&invoices).Error
	return invoices, err
}

// PublishNewFineRule implements the atomic rule-versioning flow (Flow 3):
//
//  1. Fetch the highest existing version number.
//  2. Deactivate ALL currently active rules (safety net for data inconsistency).
//  3. Insert new FineRule with version = max+1, is_active = true.
//  4. Insert all provided FineRuleDetails linked to the new rule.
//
// The entire operation runs inside a single DB transaction.
// Existing invoices are NOT touched — they retain their AppliedFineRuleID.
func (r *billingRepository) PublishNewFineRule(details []models.FineRuleDetail) (*models.FineRule, error) {
	var newRule *models.FineRule

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Determine the next version number.
		var maxVersion int
		tx.Model(&models.FineRule{}).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)

		// Step 2: Deactivate all currently active rules.
		if err := tx.Model(&models.FineRule{}).
			Where("is_active = ?", true).
			Update("is_active", false).Error; err != nil {
			return fmt.Errorf("failed to deactivate current rules: %w", err)
		}

		// Step 3: Create the new versioned rule header.
		rule := &models.FineRule{
			Version:  maxVersion + 1,
			IsActive: true,
		}
		if err := tx.Create(rule).Error; err != nil {
			return fmt.Errorf("failed to create new FineRule: %w", err)
		}

		// Step 4: Attach details to the new rule.
		for i := range details {
			details[i].ID = 0         // reset PK so GORM inserts new rows
			details[i].FineRuleID = rule.ID
		}
		if err := tx.Create(&details).Error; err != nil {
			return fmt.Errorf("failed to create FineRuleDetails: %w", err)
		}

		rule.Details = details
		newRule = rule
		return nil
	})

	if err != nil {
		return nil, err
	}
	return newRule, nil
}
