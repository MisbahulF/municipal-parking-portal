package models

import (
	"time"

	"gorm.io/gorm"
)

// Violation represents a single parking infraction reported by an officer
// or automated system. It captures the raw event and is immutable after
// creation — corrections must be made via a new record.
//
// Table: violations
type Violation struct {
	ID            uint           `gorm:"primaryKey;autoIncrement"         json:"id"`
	LicensePlate  string         `gorm:"type:varchar(20);not null;index"  json:"license_plate"`
	ViolationType ViolationType  `gorm:"type:varchar(50);not null"        json:"violation_type"`
	Location      string         `gorm:"type:text;not null"               json:"location"`

	// Timestamp is when the violation was observed, not when it was recorded.
	Timestamp time.Time `gorm:"not null"                         json:"timestamp"`

	// PhotoURL is an optional link to evidence stored in object storage.
	PhotoURL  string         `gorm:"type:text"                        json:"photo_url,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime"                   json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"                   json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                            json:"-"`

	// Invoice is the billing record generated from this violation.
	Invoice *Invoice `gorm:"foreignKey:ViolationID"           json:"invoice,omitempty"`
}
