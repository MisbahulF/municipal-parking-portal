package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// ─── Service Registry ─────────────────────────────────────────────────────────

// services holds the base URLs for all downstream microservices.
// Each value is read from an env var with a local-dev default.
var services = map[string]string{
	"violation": getEnv("VIOLATION_SERVICE_URL", "http://localhost:8081"),
	"billing":   getEnv("BILLING_SERVICE_URL", "http://localhost:8082"),
	"payment":   getEnv("PAYMENT_SERVICE_URL", "http://localhost:8083"),
}

// ─── Reverse Proxy Factory ────────────────────────────────────────────────────

// newProxy creates a Gin handler that forwards every request to target.
//
// It:
//   - Strips the X-Forwarded-For header added by previous proxies and sets a clean one.
//   - Rewrites the Host header to the target host (avoids SNI / vhost mismatches).
//   - Returns a structured JSON error body instead of an HTML error page when the
//     downstream service is unreachable.
func newProxy(targetRaw string) gin.HandlerFunc {
	target, err := url.Parse(targetRaw)
	if err != nil {
		log.Fatalf("[api-gateway] invalid service URL %q: %v", targetRaw, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Director runs before the request is forwarded — mutate as needed.
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host // forward correct Host header
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Header.Set("X-Origin-Host", target.Host)
	}

	// ErrorHandler returns JSON instead of the default plain-text error.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[api-gateway] upstream error [%s%s]: %v", targetRaw, r.URL.Path, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "upstream service unavailable",
			"service": targetRaw,
			"detail":  err.Error(),
		})
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// ─── Health Check ─────────────────────────────────────────────────────────────

// downstreamHealth pings a service's /health endpoint and returns its status.
func downstreamHealth(name, baseURL string) map[string]string {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		status := "unreachable"
		if err == nil {
			status = fmt.Sprintf("http_%d", resp.StatusCode)
		}
		return map[string]string{"service": name, "status": status}
	}
	defer resp.Body.Close()
	return map[string]string{"service": name, "status": "ok"}
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// ── CORS ─────────────────────────────────────────────────────────────────
	// AllowOrigins is read from CORS_ORIGIN (comma-separated list).
	// Default: http://localhost:3000 (Next.js dev server).
	// Example for multi-origin prod: CORS_ORIGIN=https://app.example.com,https://admin.example.com
	allowedOrigins := strings.Split(
		getEnv("CORS_ORIGIN", "http://localhost:3000"),
		",",
	)

	r.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-Forwarded-For",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
		},
		// AllowCredentials lets the browser send cookies and Authorization headers.
		AllowCredentials: true,
		// MaxAge caches the preflight response for 12 hours — reduces OPTIONS round-trips.
		MaxAge: 12 * time.Hour,
	}))

	// ── Proxy handlers ───────────────────────────────────────────────────────
	violationProxy := newProxy(services["violation"])
	billingProxy := newProxy(services["billing"])
	paymentProxy := newProxy(services["payment"])

	// ── Gateway health check ─────────────────────────────────────────────────
	// Fans out to all downstream /health endpoints concurrently and aggregates.
	r.GET("/health", func(c *gin.Context) {
		type result struct{ svc string; health map[string]string }
		ch := make(chan result, len(services))

		for name, url := range services {
			go func(n, u string) {
				ch <- result{svc: n, health: downstreamHealth(n, u)}
			}(name, url)
		}

		downstream := make([]map[string]string, 0, len(services))
		allOK := true
		for range services {
			r := <-ch
			downstream = append(downstream, r.health)
			if r.health["status"] != "ok" {
				allOK = false
			}
		}

		status := "ok"
		code := http.StatusOK
		if !allOK {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		c.JSON(code, gin.H{
			"gateway":    status,
			"downstream": downstream,
		})
	})

	// ── Routes ───────────────────────────────────────────────────────────────
	//
	// Each block registers two Gin routes per prefix:
	//   • "/prefix"          — matches the bare path (no trailing slash)
	//   • "/prefix/*action"  — matches any sub-path (Gin wildcard, requires ≥ one char)
	//
	// r.Any() forwards ALL HTTP methods (GET, POST, PUT, PATCH, DELETE, OPTIONS)
	// so new endpoints added to downstream services are automatically proxied
	// without changing the gateway.
	//
	// Routing table (matches assignment spec exactly):
	// ┌─────────────────────────┬───────────────────────────────────┐
	// │ Path prefix             │ Downstream service                │
	// ├─────────────────────────┼───────────────────────────────────┤
	// │ /api/v1/violations*     │ Violation Service  :8081          │
	// │ /api/v1/invoices*       │ Billing Service    :8082          │
	// │ /api/v1/fine-rules*     │ Billing Service    :8082          │
	// │ /api/v1/payments*       │ Payment Service    :8083          │
	// │ /api/v1/vehicles*       │ Payment Service    :8083          │
	// │ /internal/*             │ Billing Service    :8082          │
	// └─────────────────────────┴───────────────────────────────────┘

	v1 := r.Group("/api/v1")
	{
		// ── /api/v1/violations* → Violation Service :8081 ────────────────────
		v1.Any("/violations", violationProxy)
		v1.Any("/violations/*action", violationProxy)

		// ── /api/v1/invoices* → Billing Service :8082 ────────────────────────
		v1.Any("/invoices", billingProxy)
		v1.Any("/invoices/*action", billingProxy)

		// ── /api/v1/fine-rules* → Billing Service :8082 ──────────────────────
		v1.Any("/fine-rules", billingProxy)
		v1.Any("/fine-rules/*action", billingProxy)

		// ── /api/v1/payments* → Payment Service :8083 ────────────────────────
		v1.Any("/payments", paymentProxy)
		v1.Any("/payments/*action", paymentProxy)

		// ── /api/v1/vehicles* → Payment Service :8083 ────────────────────────
		v1.Any("/vehicles", paymentProxy)
		v1.Any("/vehicles/*action", paymentProxy)
	}

	// Internal routes — violation service calls billing's /internal/invoices
	// to trigger invoice generation. Exposing it at the gateway allows services
	// to use a single base URL in containerised environments.
	internal := r.Group("/internal")
	{
		internal.Any("/invoices", billingProxy)
		internal.Any("/invoices/*action", billingProxy)
	}

	// ── Start server ─────────────────────────────────────────────────────────
	port := getEnv("PORT", "8080")
	log.Printf("[api-gateway] listening on :%s", port)
	log.Printf("[api-gateway] routing:")
	log.Printf("  violation → %s", services["violation"])
	log.Printf("  billing   → %s", services["billing"])
	log.Printf("  payment   → %s", services["payment"])

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[api-gateway] server error: %v", err)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
