package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ishee11/exchange/internal/model"
)

func TestClient_Send(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		timeout    time.Duration
		wantDelay  time.Duration // максимальное время выполнения
		wantStatus SendStatus
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			timeout:    100 * time.Millisecond,
			wantDelay:  100 * time.Millisecond,
			wantStatus: StatusSuccess,
		},
		{
			name: "timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			timeout:    50 * time.Millisecond,
			wantDelay:  150 * time.Millisecond,
			wantStatus: StatusTimeout,
		},
		{
			name: "bad response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
			},
			timeout:    100 * time.Millisecond,
			wantDelay:  100 * time.Millisecond,
			wantStatus: StatusBadResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			c := New(server.URL)

			req := model.BidRequest{
				RequestID: "1",
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			start := time.Now()

			result := c.Send(ctx, req)

			elapsed := time.Since(start)

			if elapsed > tt.wantDelay {
				t.Fatalf("too slow: %v", elapsed)
			}

			if result.Status != tt.wantStatus {
				t.Fatalf("unexpected status: got %q want %q", result.Status, tt.wantStatus)
			}
		})
	}
}

func TestClient_Send_ConnectionError(t *testing.T) {
	c := New("http://localhost:9999") // ничего не слушает

	req := model.BidRequest{
		RequestID: "1",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result := c.Send(ctx, req)

	if result.Status != StatusRequestError {
		t.Fatalf("unexpected status: got %q want %q", result.Status, StatusRequestError)
	}
}
