package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/dhill/parking-violation-portal/backend/payment/service"
	"github.com/gin-gonic/gin"
)

// PaymentHandler holds the HTTP handler methods for the payment service.
type PaymentHandler struct {
	svc service.PaymentService
}

// New returns a PaymentHandler wired to the given service.
func New(svc service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// payRequest is the JSON body accepted by POST /api/v1/invoices/:id/pay.
type payRequest struct {
	// Scenario controls the mock engine behaviour.
	// Accepted values: "success" (default) | "failed"
	Scenario string `json:"scenario" binding:"required,oneof=success failed"`
}

// Pay handles POST /api/v1/invoices/:id/pay — Flow 4.
//
// Request body:
//
//	{ "scenario": "success" }   →  invoice marked PAID, 200 OK
//	{ "scenario": "failed"  }   →  engine declined, 402 Payment Required
//
// Other responses:
//   - 404 Not Found            invoice does not exist
//   - 409 Conflict             invoice already paid
//   - 422 Unprocessable Entity invoice voided
func (h *PaymentHandler) Pay(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}

	var req payRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "invalid request body",
			"detail": err.Error(),
		})
		return
	}

	result, err := h.svc.ProcessPayment(id, req.Scenario)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvoiceNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

		case errors.Is(err, service.ErrInvoiceAlreadyPaid):
			c.JSON(http.StatusConflict, gin.H{
				"error":  "invoice has already been paid",
				"detail": err.Error(),
			})

		case errors.Is(err, service.ErrInvoiceVoided):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})

		case errors.Is(err, service.ErrPaymentFailed):
			// Engine declined: return trace info so the caller knows the TX ID.
			c.JSON(http.StatusPaymentRequired, gin.H{
				"message":        "Payment gateway declined the transaction.",
				"charge_status":  result.ChargeStatus,
				"transaction_id": result.TransactionID,
				"invoice_status": result.Invoice.Status, // still UNPAID
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Payment confirmed. Invoice marked as PAID.",
		"charge_status":  result.ChargeStatus,
		"transaction_id": result.TransactionID,
		"data":           result.Invoice,
	})
}

// PayPublic handles POST /api/v1/payments/:invoice_id/pay — the public member endpoint.
//
// Identical contract to Pay but:
//   - Uses the :invoice_id param name (matches the spec route).
//   - Calls PayInvoice which logs every attempt and persists transaction_id on success.
func (h *PaymentHandler) PayPublic(c *gin.Context) {
	idStr := c.Param("invoice_id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invoice_id must be a positive integer"})
		return
	}

	var req payRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "invalid request body",
			"detail": err.Error(),
		})
		return
	}

	result, err := h.svc.PayInvoice(uint(id), req.Scenario)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvoiceNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInvoiceAlreadyPaid):
			c.JSON(http.StatusConflict, gin.H{"error": "invoice has already been paid", "detail": err.Error()})
		case errors.Is(err, service.ErrInvoiceVoided):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrPaymentFailed):
			c.JSON(http.StatusPaymentRequired, gin.H{
				"message":        "Payment gateway declined the transaction.",
				"charge_status":  result.ChargeStatus,
				"transaction_id": result.TransactionID,
				"invoice_status": result.Invoice.Status,
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Payment confirmed. Invoice marked as PAID.",
		"charge_status":  result.ChargeStatus,
		"transaction_id": result.TransactionID,
		"data":           result.Invoice,
	})
}

// GetInvoiceStatus handles GET /api/v1/invoices/:id/status.
func (h *PaymentHandler) GetInvoiceStatus(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}

	invoice, err := h.svc.GetInvoiceStatus(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invoice})
}

// GetInvoicesByPlate handles GET /api/v1/vehicles/:plate/invoices.
func (h *PaymentHandler) GetInvoicesByPlate(c *gin.Context) {
	plate := c.Param("plate")
	if plate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license plate is required"})
		return
	}

	invoices, err := h.svc.GetInvoicesByPlate(plate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"license_plate": plate,
		"count":         len(invoices),
		"data":          invoices,
	})
}

// parseID extracts and validates the :id route parameter.
func parseID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a positive integer"})
		return 0, err
	}
	return uint(id), nil
}
