package repository

import (
	"github.com/dhill/parking-violation-portal/shared/models"
	"gorm.io/gorm"
)

// ViolationRepository defines the persistence interface for the violation service.
type ViolationRepository interface {
	Create(v *models.Violation) error
	FindByID(id uint) (*models.Violation, error)
	// CountUnpaidByPlate is used by the billing service trigger to determine
	// the repeat-offender multiplier for a given license plate.
	CountUnpaidByPlate(licensePlate string) (int64, error)
}

type violationRepository struct {
	db *gorm.DB
}

// New returns a concrete ViolationRepository backed by the given GORM DB.
func New(db *gorm.DB) ViolationRepository {
	return &violationRepository{db: db}
}

func (r *violationRepository) Create(v *models.Violation) error {
	return r.db.Create(v).Error
}

func (r *violationRepository) FindByID(id uint) (*models.Violation, error) {
	var v models.Violation
	// Preload Invoice so callers get the full billing snapshot in one call.
	if err := r.db.Preload("Invoice").First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *violationRepository) CountUnpaidByPlate(licensePlate string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Invoice{}).
		Joins("JOIN violations ON violations.id = invoices.violation_id").
		Where("violations.license_plate = ? AND invoices.status = ?",
			licensePlate, models.InvoiceStatusUnpaid).
		Count(&count).Error
	return count, err
}
