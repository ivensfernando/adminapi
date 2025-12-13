package model

import "time"

//type Alerts struct {
//	ID        uint   `gorm="primaryKey"`
//	Symbol    string `gorm:"not null"`
//	Action    string `gorm:"not null"`
//	Message   string
//	CreatedAt time.Time
//}

type Alert struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	AlertName    *string    `gorm:"column:alert_name"`    // nullable
	Body         *string    `gorm:"type:jsonb"`           // jsonb, nullable
	ReceivedAt   *time.Time `gorm:"column:received_at"`   // nullable
	Event        *string    `gorm:"column:event"`         // nullable
	Description  *string    `gorm:"column:description"`   // nullable
	Symbol       *string    `gorm:"column:symbol"`        // nullable
	Exchange     *string    `gorm:"column:exchange"`      // nullable
	Interval     *string    `gorm:"column:interval"`      // nullable
	Open         *float64   `gorm:"column:open"`          // numeric
	Close        *float64   `gorm:"column:close"`         // numeric
	High         *float64   `gorm:"column:high"`          // numeric
	Low          *float64   `gorm:"column:low"`           // numeric
	Volume       *float64   `gorm:"column:volume"`        // numeric
	Currency     *string    `gorm:"column:currency"`      // nullable
	BaseCurrency *string    `gorm:"column:base_currency"` // nullable
	Plot         *string    `gorm:"column:plot"`          // nullable
	AlertTime    *time.Time `gorm:"column:alert_time"`    // nullable
	ServerTime   *time.Time `gorm:"column:server_time"`   // nullable
	Action       *string    `gorm:"column:action"`        // needs to be added in DB too
	CreatedAt    time.Time
}
