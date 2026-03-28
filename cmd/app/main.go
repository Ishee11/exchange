package main

import (
	"context"
	"math/rand"
	"os"
	"time"

	"github.com/Ishee11/exchange/internal/client"
	"github.com/Ishee11/exchange/internal/generator"
)

type BidRequest struct {
	RequestID   string `json:"request_id"`   // идемпотентность
	ImpID       string `json:"imp_id"`       // конкретный показ
	SiteID      string `json:"site_id"`      // площадка
	PlacementID string `json:"placement_id"` // слот

	FloorPrice float64 `json:"floor_price"`

	UserID     string `json:"user_id"`
	DeviceType string `json:"device_type"`

	Timestamp int64 `json:"ts"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	url := os.Getenv("DSP_URL")
	if url == "" {
		url = "http://localhost:8080/bid"
	}

	c := client.New(url)

	gen := generator.New(
		10,
		100*time.Millisecond, // ← SLA
		c,
	)

	gen.Start(context.Background())
}
