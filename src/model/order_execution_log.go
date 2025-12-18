// model/order_execution_log.go
package model

import "time"

// OrderExecutionStatus constants represent the lifecycle of an order execution.
// You can adjust these values to fit exactly your domain.
const (
	OrderExecutionStatusPending = "pending"
	//OrderExecutionStatusSent       = "sent"
	//OrderExecutionStatusAccepted   = "accepted"
	//OrderExecutionStatusRejected   = "rejected"
	//OrderExecutionStatusPartFilled = "part_filled"
	OrderExecutionStatusFilled        = "filled"
	OrderExecutionStatusCanceled      = "canceled"
	OrderExecutionStatusError         = "error"
	OrderExecutionStatusCanceledError = "canceled_error"
)

// OrderExecutionLog stores the detailed history of each interaction with the exchange
// and the final conclusion of the order execution.
type OrderExecutionLog struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// Foreign key to Order
	OrderID uint   `gorm:"index" json:"order_id"`
	Order   *Order `gorm:"constraint:OnDelete:CASCADE" json:"order,omitempty"`

	// Snapshot of the order at the moment of this log entry
	Symbol    string   `gorm:"size:100" json:"symbol"`
	Side      string   `gorm:"size:20" json:"side"`
	OrderType string   `gorm:"size:50" json:"order_type"`
	Quantity  float64  `json:"quantity"`
	Price     *float64 `json:"price,omitempty"`

	// Exchange-specific identifiers
	ExchangeID        uint   `gorm:"index" json:"exchange_id"`
	ExchangeOrderID   string `gorm:"size:255" json:"exchange_order_id"`  // ID returned by the exchange
	ExchangeClientID  string `gorm:"size:255" json:"exchange_client_id"` // clientOrderId or similar
	ExternalReference string `gorm:"size:255" json:"external_reference"` // any extra tracking code (optional)

	// Execution / conclusion details
	Status       string     `gorm:"size:50;not null" json:"status"` // see OrderExecutionStatus* constants
	Reason       string     `gorm:"size:255" json:"reason"`         // human-readable reason (e.g. "filled", "canceled by user", "rejected by exchange")
	ErrorMessage *string    `json:"error_message,omitempty"`        // error from exchange or internal error
	RequestedAt  time.Time  `json:"requested_at"`                   // when we sent the order to the exchange
	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`          // when the exchange accepted the order
	ExecutedAt   *time.Time `json:"executed_at,omitempty"`          // when the first fill happened
	CompletedAt  *time.Time `json:"completed_at,omitempty"`         // when the order reached a final state
	CreatedAt    time.Time  `json:"created_at"`                     // log creation
	UpdatedAt    time.Time  `json:"updated_at"`                     // last update of this log entry
}

// TableName allows you to control the exact table name for execution logs.
func (OrderExecutionLog) TableName() string {
	return "order_execution_logs"
}

type OrderLog struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// Foreign key to Order
	OrderID uint   `gorm:"index" json:"order_id"`
	Order   *Order `gorm:"constraint:OnDelete:CASCADE" json:"order,omitempty"`
	// Snapshot of the order at the moment of this log entry
	Symbol        string   `gorm:"size:100" json:"symbol"`
	Side          string   `gorm:"size:20" json:"side"`
	PosSide       string   `json:"pos_side"`
	OrderType     string   `gorm:"size:50" json:"order_type"`
	Quantity      float64  `json:"quantity"`
	StopLossPct   float64  `json:"stop_loss_pct"`
	TakeProfitPct float64  `json:"take_profit_pct"`
	Price         *float64 `json:"price,omitempty"`

	// Exchange-specific identifiers
	ExchangeID uint `gorm:"index" json:"exchange_id"`
	// Execution / conclusion details
	Status    string    `gorm:"size:50;not null" json:"status"` // see OrderExecutionStatus* constants
	CreatedAt time.Time `json:"created_at"`                     // log creation
}

// TableName allows you to control the exact table name for orders.
func (OrderLog) TableName() string {
	return "order_logs"
}
