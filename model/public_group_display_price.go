package model

// PublicGroupDisplayPrice is an effective USD quote for the model plaza only.
// Token fields are USD/token; fixed prices are USD/request (or image/video).
// It is deliberately separate from ratio_setting and never used by billing.
type PublicGroupDisplayPrice struct {
	BillingMode      string   `json:"billing_mode"`
	InputPrice       *float64 `json:"input_price,omitempty"`
	OutputPrice      *float64 `json:"output_price,omitempty"`
	ImageInputPrice  *float64 `json:"image_input_price,omitempty"`
	ImageOutputPrice *float64 `json:"image_output_price,omitempty"`
	PerRequestPrice  *float64 `json:"per_request_price,omitempty"`
	CacheReadPrice   *float64 `json:"cache_read_price,omitempty"`
	CacheWritePrice  *float64 `json:"cache_write_price,omitempty"`
}
