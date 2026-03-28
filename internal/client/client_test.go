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
		name      string
		handler   http.HandlerFunc
		timeout   time.Duration
		wantDelay time.Duration // максимальное время выполнения
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			timeout:   100 * time.Millisecond,
			wantDelay: 100 * time.Millisecond,
		},
		{
			name: "timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			timeout:   50 * time.Millisecond,
			wantDelay: 150 * time.Millisecond,
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

			c.Send(ctx, req)

			elapsed := time.Since(start)

			if elapsed > tt.wantDelay {
				t.Fatalf("too slow: %v", elapsed)
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

	c.Send(ctx, req)
}
