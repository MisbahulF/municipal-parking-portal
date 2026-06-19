package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/dhill/parking-violation-portal/backend/violation/service"
	"github.com/dhill/parking-violation-portal/shared/models"
	"github.com/gin-gonic/gin"
)

// ViolationHandler holds the HTTP handler methods for the violation service.
type ViolationHandler struct {
	svc service.ViolationService
}

// New returns a ViolationHandler wired to the given service.
func New(svc service.ViolationService) *ViolationHandler {
	return &ViolationHandler{svc: svc}
}

// createViolationRequest is the JSON body accepted by POST /violations.
type createViolationRequest struct {
	LicensePlate  string               `json:"license_plate"  binding:"required"`
	ViolationType models.ViolationType `json:"violation_type" binding:"required"`
	Location      string               `json:"location"       binding:"required"`
	// Timestamp is the observed time of the violation (ISO 8601 / RFC 3339).
	// If omitted, the server time is used.
	Timestamp time.Time `json:"timestamp"`
	PhotoURL  string    `json:"photo_url"`
}

// CreateViolation handles POST /violations.
//
// Flow:
//  1. Validate request body.
//  2. Default Timestamp to now if not provided.
//  3. Delegate to service layer (saves + triggers billing).
//  4. Return 201 with the created violation record.
func (h *ViolationHandler) CreateViolation(c *gin.Context) {
	var req createViolationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "invalid request body",
			"detail": err.Error(),
		})
		return
	}

	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	v, err := h.svc.RecordViolation(service.CreateViolationInput{
		LicensePlate:  req.LicensePlate,
		ViolationType: req.ViolationType,
		Location:      req.Location,
		Timestamp:     req.Timestamp,
		PhotoURL:      req.PhotoURL,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Violation recorded. Invoice generation has been triggered.",
		"data":    v,
	})
}

// GetViolation handles GET /violations/:id.
// Returns the violation with its preloaded invoice (if already generated).
func (h *ViolationHandler) GetViolation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a positive integer"})
		return
	}

	v, err := h.svc.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": v})
}
