package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

type BidRequest struct {
	UserID      string  `json:"user_id"`
	PlacementID string  `json:"placement_id"`
	FloorPrice  float64 `json:"floor_price"`
}

func main() {
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
