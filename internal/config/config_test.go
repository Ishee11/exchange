package config

import (
	"testing"
	"time"
)

func TestLoad_ProfileDefaults(t *testing.T) {
	t.Setenv("TRAFFIC_PROFILE", ProfileBurst)
	t.Setenv("DSP_URL", "http://dsp.local/bid")
	t.Setenv("METRICS_ADDR", ":9999")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.TrafficProfile != ProfileBurst {
		t.Fatalf("unexpected profile: got %q want %q", cfg.TrafficProfile, ProfileBurst)
	}

	if cfg.TargetRPS != 1000 {
		t.Fatalf("unexpected rps: got %d want %d", cfg.TargetRPS, 1000)
	}

	if cfg.ConcurrencyLimit != 256 {
		t.Fatalf("unexpected concurrency: got %d want %d", cfg.ConcurrencyLimit, 256)
	}

	if cfg.RequestTimeout != 90*time.Millisecond {
		t.Fatalf("unexpected timeout: got %v want %v", cfg.RequestTimeout, 90*time.Millisecond)
	}

	if cfg.DSPURL != "http://dsp.local/bid" || cfg.MetricsAddr != ":9999" {
		t.Fatalf("unexpected addresses: %+v", cfg)
	}
}

func TestLoad_OverridesProfileDefaults(t *testing.T) {
	t.Setenv("TRAFFIC_PROFILE", ProfileNormal)
	t.Setenv("TARGET_RPS", "777")
	t.Setenv("CONCURRENCY_LIMIT", "42")
	t.Setenv("REQUEST_TIMEOUT", "250ms")
	t.Setenv("REQUEST_JITTER", "8ms")
	t.Setenv("INVALID_SHARE", "0.05")
	t.Setenv("EXPENSIVE_SHARE", "0.20")
	t.Setenv("NO_BID_PRONE_SHARE", "0.30")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.TargetRPS != 777 || cfg.ConcurrencyLimit != 42 {
		t.Fatalf("overrides not applied: %+v", cfg)
	}

	if cfg.RequestTimeout != 250*time.Millisecond || cfg.Jitter != 8*time.Millisecond {
		t.Fatalf("duration overrides not applied: %+v", cfg)
	}

	if cfg.InvalidShare != 0.05 || cfg.ExpensiveShare != 0.20 || cfg.NoBidProneShare != 0.30 {
		t.Fatalf("mix overrides not applied: %+v", cfg)
	}
}
