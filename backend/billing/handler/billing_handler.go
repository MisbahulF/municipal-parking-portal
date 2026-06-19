package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dhill/parking-violation-portal/backend/billing/service"
	"github.com/dhill/parking-violation-portal/shared/models"
	"github.com/gin-gonic/gin"
)

// BillingHandler holds the HTTP handler methods for the billing service.
type BillingHandler struct {
	svc service.BillingService
}

// New returns a BillingHandler wired to the given service.
func New(svc service.BillingService) *BillingHandler {
	return &BillingHandler{svc: svc}
}

// generateInvoiceRequest is the JSON body accepted by POST /internal/invoices.
// This endpoint is internal-only and called exclusively by the violation service.
type generateInvoiceRequest struct {
	ViolationID  uint   `json:"violation_id"  binding:"required"`
	LicensePlate string `json:"license_plate" binding:"required"`
}

// GenerateInvoice handles POST /internal/invoices.
//
// Called by the violation service after a new violation is persisted.
// Returns 201 on success, 409 if an invoice already exists (idempotent),
// 422 if the active FineRule has no matching detail for the violation type.
func (h *BillingHandler) GenerateInvoice(c *gin.Context) {
	var req generateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "invalid request body",
			"detail": err.Error(),
		})
		return
	}

	invoice, err := h.svc.GenerateInvoice(service.GenerateInvoiceInput{
		ViolationID:  req.ViolationID,
		LicensePlate: req.LicensePlate,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvoiceAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "invoice already exists for this violation",
			})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Invoice generated successfully.",
		"data":    invoice,
	})
}

// GetInvoice handles GET /api/v1/invoices/:id.
// Returns the full invoice with preloaded Violation and AppliedFineRule.
func (h *BillingHandler) GetInvoice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a positive integer"})
		return
	}

	invoice, err := h.svc.GetInvoiceByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invoice})
}

// ─── Fine Rule Management (Flow 3) ───────────────────────────────────────────

// publishFineRuleDetailRequest represents a single violation type's rule config.
type publishFineRuleDetailRequest struct {
	ViolationType models.ViolationType `json:"violation_type" binding:"required"`
	BaseAmount    float64              `json:"base_amount"    binding:"required,gt=0"`
	LogicType     models.LogicType     `json:"logic_type"     binding:"required"`
	// TimeWindow is the interval in minutes used for PROGRESSIVE logic (0 for FLAT).
	TimeWindow int     `json:"time_window"`
	Multiplier float64 `json:"multiplier"`
}

// publishFineRuleRequest is the JSON body accepted by POST /fine-rules.
type publishFineRuleRequest struct {
	// Details must contain at least one entry and no duplicate violation_types.
	Details []publishFineRuleDetailRequest `json:"details" binding:"required,min=1,dive"`
}

// PublishFineRule handles POST /api/v1/fine-rules.
//
// Flow 3: An officer submits a new complete ruleset. The service:
//  1. Validates the payload (no empty list, no duplicate types, base_amount > 0).
//  2. Atomically deactivates the current active FineRule.
//  3. Creates a new FineRule (version = prev + 1, is_active = true).
//  4. Creates FineRuleDetails for the new version.
//  5. Returns the new rule. All EXISTING invoices remain unchanged.
func (h *BillingHandler) PublishFineRule(c *gin.Context) {
	var req publishFineRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "invalid request body",
			"detail": err.Error(),
		})
		return
	}

	// Map request DTOs → service input
	details := make([]service.FineRuleDetailInput, len(req.Details))
	for i, d := range req.Details {
		details[i] = service.FineRuleDetailInput{
			ViolationType: d.ViolationType,
			BaseAmount:    d.BaseAmount,
			LogicType:     d.LogicType,
			TimeWindow:    d.TimeWindow,
			Multiplier:    d.Multiplier,
		}
	}

	rule, err := h.svc.PublishFineRule(service.PublishFineRuleInput{Details: details})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoFineRuleDetails):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrDuplicateViolationType):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "New fine rule version published. Existing invoices are unaffected.",
		"data":    rule,
	})
}

// GetActiveFineRule handles GET /api/v1/fine-rules/active.
func (h *BillingHandler) GetActiveFineRule(c *gin.Context) {
	rule, err := h.svc.GetActiveFineRule()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active fine rule found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rule})
}

// GetAllInvoices handles GET /api/v1/invoices.
func (h *BillingHandler) GetAllInvoices(c *gin.Context) {
	invoices, err := h.svc.GetAllInvoices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": invoices})
}
