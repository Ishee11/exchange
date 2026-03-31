package generator

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ishee11/exchange/internal/client"
	"github.com/Ishee11/exchange/internal/model"
)

type mockSender struct {
	mu    sync.Mutex
	calls int
}

func (m *mockSender) Send(ctx context.Context, req model.BidRequest) client.SendResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	return client.SendResult{}
}

func (m *mockSender) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestGenerator_SendsRequests(t *testing.T) {
	sender := &mockSender{}

	gen := New(
		Config{
			TargetRPS:        10,
			Timeout:          100 * time.Millisecond,
			ConcurrencyLimit: 2,
			PlateauDuration:  0,
		},
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

type slowSender struct {
	active int64
	max    int64
	delay  time.Duration
}

func (s *slowSender) Send(ctx context.Context, req model.BidRequest) client.SendResult {
	current := atomic.AddInt64(&s.active, 1)

	for {
		max := atomic.LoadInt64(&s.max)
		if current <= max {
			break
		}

		if atomic.CompareAndSwapInt64(&s.max, max, current) {
			break
		}
	}

	defer atomic.AddInt64(&s.active, -1)

	select {
	case <-ctx.Done():
		return client.SendResult{Status: client.StatusTimeout}
	case <-time.After(s.delay):
		return client.SendResult{Status: client.StatusSuccess}
	}
}

type timestampSender struct {
	mu    sync.Mutex
	times []time.Time
}

func (s *timestampSender) Send(ctx context.Context, req model.BidRequest) client.SendResult {
	s.mu.Lock()
	s.times = append(s.times, time.Now())
	s.mu.Unlock()
	return client.SendResult{Status: client.StatusSuccess}
}

func (s *timestampSender) CountBetween(start time.Time, from, to time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, ts := range s.times {
		elapsed := ts.Sub(start)
		if elapsed >= from && elapsed < to {
			count++
		}
	}

	return count
}

func TestGenerator_RespectsConcurrencyLimit(t *testing.T) {
	sender := &slowSender{delay: 40 * time.Millisecond}

	gen := New(
		Config{
			TargetRPS:        200,
			Timeout:          200 * time.Millisecond,
			ConcurrencyLimit: 2,
			Jitter:           0,
			PlateauDuration:  80 * time.Millisecond,
		},
		sender,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	gen.Start(ctx)

	if got := atomic.LoadInt64(&sender.max); got > 2 {
		t.Fatalf("concurrency limit exceeded: got %d want <= 2", got)
	}
}

func TestBuildRequest_ByClass(t *testing.T) {
	tests := []struct {
		name  string
		class RequestClass
		check func(t *testing.T, req model.BidRequest)
	}{
		{
			name:  "hot path",
			class: RequestClassHot,
			check: func(t *testing.T, req model.BidRequest) {
				if req.SiteID == "" || req.FloorPrice < 0 {
					t.Fatalf("hot request should be valid: %+v", req)
				}
			},
		},
		{
			name:  "expensive",
			class: RequestClassExpensive,
			check: func(t *testing.T, req model.BidRequest) {
				if req.FloorPrice < 2.5 {
					t.Fatalf("expensive request should have high floor: %+v", req)
				}
			},
		},
		{
			name:  "no bid prone",
			class: RequestClassNoBidProne,
			check: func(t *testing.T, req model.BidRequest) {
				if req.FloorPrice < 4 {
					t.Fatalf("no-bid-prone request should be hard to win: %+v", req)
				}
			},
		},
		{
			name:  "invalid",
			class: RequestClassInvalid,
			check: func(t *testing.T, req model.BidRequest) {
				if req.SiteID != "" || req.FloorPrice >= 0 || req.UserID != "" {
					t.Fatalf("invalid request should contain broken fields: %+v", req)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := buildRequest(tt.class)
			tt.check(t, req)
		})
	}
}

func TestGenerator_RunsScenarioStepsSequentially(t *testing.T) {
	sender := &timestampSender{}

	steps := []ScenarioStep{
		{
			Name:     "normal",
			Duration: 80 * time.Millisecond,
			Config: Config{
				TargetRPS:        20,
				Timeout:          50 * time.Millisecond,
				ConcurrencyLimit: 2,
			},
		},
		{
			Name:     "heavy",
			Duration: 80 * time.Millisecond,
			Config: Config{
				TargetRPS:        200,
				Timeout:          50 * time.Millisecond,
				ConcurrencyLimit: 2,
			},
		},
	}

	gen := NewWithScenario(
		Config{
			TargetRPS:        10,
			Timeout:          50 * time.Millisecond,
			ConcurrencyLimit: 2,
		},
		steps,
		sender,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	gen.Start(ctx)

	firstStepCount := sender.CountBetween(start, 0, 80*time.Millisecond)
	secondStepCount := sender.CountBetween(start, 80*time.Millisecond, 160*time.Millisecond)

	if firstStepCount == 0 {
		t.Fatalf("expected requests during first scenario step, got %d", firstStepCount)
	}

	if secondStepCount <= firstStepCount {
		t.Fatalf("expected second step to send more requests after profile switch: first=%d second=%d", firstStepCount, secondStepCount)
	}
}

func TestEffectiveRPS_UsesSpikeWindow(t *testing.T) {
	cfg := Config{
		TargetRPS:        450,
		Timeout:          100 * time.Millisecond,
		ConcurrencyLimit: 32,
		SpikeMultiplier:  1.6,
		SpikeDuration:    8 * time.Second,
		SpikeInterval:    35 * time.Second,
	}

	if got := effectiveRPS(cfg, 450, 2*time.Second); got != 720 {
		t.Fatalf("unexpected spike rps: got %d want %d", got, 720)
	}

	if got := effectiveRPS(cfg, 450, 12*time.Second); got != 450 {
		t.Fatalf("unexpected steady-state rps: got %d want %d", got, 450)
	}

	if got := effectiveRPS(cfg, 450, 36*time.Second); got != 720 {
		t.Fatalf("unexpected recurring spike rps: got %d want %d", got, 720)
	}
}
