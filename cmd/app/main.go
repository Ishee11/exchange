package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/Ishee11/exchange/internal/client"
	"github.com/Ishee11/exchange/internal/generator"
	"github.com/Ishee11/exchange/internal/metrics"
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

	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":2112"
	}

	go serveMetrics(metricsAddr)

	c := client.New(url)

	gen := generator.New(
		10,
		100*time.Millisecond, // ← SLA
		c,
	)

	gen.Start(context.Background())
}

func serveMetrics(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())

	log.Printf("metrics server started: addr=%s path=/metrics\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("metrics server failed: %v", err)
	}
}
