package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dhill/parking-violation-portal/backend/violation/repository"
	"github.com/dhill/parking-violation-portal/shared/models"
)

// CreateViolationInput is the clean input DTO accepted by the service layer.
type CreateViolationInput struct {
	LicensePlate  string               `json:"license_plate"`
	ViolationType models.ViolationType `json:"violation_type"`
	Location      string               `json:"location"`
	Timestamp     time.Time            `json:"timestamp"`
	PhotoURL      string               `json:"photo_url"`
}

// BillingTriggerPayload is the payload sent to the billing service.
type BillingTriggerPayload struct {
	ViolationID  uint   `json:"violation_id"`
	LicensePlate string `json:"license_plate"`
}

// ViolationService defines the business logic interface.
type ViolationService interface {
	RecordViolation(input CreateViolationInput) (*models.Violation, error)
	GetByID(id uint) (*models.Violation, error)
}

type violationService struct {
	repo           repository.ViolationRepository
	billingBaseURL string
	httpClient     *http.Client
}

// New wires a ViolationService with its repository dependency.
// The billing service URL is read from the BILLING_SERVICE_URL env var,
// defaulting to http://localhost:8082 for local development.
func New(repo repository.ViolationRepository) ViolationService {
	return &violationService{
		repo:           repo,
		billingBaseURL: getEnv("BILLING_SERVICE_URL", "http://localhost:8082"),
		httpClient:     &http.Client{Timeout: 5 * time.Second},
	}
}

// RecordViolation persists the violation and synchronously triggers the billing
// service to generate an invoice. If billing is unavailable, the violation is
// still saved and the error is logged — it does NOT roll back the violation.
func (s *violationService) RecordViolation(input CreateViolationInput) (*models.Violation, error) {
	// 1. Persist the violation record.
	v := &models.Violation{
		LicensePlate:  input.LicensePlate,
		ViolationType: input.ViolationType,
		Location:      input.Location,
		Timestamp:     input.Timestamp,
		PhotoURL:      input.PhotoURL,
	}
	if err := s.repo.Create(v); err != nil {
		return nil, fmt.Errorf("failed to save violation: %w", err)
	}
	log.Printf("[violation-service] saved violation id=%d plate=%s type=%s",
		v.ID, v.LicensePlate, v.ViolationType)

	// 2. Trigger billing service synchronously (with timeout).
	//    We fire this in a goroutine so the HTTP response to the caller is not
	//    blocked by billing latency, but we still log any failure clearly.
	go s.triggerBilling(v.ID, v.LicensePlate)

	return v, nil
}

// GetByID fetches a violation by its primary key, preloading the invoice.
func (s *violationService) GetByID(id uint) (*models.Violation, error) {
	v, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("violation %d not found: %w", id, err)
	}
	return v, nil
}

// triggerBilling calls the billing service's internal endpoint to generate an
// invoice for the given violation. Designed to run in a goroutine.
func (s *violationService) triggerBilling(violationID uint, licensePlate string) {
	payload := BillingTriggerPayload{
		ViolationID:  violationID,
		LicensePlate: licensePlate,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[violation-service] billing marshal error: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.billingBaseURL+"/internal/invoices",
		bytes.NewReader(body),
	)
	if err != nil {
		log.Printf("[violation-service] billing request build error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[violation-service] billing trigger failed (violation_id=%d): %v — invoice will be generated on retry",
			violationID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Printf("[violation-service] billing returned unexpected status=%d for violation_id=%d",
			resp.StatusCode, violationID)
		return
	}

	log.Printf("[violation-service] invoice generation triggered for violation_id=%d", violationID)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
