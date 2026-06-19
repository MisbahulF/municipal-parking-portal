package main

import (
	"log"
	"os"

	"github.com/dhill/parking-violation-portal/backend/billing/handler"
	"github.com/dhill/parking-violation-portal/backend/billing/repository"
	"github.com/dhill/parking-violation-portal/backend/billing/service"
	"github.com/dhill/parking-violation-portal/shared/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// ── 1. Database ──────────────────────────────────────────────────────────
	if err := database.InitPostgres(); err != nil {
		log.Fatalf("[billing-service] DB init failed: %v", err)
	}
	// Note: billing service does NOT re-seed; violation service owns the seed.
	// AutoMigrate alone ensures schema is up-to-date.
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("[billing-service] DB migration failed: %v", err)
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

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "billing", "status": "ok"})
	})

	// Internal endpoint — called by the violation service only.
	// In production this should be protected by network policy or an HMAC header.
	internal := r.Group("/internal")
	{
		internal.POST("/invoices", h.GenerateInvoice) // Flow 2 & 5: calculate + create invoice
	}

	// Public API
	v1 := r.Group("/api/v1")
	{
		v1.GET("/invoices", h.GetAllInvoices)      // Fetch all invoices
		v1.GET("/invoices/:id", h.GetInvoice)      // Fetch invoice with full breakdown
		v1.POST("/fine-rules", h.PublishFineRule)  // Flow 3: publish new rule version
		v1.GET("/fine-rules/active", h.GetActiveFineRule) // Fetch active fine ruleset
	}

	// ── 4. Start server ──────────────────────────────────────────────────────
	port := getEnv("PORT", "8082")
	log.Printf("[billing-service] starting on :%s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[billing-service] server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
