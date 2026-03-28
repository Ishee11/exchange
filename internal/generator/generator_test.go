package generator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Ishee11/exchange/internal/model"
)

type mockSender struct {
	mu    sync.Mutex
	calls int
}

func (m *mockSender) Send(ctx context.Context, req model.BidRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
}

func (m *mockSender) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestGenerator_SendsRequests(t *testing.T) {
	sender := &mockSender{}

	gen := New(
		10, // 10 RPS
		100*time.Millisecond,
		sender,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go gen.Start(ctx)

	time.Sleep(250 * time.Millisecond)

	count := sender.Count()

	if count == 0 {
		t.Fatal("expected at least one request, got 0")
	}
}
