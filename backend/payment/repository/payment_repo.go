package repository

import (
	"time"

	"github.com/dhill/parking-violation-portal/shared/models"
	"gorm.io/gorm"
)

// PaymentRepository defines the DB operations needed by the payment service.
type PaymentRepository interface {
	// FindInvoiceByID returns a single invoice with its preloaded Violation.
	FindInvoiceByID(id uint) (*models.Invoice, error)

	// MarkAsPaid atomically sets status = PAID and paid_at = paidAt for the given invoice.
	MarkAsPaid(id uint, paidAt time.Time) (*models.Invoice, error)

	// MarkAsPaidWithTx sets status = PAID, paid_at, and transaction_id atomically.
	// Used by the public payment endpoint which persists the gateway TX reference.
	MarkAsPaidWithTx(id uint, paidAt time.Time, transactionID string) (*models.Invoice, error)

	// LogPaymentAttempt appends an audit row to payment_attempts for every charge
	// call, regardless of outcome. Never returns an error that should block the response.
	LogPaymentAttempt(attempt *models.PaymentAttempt) error

	// FindInvoicesByPlate returns all invoices for a license plate, newest-first.
	FindInvoicesByPlate(licensePlate string) ([]models.Invoice, error)
}

type paymentRepository struct {
	db *gorm.DB
}

// New returns a concrete PaymentRepository backed by the given GORM DB.
func New(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) FindInvoiceByID(id uint) (*models.Invoice, error) {
	var inv models.Invoice
	err := r.db.
		Preload("Violation").
		Preload("AppliedFineRule").
		First(&inv, id).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// MarkAsPaid uses a targeted UPDATE inside a transaction so that:
//   - The WHERE clause enforces status = 'UNPAID' at the DB level (optimistic guard).
//   - paid_at is set to the exact moment the payment was confirmed.
//   - The updated record is returned with a fresh SELECT.
func (r *paymentRepository) MarkAsPaid(id uint, paidAt time.Time) (*models.Invoice, error) {
	var updated models.Invoice

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Invoice{}).
			Where("id = ? AND status = ?", id, models.InvoiceStatusUnpaid).
			Updates(map[string]interface{}{
				"status":  models.InvoiceStatusPaid,
				"paid_at": paidAt,
			})

		if result.Error != nil {
			return result.Error
		}
		// RowsAffected == 0 means either not found OR already paid/voided.
		// The caller (service) is responsible for distinguishing these cases
		// by pre-fetching the invoice first.
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return tx.Preload("Violation").Preload("AppliedFineRule").First(&updated, id).Error
	})

	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// MarkAsPaidWithTx is the full payment finalisation: sets status, paid_at, AND transaction_id.
func (r *paymentRepository) MarkAsPaidWithTx(id uint, paidAt time.Time, transactionID string) (*models.Invoice, error) {
	var updated models.Invoice

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Invoice{}).
			Where("id = ? AND status = ?", id, models.InvoiceStatusUnpaid).
			Updates(map[string]interface{}{
				"status":         models.InvoiceStatusPaid,
				"paid_at":        paidAt,
				"transaction_id": transactionID,
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Preload("Violation").Preload("AppliedFineRule").First(&updated, id).Error
	})

	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// LogPaymentAttempt inserts an audit record for every charge call.
// Errors are suppressed — the audit trail must never block the payment response.
func (r *paymentRepository) LogPaymentAttempt(attempt *models.PaymentAttempt) error {
	return r.db.Create(attempt).Error
}

func (r *paymentRepository) FindInvoicesByPlate(licensePlate string) ([]models.Invoice, error) {
	var invoices []models.Invoice
	err := r.db.
		Preload("Violation").
		Joins("JOIN violations ON violations.id = invoices.violation_id").
		Where("violations.license_plate = ?", licensePlate).
		Order("invoices.created_at DESC").
		Find(&invoices).Error
	if err != nil {
		return nil, err
	}
	return invoices, nil
}
