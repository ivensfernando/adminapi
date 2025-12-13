package payloads

type CreateWebhookPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Tickers     string `json:"tickers"`
	Active      *bool  `json:"active,omitempty"`
}

type UpdateWebhookPayload struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Type        *string `json:"type,omitempty"`
	Tickers     *string `json:"tickers,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}
