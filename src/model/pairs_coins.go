package model

type PairsCoins struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Coin1   string `gorm:"not null" json:"coin1"`   // e.g. BTC
	Coin2   string `gorm:"not null" json:"coin2"`   // e.g. USDT
	Display string `gorm:"not null" json:"display"` // e.g. BTC/USDT or BTC-USDT

	// Unique constraint for (coin1, coin2)
	// Note: GORM creates this automatically via tag below
}

func (PairsCoins) TableName() string {
	return "pairs_coins"
}
