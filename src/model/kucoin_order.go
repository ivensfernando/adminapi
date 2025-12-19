package model

//import "time"
//
//// KucoinOrder stores normalized KuCoin order details linked to an internal Order record.
//type KucoinOrder struct {
//	ID uint `gorm:"primaryKey" json:"id"`
//
//	// Foreign key to the generic Order table
//	OrderID uint  `gorm:"index;not null" json:"order_id"`
//	Order   Order `gorm:"constraint:OnDelete:CASCADE" json:"-"`
//
//	// Exchange identifiers
//	ExchangeOrderID string `gorm:"size:100;uniqueIndex" json:"exchange_order_id"`
//	ClientOid       string `gorm:"size:100" json:"client_oid"`
//	Symbol          string `gorm:"size:50;index" json:"symbol"`
//	Side            string `gorm:"size:10" json:"side"`
//
//	// Order details
//	OrderType   string  `gorm:"size:30" json:"order_type"`
//	Status      string  `gorm:"size:30" json:"status"`
//	Price       float64 `json:"price"`
//	Size        float64 `json:"size"`
//	FilledSize  float64 `json:"filled_size"`
//	FilledValue float64 `json:"filled_value"`
//	Leverage    float64 `json:"leverage"`
//	Fee         float64 `json:"fee"`
//	FeeCurrency string  `gorm:"size:20" json:"fee_currency"`
//	TimeInForce string  `gorm:"size:20" json:"time_in_force"`
//	Remark      string  `gorm:"size:255" json:"remark"`
//
//	// Timestamps
//	OrderTime time.Time `json:"order_time"`
//	CreatedAt time.Time `json:"created_at"`
//}
//
//// KucoinOrderResponse represents the main fields returned when placing an order on KuCoin.
//type KucoinOrderResponse struct {
//	OrderID     string `json:"orderId"`
//	ClientOid   string `json:"clientOid"`
//	Symbol      string `json:"symbol,omitempty"`
//	Type        string `json:"type,omitempty"`
//	Side        string `json:"side,omitempty"`
//	Price       string `json:"price,omitempty"`
//	Size        string `json:"size,omitempty"`
//	DealSize    string `json:"dealSize,omitempty"`
//	DealValue   string `json:"dealValue,omitempty"`
//	Leverage    string `json:"leverage,omitempty"`
//	Fee         string `json:"fee,omitempty"`
//	FeeCurrency string `json:"feeCurrency,omitempty"`
//	TimeInForce string `json:"timeInForce,omitempty"`
//	Remark      string `json:"remark,omitempty"`
//	OrderTime   int64  `json:"orderTime,omitempty"`
//	Status      string `json:"status,omitempty"`
//}
