package model

import "time"

type UserExchange struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"not null;index:idx_user_exchange,unique" json:"user_id"`
	ExchangeID        uint      `gorm:"not null;index:idx_user_exchange,unique" json:"exchange_id"`
	APIKeyHash        string    `gorm:"column:api_key;type:text" json:"-"`
	APISecretHash     string    `gorm:"column:api_secret;type:text" json:"-"`
	APIPassphraseHash string    `gorm:"column:api_passphrase;type:text" json:"-"`
	ShowInForms       bool      `gorm:"not null;default:false" json:"show_in_forms"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	User     *User     `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Exchange *Exchange `gorm:"constraint:OnDelete:CASCADE" json:"exchange"`
}

type UpsertUserExchangePayload struct {
	ExchangeID    uint   `json:"exchangeId"`
	APIKey        string `json:"apiKey"`
	APISecret     string `json:"apiSecret"`
	APIPassphrase string `json:"apiPassphrase"`
	ShowInForms   bool   `json:"showInForms"`
}

type UserExchangeResponse struct {
	ID               uint   `json:"id"`
	ExchangeID       uint   `json:"exchangeId"`
	ExchangeName     string `json:"exchangeName,omitempty"`
	ShowInForms      bool   `json:"showInForms"`
	HasAPIKey        bool   `json:"hasApiKey"`
	HasAPISecret     bool   `json:"hasApiSecret"`
	HasAPIPassphrase bool   `json:"hasApiPassphrase"`
}

func NewUserExchangeResponse(ue *UserExchange) UserExchangeResponse {
	if ue == nil {
		return UserExchangeResponse{}
	}

	resp := UserExchangeResponse{
		ID:               ue.ID,
		ExchangeID:       ue.ExchangeID,
		ShowInForms:      ue.ShowInForms,
		HasAPIKey:        ue.APIKeyHash != "",
		HasAPISecret:     ue.APISecretHash != "",
		HasAPIPassphrase: ue.APIPassphraseHash != "",
	}

	if ue.Exchange != nil {
		resp.ExchangeName = ue.Exchange.Name
	}

	return resp
}
