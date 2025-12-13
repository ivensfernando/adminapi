package model

import "time"

// Exception represents a system-level error that must be persisted
// for auditing, debugging, and monitoring purposes.
type Exception struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// Where the error happened
	Service string `gorm:"size:100;index" json:"service"` // e.g. "strategy_executor"
	Module  string `gorm:"size:100;index" json:"module"`  // e.g. "phemex_client"
	Method  string `gorm:"size:100" json:"method"`        // e.g. "PlaceOrder"

	// Error information
	Message string `gorm:"type:text" json:"message"` // err.Error()
	Stack   string `gorm:"type:text" json:"stack"`   // stack trace (optional)

	// Severity level
	Level string `gorm:"size:20;index" json:"level"` // debug | info | warn | error | fatal

	// Extra context stored as JSON (optional)
	Context string `gorm:"type:jsonb" json:"context,omitempty"`

	// Audit info
	CreatedAt time.Time `json:"created_at"`
}
