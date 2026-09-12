package controller

// PublicGroupSyncSnapshotVersion is the wire version for public group snapshots.
const PublicGroupSyncSnapshotVersion = 1

// PublicGroupSyncRequest carries a public group's configuration between services.
type PublicGroupSyncRequest struct {
	EventID       string                          `json:"event_id"`
	Version       int                             `json:"version"`
	Revision      int64                           `json:"revision"`
	GroupID       int64                           `json:"group_id"`
	GroupName     string                          `json:"group_name"`
	PublicEnabled bool                            `json:"public_enabled"`
	GroupRatio    float64                         `json:"group_ratio"`
	BillingMode   string                          `json:"billing_mode"`
	Models        []string                        `json:"models"`
	ModelMapping  map[string]string               `json:"model_mapping"`
	ModelPricing  map[string]PublicGroupSyncModel `json:"model_pricing"`
}

// PublicGroupSyncModel contains the public display name, upstream mapping and
// prices needed to reproduce a group's billing behavior.
type PublicGroupSyncModel struct {
	Platform        string   `json:"platform"`
	DisplayName     string   `json:"display_name"`
	UpstreamModel   string   `json:"upstream_model"`
	BillingMode     string   `json:"billing_mode"`
	InputPrice      *float64 `json:"input_price,omitempty"`
	OutputPrice     *float64 `json:"output_price,omitempty"`
	PerRequestPrice *float64 `json:"per_request_price,omitempty"`
	CacheWritePrice *float64 `json:"cache_write_price,omitempty"`
	CacheReadPrice  *float64 `json:"cache_read_price,omitempty"`
}
