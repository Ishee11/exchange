package model

type BidRequest struct {
	RequestID   string  `json:"request_id"`
	ImpID       string  `json:"imp_id"`
	SiteID      string  `json:"site_id"`
	PlacementID string  `json:"placement_id"`
	FloorPrice  float64 `json:"floor_price"`
	UserID      string  `json:"user_id"`
	DeviceType  string  `json:"device_type"`
	Timestamp   int64   `json:"ts"`
}
