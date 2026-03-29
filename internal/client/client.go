package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Ishee11/exchange/internal/model"
)

type Client struct {
	url    string
	client *http.Client
}

type SendStatus string

const (
	StatusSuccess      SendStatus = "success"
	StatusTimeout      SendStatus = "timeout"
	StatusRequestError SendStatus = "request_error"
	StatusBadResponse  SendStatus = "bad_response"
	StatusMarshalError SendStatus = "marshal_error"
	StatusBuildError   SendStatus = "build_error"
)

type SendResult struct {
	Status     SendStatus
	HTTPStatus int
	Latency    time.Duration
	Err        error
}

func New(url string) *Client {
	return &Client{
		url: url,
		client: &http.Client{
			Timeout: 0, // используем context timeout
		},
	}
}

func (c *Client) Send(ctx context.Context, req model.BidRequest) SendResult {
	data, err := json.Marshal(req)
	if err != nil {
		log.Println("marshal error:", err)
		return SendResult{
			Status: StatusMarshalError,
			Err:    err,
		}
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.url,
		bytes.NewBuffer(data),
	)
	if err != nil {
		log.Println("request build error:", err)
		return SendResult{
			Status: StatusBuildError,
			Err:    err,
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()

	resp, err := c.client.Do(httpReq)
	latency := time.Since(start)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// timeout → DSP не уложился в SLA
			log.Printf("timeout: latency=%v\n", latency)
			return SendResult{
				Status:  StatusTimeout,
				Latency: latency,
				Err:     err,
			}
		}

		log.Printf("request error: %v latency=%v\n", err, latency)
		return SendResult{
			Status:  StatusRequestError,
			Latency: latency,
			Err:     err,
		}
	}
	defer resp.Body.Close()

	result := SendResult{
		HTTPStatus: resp.StatusCode,
		Latency:    latency,
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		result.Status = StatusSuccess
		log.Printf("response: status=%s http_status=%d latency=%v\n", result.Status, result.HTTPStatus, result.Latency)
		return result
	}

	result.Status = StatusBadResponse
	log.Printf("response: status=%s http_status=%d latency=%v\n", result.Status, result.HTTPStatus, result.Latency)
	return result
}
