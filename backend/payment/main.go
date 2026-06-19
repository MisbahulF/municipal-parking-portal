package main

import (
	"log"
	"os"

	"github.com/dhill/parking-violation-portal/backend/payment/handler"
	"github.com/dhill/parking-violation-portal/backend/payment/repository"
	"github.com/dhill/parking-violation-portal/backend/payment/service"
	"github.com/dhill/parking-violation-portal/shared/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// ── 1. Database ──────────────────────────────────────────────────────────
	if err := database.InitPostgres(); err != nil {
		log.Fatalf("[payment-service] DB init failed: %v", err)
	}
	// Payment service only needs schema to be up-to-date, not the seed.
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("[payment-service] DB migration failed: %v", err)
	}

	// ── 2. Dependency injection ──────────────────────────────────────────────
	repo := repository.New(database.GetDB())
	svc := service.New(repo)
	h := handler.New(svc)

	// ── 3. Gin router ────────────────────────────────────────────────────────
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Health check — used by api-gateway and Docker health checks.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "payment", "status": "ok"})
	})

	// Public API
	v1 := r.Group("/api/v1")
	{
		// Flow 4 (officer/internal): pay via invoice ID
		v1.POST("/invoices/:id/pay", h.Pay)

		// Flow 4 (public member endpoint): POST /payments/:invoice_id/pay
		v1.POST("/payments/:invoice_id/pay", h.PayPublic)

		// Check current invoice status (unpaid / paid / voided)
		v1.GET("/invoices/:id/status", h.GetInvoiceStatus)

		// Officer view: all invoices for a vehicle's license plate
		v1.GET("/vehicles/:plate/invoices", h.GetInvoicesByPlate)
	}

	// ── 4. Start server ──────────────────────────────────────────────────────
	port := getEnv("PORT", "8083")
	log.Printf("[payment-service] starting on :%s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[payment-service] server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
