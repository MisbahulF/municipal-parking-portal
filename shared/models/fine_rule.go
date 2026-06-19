package models

import (
	"time"

	"gorm.io/gorm"
)

// FineRule represents a versioned ruleset snapshot.
// Only one version may be active at a time (enforced at the service layer).
// A new version must be created rather than mutating an existing one,
// ensuring full audit-trail integrity.
//
// Table: fine_rules
type FineRule struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"              json:"id"`
	Version   int            `gorm:"not null;uniqueIndex"                  json:"version"`
	IsActive  bool           `gorm:"not null;default:false"                json:"is_active"`
	CreatedAt time.Time      `gorm:"autoCreateTime"                        json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                 json:"-"`

	// Details contains the individual rules belonging to this version.
	Details []FineRuleDetail `gorm:"foreignKey:FineRuleID;constraint:OnDelete:CASCADE" json:"details,omitempty"`
}

// FineRuleDetail stores the fine configuration for a specific violation type
// within a given FineRule version. Multiple detail rows can belong to the
// same FineRule, one per ViolationType.
//
// Table: fine_rule_details
type FineRuleDetail struct {
	ID            uint          `gorm:"primaryKey;autoIncrement"              json:"id"`
	FineRuleID    uint          `gorm:"not null;index"                        json:"fine_rule_id"`
	ViolationType ViolationType `gorm:"type:varchar(50);not null"             json:"violation_type"`

	// BaseAmount is the starting fine in the smallest currency unit (e.g. IDR cents or whole IDR).
	BaseAmount float64 `gorm:"type:numeric(15,2);not null"           json:"base_amount"`

	// TimeWindow is the interval in minutes used by PROGRESSIVE logic
	// to determine how many multiplier steps have elapsed.
	// Ignored when LogicType is FLAT.
	TimeWindow int `gorm:"default:0"                            json:"time_window"`

	// Multiplier is applied per time_window elapsed for PROGRESSIVE logic.
	// Example: base=50_000, multiplier=1.5, window=30 min →
	//   after 60 min: fine = 50_000 × 1.5² = 112_500.
	Multiplier float64 `gorm:"type:numeric(5,2);default:1.0"        json:"multiplier"`

	LogicType LogicType `gorm:"type:varchar(20);not null;default:'FLAT'" json:"logic_type"`

	FineRule FineRule `gorm:"foreignKey:FineRuleID" json:"-"`
}
