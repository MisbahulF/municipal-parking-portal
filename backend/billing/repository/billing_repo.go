package repository

import (
	"fmt"
	"time"

	"github.com/dhill/parking-violation-portal/shared/models"
	"gorm.io/gorm"
)

type BillingRepository interface {
	FindViolationByID(id uint) (*models.Violation, error)
	FindActiveFineRule() (*models.FineRule, error)
	FindDetailByViolationType(fineRuleID uint, vtype models.ViolationType) (*models.FineRuleDetail, error)
	CountPriorUnpaid(licensePlate string, refTime time.Time, excludeViolationID uint) (int64, error)
	InvoiceExistsByViolationID(violationID uint) (bool, error)
	CreateInvoice(inv *models.Invoice) error
	FindInvoiceByID(id uint) (*models.Invoice, error)
	FindAllInvoices() ([]models.Invoice, error)
	PublishNewFineRule(details []models.FineRuleDetail) (*models.FineRule, error)
}

type billingRepository struct {
	db *gorm.DB
}

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
		Order("version DESC").
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

func (r *billingRepository) PublishNewFineRule(details []models.FineRuleDetail) (*models.FineRule, error) {
	var newRule *models.FineRule

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var maxVersion int
		tx.Model(&models.FineRule{}).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)

		if err := tx.Model(&models.FineRule{}).
			Where("is_active = ?", true).
			Update("is_active", false).Error; err != nil {
			return fmt.Errorf("failed to deactivate current rules: %w", err)
		}

		rule := &models.FineRule{
			Version:  maxVersion + 1,
			IsActive: true,
		}
		if err := tx.Create(rule).Error; err != nil {
			return fmt.Errorf("failed to create new FineRule: %w", err)
		}

		for i := range details {
			details[i].ID = 0
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
