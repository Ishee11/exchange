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

	if cfg.TargetRPS != 450 {
		t.Fatalf("unexpected rps: got %d want %d", cfg.TargetRPS, 450)
	}

	if cfg.ConcurrencyLimit != 160 {
		t.Fatalf("unexpected concurrency: got %d want %d", cfg.ConcurrencyLimit, 160)
	}

	if cfg.RequestTimeout != 100*time.Millisecond {
		t.Fatalf("unexpected timeout: got %v want %v", cfg.RequestTimeout, 100*time.Millisecond)
	}

	if cfg.SpikeMultiplier != 1.6 || cfg.SpikeDuration != 8*time.Second || cfg.SpikeInterval != 35*time.Second {
		t.Fatalf("unexpected spike defaults: %+v", cfg)
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
	t.Setenv("SPIKE_MULTIPLIER", "2.0")
	t.Setenv("SPIKE_DURATION", "12s")
	t.Setenv("SPIKE_INTERVAL", "40s")
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

	if cfg.SpikeMultiplier != 2.0 || cfg.SpikeDuration != 12*time.Second || cfg.SpikeInterval != 40*time.Second {
		t.Fatalf("spike overrides not applied: %+v", cfg)
	}

	if cfg.InvalidShare != 0.05 || cfg.ExpensiveShare != 0.20 || cfg.NoBidProneShare != 0.30 {
		t.Fatalf("mix overrides not applied: %+v", cfg)
	}
}

func TestLoad_ParsesScenario(t *testing.T) {
	t.Setenv("LOAD_SCENARIO", "normal:2m, burst:45s,heavy:90s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.LoadScenario) != 3 {
		t.Fatalf("unexpected scenario length: got %d want %d", len(cfg.LoadScenario), 3)
	}

	if cfg.LoadScenario[0].Profile != ProfileNormal || cfg.LoadScenario[0].Duration != 2*time.Minute {
		t.Fatalf("unexpected first step: %+v", cfg.LoadScenario[0])
	}

	if cfg.LoadScenario[2].Profile != ProfileHeavy || cfg.LoadScenario[2].Duration != 90*time.Second {
		t.Fatalf("unexpected last step: %+v", cfg.LoadScenario[2])
	}
}
