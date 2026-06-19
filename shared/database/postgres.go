package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dhill/parking-violation-portal/shared/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global GORM database instance shared across the service.
var DB *gorm.DB

// ─── Day-One Fine Formula Constants ───────────────────────────────────────────
//
// Time-window multipliers (applied based on hour of violation observation).
// The billing service must call TimeWindowMultiplier(hour) to retrieve the
// correct factor before computing the final fine.
const (
	DaytimeStartHour = 6  // 06:00 inclusive
	DaytimeEndHour   = 22 // 22:00 exclusive

	MultiplierDaytime float64 = 1.0 // 06:00–21:59
	MultiplierNight   float64 = 1.5 // 22:00–05:59
)

// Repeat-offender multipliers (applied based on the vehicle's count of
// currently UNPAID invoices at the time a new invoice is generated).
const (
	MultiplierRepeat0 float64 = 1.0 // 0 unpaid invoices
	MultiplierRepeat1 float64 = 1.5 // exactly 1 unpaid invoice
	MultiplierRepeat2 float64 = 2.0 // 2 or more unpaid invoices
)

// TimeWindowMultiplier returns the correct multiplier for the given hour (0–23).
// Used by the billing service during fine calculation.
func TimeWindowMultiplier(hour int) float64 {
	if hour >= DaytimeStartHour && hour < DaytimeEndHour {
		return MultiplierDaytime
	}
	return MultiplierNight
}

// RepeatOffenderMultiplier returns the correct multiplier for the given count
// of currently UNPAID invoices associated with a license plate.
func RepeatOffenderMultiplier(unpaidCount int) float64 {
	switch {
	case unpaidCount >= 2:
		return MultiplierRepeat2
	case unpaidCount == 1:
		return MultiplierRepeat1
	default:
		return MultiplierRepeat0
	}
}

// ─── Config ───────────────────────────────────────────────────────────────────

// Config holds the PostgreSQL connection parameters
// sourced from environment variables.
type Config struct {
	Host     string
	User     string
	Password string
	Name     string
	Port     string
}

// configFromEnv reads database connection settings from environment variables.
// Defaults are provided for local development.
func configFromEnv() Config {
	return Config{
		Host:     getEnv("DB_HOST", "localhost"),
		User:     getEnv("DB_USER", "pvp_user"),
		Password: getEnv("DB_PASSWORD", "pvp_password"),
		Name:     getEnv("DB_NAME", "pvp_db"),
		Port:     getEnv("DB_PORT", "5432"),
	}
}

// DSN builds the PostgreSQL Data Source Name string from the config.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		c.Host, c.User, c.Password, c.Name, c.Port,
	)
}

// ─── Init ─────────────────────────────────────────────────────────────────────

// InitPostgres initializes the GORM connection pool and assigns it to DB.
// It configures connection pool settings suitable for production use and
// returns an error if the connection cannot be established.
func InitPostgres() error {
	cfg := configFromEnv()

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure the underlying sql.DB connection pool.
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)

	DB = db
	log.Printf("[database] connected to PostgreSQL at %s:%s/%s\n", cfg.Host, cfg.Port, cfg.Name)
	return nil
}

// GetDB returns the active GORM DB instance.
// Panics if InitPostgres has not been called first.
func GetDB() *gorm.DB {
	if DB == nil {
		panic("database: DB is nil — call InitPostgres() before GetDB()")
	}
	return DB
}

// ─── Migration ────────────────────────────────────────────────────────────────

// AutoMigrate runs GORM schema migrations for all domain models in dependency
// order (parent tables before child tables to satisfy FK constraints).
func AutoMigrate() error {
	log.Println("[database] running auto-migrations...")
	if err := DB.AutoMigrate(
		&models.FineRule{},          // no FK dependencies
		&models.FineRuleDetail{},    // FK → fine_rules
		&models.Violation{},         // no FK dependencies
		&models.Invoice{},           // FK → violations, fine_rules
		&models.PaymentAttempt{},    // FK → invoices
	); err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}
	log.Println("[database] auto-migrations complete")
	return nil
}

// ─── Seed ─────────────────────────────────────────────────────────────────────

// dayOneFineDetails defines the base fine amounts for each violation type
// in the initial "Day One" rule version.
//
// Final fine = base_amount × TimeWindowMultiplier(hour) × RepeatOffenderMultiplier(unpaidCount)
//
// Examples:
//   - EXPIRED_METER at 14:00, 0 unpaid  → 50,000 × 1.0 × 1.0 =  50,000
//   - EXPIRED_METER at 23:00, 1 unpaid  → 50,000 × 1.5 × 1.5 = 112,500
//   - DISABLED_SPOT at 02:00, 2+ unpaid → 500,000 × 1.5 × 2.0 = 1,500,000
var dayOneFineDetails = []models.FineRuleDetail{
	{
		ViolationType: models.ViolationExpiredMeter,
		BaseAmount:    50_000,
		LogicType:     models.LogicFlat,
		Multiplier:    1.0,
		TimeWindow:    0,
	},
	{
		ViolationType: models.ViolationNoParkingZone,
		BaseAmount:    150_000,
		LogicType:     models.LogicFlat,
		Multiplier:    1.0,
		TimeWindow:    0,
	},
	{
		ViolationType: models.ViolationBlockingHydrant,
		BaseAmount:    250_000,
		LogicType:     models.LogicFlat,
		Multiplier:    1.0,
		TimeWindow:    0,
	},
	{
		ViolationType: models.ViolationDisabledSpot,
		BaseAmount:    500_000,
		LogicType:     models.LogicFlat,
		Multiplier:    1.0,
		TimeWindow:    0,
	},
}

// SeedDayOneFormula inserts the initial "Day One" FineRule (Version 1, active)
// with its four violation-type details if no FineRule records exist yet.
//
// This function is idempotent — safe to call on every service startup.
// All writes are wrapped in a single transaction; on failure, nothing is committed.
func SeedDayOneFormula() error {
	var count int64
	if err := DB.Model(&models.FineRule{}).Count(&count).Error; err != nil {
		return fmt.Errorf("[seed] count check failed: %w", err)
	}
	if count > 0 {
		log.Println("[database] seed skipped — FineRule data already exists")
		return nil
	}

	log.Println("[database] seeding Day-One fine formula...")

	return DB.Transaction(func(tx *gorm.DB) error {
		// 1. Create the versioned ruleset header.
		rule := models.FineRule{
			Version:  1,
			IsActive: true,
		}
		if err := tx.Create(&rule).Error; err != nil {
			return fmt.Errorf("[seed] failed to create FineRule: %w", err)
		}

		// 2. Attach each violation-type detail to the rule.
		for i := range dayOneFineDetails {
			dayOneFineDetails[i].FineRuleID = rule.ID
		}
		if err := tx.Create(&dayOneFineDetails).Error; err != nil {
			return fmt.Errorf("[seed] failed to create FineRuleDetails: %w", err)
		}

		log.Printf(
			"[database] seed complete — FineRule v%d (id=%d) with %d violation types\n",
			rule.Version, rule.ID, len(dayOneFineDetails),
		)
		return nil
	})
}

// MigrateAndSeed is a convenience helper that runs AutoMigrate followed by
// SeedDayOneFormula. Call this once during service startup.
func MigrateAndSeed() error {
	if err := AutoMigrate(); err != nil {
		return err
	}
	return SeedDayOneFormula()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// getEnv returns the value of the environment variable key,
// falling back to fallback if the variable is not set.
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

