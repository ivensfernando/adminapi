package model

import "time"

// PhemexOrder represents a normalized Phemex order response stored in the database.
type PhemexOrder struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// Identifiers
	// 🔑 Foreign key to internal Order
	OrderID uint  `gorm:"index;not null" json:"order_id"`
	Order   Order `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	// Exchange identifiers
	ExchangeOrderID string `gorm:"size:100;uniqueIndex" json:"exchange_order_id"`
	ClOrdID         string `gorm:"size:100" json:"cl_ord_id"`
	Symbol          string `gorm:"size:50;index" json:"symbol"`
	Side            string `gorm:"size:10" json:"side"`

	// Timestamps (nanoseconds -> time)
	ActionTime   time.Time `json:"action_time"`
	TransactTime time.Time `json:"transact_time"`

	// Order info
	OrderType   string  `gorm:"size:30" json:"order_type"`
	Price       float64 `json:"price"`
	OrderQty    float64 `json:"order_qty"`
	DisplayQty  float64 `json:"display_qty"`
	TimeInForce string  `gorm:"size:50" json:"time_in_force"`

	// PnL / position
	ClosedPnl  float64 `json:"closed_pnl"`
	ClosedSize float64 `json:"closed_size"`

	// Cumulative
	CumQty   float64 `json:"cum_qty"`
	CumValue float64 `json:"cum_value"`

	// Leaves
	LeavesQty   float64 `json:"leaves_qty"`
	LeavesValue float64 `json:"leaves_value"`

	// Stops
	StopDirection string  `gorm:"size:30" json:"stop_direction"`
	StopPrice     float64 `json:"stop_price"`
	Trigger       string  `gorm:"size:30" json:"trigger"`

	// Peg
	PegOffsetValue      float64 `json:"peg_offset_value"`
	PegOffsetProportion float64 `json:"peg_offset_proportion"`
	PegPriceType        string  `gorm:"size:30" json:"peg_price_type"`

	// Execution state
	ExecStatus string `gorm:"size:50" json:"exec_status"`
	OrdStatus  string `gorm:"size:50" json:"ord_status"`
	ExecInst   string `gorm:"size:50" json:"exec_inst"`

	// TP / SL
	TakeProfit float64 `json:"take_profit"`
	StopLoss   float64 `json:"stop_loss"`
	SlPrice    float64 `json:"sl_price"`
	TpPrice    float64 `json:"tp_price"`

	CreatedAt time.Time `json:"created_at"`
}

type PhemexOrderResponse struct {
	BizError              int    `json:"bizError"`
	OrderID               string `json:"orderID"`
	ClOrdID               string `json:"clOrdID"`
	Symbol                string `json:"symbol"`
	Side                  string `json:"side"`
	ActionTimeNs          int64  `json:"actionTimeNs"`
	TransactTimeNs        int64  `json:"transactTimeNs"`
	OrderType             string `json:"orderType"`
	PriceRp               string `json:"priceRp"`
	OrderQtyRq            string `json:"orderQtyRq"`
	DisplayQtyRq          string `json:"displayQtyRq"`
	TimeInForce           string `json:"timeInForce"`
	ClosedPnlRv           string `json:"closedPnlRv"`
	ClosedSizeRq          string `json:"closedSizeRq"`
	CumQtyRq              string `json:"cumQtyRq"`
	CumValueRv            string `json:"cumValueRv"`
	LeavesQtyRq           string `json:"leavesQtyRq"`
	LeavesValueRv         string `json:"leavesValueRv"`
	StopDirection         string `json:"stopDirection"`
	StopPxRp              string `json:"stopPxRp"`
	Trigger               string `json:"trigger"`
	PegOffsetValueRp      string `json:"pegOffsetValueRp"`
	PegOffsetProportionRr string `json:"pegOffsetProportionRr"`
	ExecStatus            string `json:"execStatus"`
	PegPriceType          string `json:"pegPriceType"`
	OrdStatus             string `json:"ordStatus"`
	ExecInst              string `json:"execInst"`
	TakeProfitRp          string `json:"takeProfitRp"`
	StopLossRp            string `json:"stopLossRp"`
	SlPxRp                string `json:"slPxRp"`
	TpPxRp                string `json:"tpPxRp"`
}
