package client

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Ishee11/exchange/internal/model"
)

type Client struct {
	url    string
	client *http.Client
}

func New(url string) *Client {
	return &Client{
		url: url,
		client: &http.Client{
			Timeout: 0, // используем context timeout
		},
	}
}

func (c *Client) Send(ctx context.Context, req model.BidRequest) {
	data, err := json.Marshal(req)
	if err != nil {
		log.Println("marshal error:", err)
		return
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.url,
		bytes.NewBuffer(data),
	)
	if err != nil {
		log.Println("request build error:", err)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()

	resp, err := c.client.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		log.Printf("request failed: err=%v latency=%v\n", err, latency)
		return
	}
	defer resp.Body.Close()

	log.Printf("response: status=%d latency=%v\n", resp.StatusCode, latency)
}
