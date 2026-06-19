package main

import (
	"log"
	"os"

	"github.com/dhill/parking-violation-portal/backend/violation/handler"
	"github.com/dhill/parking-violation-portal/backend/violation/repository"
	"github.com/dhill/parking-violation-portal/backend/violation/service"
	"github.com/dhill/parking-violation-portal/shared/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// ── 1. Database ──────────────────────────────────────────────────────────
	if err := database.InitPostgres(); err != nil {
		log.Fatalf("[violation-service] DB init failed: %v", err)
	}
	if err := database.MigrateAndSeed(); err != nil {
		log.Fatalf("[violation-service] DB migration/seed failed: %v", err)
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
		c.JSON(200, gin.H{"service": "violation", "status": "ok"})
	})

	// Public API
	v1 := r.Group("/api/v1")
	{
		v1.POST("/violations", h.CreateViolation) // Flow 1: record a new violation
		v1.GET("/violations/:id", h.GetViolation) // Fetch violation + its invoice
	}

	// ── 4. Start server ──────────────────────────────────────────────────────
	port := getEnv("PORT", "8081")
	log.Printf("[violation-service] starting on :%s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[violation-service] server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
