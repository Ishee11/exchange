package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"
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
	reqBody := BidRequest{
		UserID:      "123",
		PlacementID: "banner_top",
		FloorPrice:  1.0,
	}

	data, _ := json.Marshal(reqBody)

	resp, err := http.Post("http://localhost:8080/bid", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	log.Printf("response: %+v\n", result)
}
