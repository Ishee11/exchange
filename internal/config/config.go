package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ProfileNormal = "normal"
	ProfileBurst  = "burst"
	ProfileHeavy  = "heavy"
)

type Config struct {
	DSPURL           string
	MetricsAddr      string
	TrafficProfile   string
	LoadScenario     []ScenarioStep
	TargetRPS        int
	RequestTimeout   time.Duration
	ConcurrencyLimit int
	Jitter           time.Duration
	RampUpDuration   time.Duration
	PlateauDuration  time.Duration
	RampDownDuration time.Duration
	SpikeMultiplier  float64
	SpikeDuration    time.Duration
	SpikeInterval    time.Duration
	InvalidShare     float64
	ExpensiveShare   float64
	NoBidProneShare  float64
}

type ScenarioStep struct {
	Profile  string
	Duration time.Duration
}

func Load() (Config, error) {
	profile := strings.ToLower(getEnv("TRAFFIC_PROFILE", ProfileNormal))

	cfg, ok := ProfileDefaults(profile)
	if !ok {
		return Config{}, fmt.Errorf("unknown traffic profile %q", profile)
	}

	var err error

	cfg.TrafficProfile = profile
	cfg.DSPURL = getEnv("DSP_URL", "http://localhost:8080/bid")
	cfg.MetricsAddr = getEnv("METRICS_ADDR", ":2112")
	cfg.LoadScenario, err = parseScenario(getEnv("LOAD_SCENARIO", ""))
	if err != nil {
		return Config{}, err
	}

	if cfg.TargetRPS, err = getEnvInt("TARGET_RPS", cfg.TargetRPS); err != nil {
		return Config{}, err
	}

	if cfg.ConcurrencyLimit, err = getEnvInt("CONCURRENCY_LIMIT", cfg.ConcurrencyLimit); err != nil {
		return Config{}, err
	}

	if cfg.RequestTimeout, err = getEnvDuration("REQUEST_TIMEOUT", cfg.RequestTimeout); err != nil {
		return Config{}, err
	}

	if cfg.Jitter, err = getEnvDuration("REQUEST_JITTER", cfg.Jitter); err != nil {
		return Config{}, err
	}

	if cfg.RampUpDuration, err = getEnvDuration("RAMP_UP_DURATION", cfg.RampUpDuration); err != nil {
		return Config{}, err
	}

	if cfg.PlateauDuration, err = getEnvDuration("PLATEAU_DURATION", cfg.PlateauDuration); err != nil {
		return Config{}, err
	}

	if cfg.RampDownDuration, err = getEnvDuration("RAMP_DOWN_DURATION", cfg.RampDownDuration); err != nil {
		return Config{}, err
	}

	if cfg.SpikeMultiplier, err = getEnvFloat("SPIKE_MULTIPLIER", cfg.SpikeMultiplier); err != nil {
		return Config{}, err
	}

	if cfg.SpikeDuration, err = getEnvDuration("SPIKE_DURATION", cfg.SpikeDuration); err != nil {
		return Config{}, err
	}

	if cfg.SpikeInterval, err = getEnvDuration("SPIKE_INTERVAL", cfg.SpikeInterval); err != nil {
		return Config{}, err
	}

	if cfg.InvalidShare, err = getEnvFloat("INVALID_SHARE", cfg.InvalidShare); err != nil {
		return Config{}, err
	}

	if cfg.ExpensiveShare, err = getEnvFloat("EXPENSIVE_SHARE", cfg.ExpensiveShare); err != nil {
		return Config{}, err
	}

	if cfg.NoBidProneShare, err = getEnvFloat("NO_BID_PRONE_SHARE", cfg.NoBidProneShare); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.TargetRPS <= 0 {
		return fmt.Errorf("TARGET_RPS must be > 0")
	}

	if c.ConcurrencyLimit <= 0 {
		return fmt.Errorf("CONCURRENCY_LIMIT must be > 0")
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("REQUEST_TIMEOUT must be > 0")
	}

	if c.Jitter < 0 || c.RampUpDuration < 0 || c.PlateauDuration < 0 || c.RampDownDuration < 0 {
		return fmt.Errorf("durations must be >= 0")
	}

	if c.SpikeMultiplier < 1 {
		return fmt.Errorf("SPIKE_MULTIPLIER must be >= 1")
	}

	if c.SpikeDuration < 0 || c.SpikeInterval < 0 {
		return fmt.Errorf("spike durations must be >= 0")
	}

	if c.SpikeDuration > 0 && c.SpikeInterval == 0 {
		return fmt.Errorf("SPIKE_INTERVAL must be > 0 when SPIKE_DURATION is set")
	}

	if c.SpikeInterval > 0 && c.SpikeDuration > c.SpikeInterval {
		return fmt.Errorf("SPIKE_DURATION must be <= SPIKE_INTERVAL")
	}

	totalShare := c.InvalidShare + c.ExpensiveShare + c.NoBidProneShare
	if c.InvalidShare < 0 || c.ExpensiveShare < 0 || c.NoBidProneShare < 0 {
		return fmt.Errorf("request class shares must be >= 0")
	}

	if totalShare > 1 {
		return fmt.Errorf("request class shares must sum to <= 1")
	}

	return nil
}

func ProfileDefaults(profile string) (Config, bool) {
	switch profile {
	case ProfileNormal:
		return Config{
			TargetRPS:        180,
			RequestTimeout:   120 * time.Millisecond,
			ConcurrencyLimit: 64,
			Jitter:           5 * time.Millisecond,
			RampUpDuration:   5 * time.Second,
			PlateauDuration:  0,
			RampDownDuration: 0,
			SpikeMultiplier:  1.5,
			SpikeDuration:    5 * time.Second,
			SpikeInterval:    45 * time.Second,
			InvalidShare:     0.01,
			ExpensiveShare:   0.15,
			NoBidProneShare:  0.25,
		}, true
	case ProfileBurst:
		return Config{
			TargetRPS:        450,
			RequestTimeout:   100 * time.Millisecond,
			ConcurrencyLimit: 160,
			Jitter:           3 * time.Millisecond,
			RampUpDuration:   2 * time.Second,
			PlateauDuration:  15 * time.Second,
			RampDownDuration: 3 * time.Second,
			SpikeMultiplier:  1.6,
			SpikeDuration:    8 * time.Second,
			SpikeInterval:    35 * time.Second,
			InvalidShare:     0.02,
			ExpensiveShare:   0.25,
			NoBidProneShare:  0.3,
		}, true
	case ProfileHeavy:
		return Config{
			TargetRPS:        700,
			RequestTimeout:   80 * time.Millisecond,
			ConcurrencyLimit: 256,
			Jitter:           1 * time.Millisecond,
			RampUpDuration:   5 * time.Second,
			PlateauDuration:  30 * time.Second,
			RampDownDuration: 5 * time.Second,
			SpikeMultiplier:  1.14,
			SpikeDuration:    10 * time.Second,
			SpikeInterval:    30 * time.Second,
			InvalidShare:     0.03,
			ExpensiveShare:   0.35,
			NoBidProneShare:  0.35,
		}, true
	default:
		return Config{}, false
	}
}

func parseScenario(raw string) ([]ScenarioStep, error) {
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	steps := make([]ScenarioStep, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		pieces := strings.Split(part, ":")
		if len(pieces) != 2 {
			return nil, fmt.Errorf("LOAD_SCENARIO step %q must have format profile:duration", part)
		}

		profile := strings.ToLower(strings.TrimSpace(pieces[0]))
		if _, ok := ProfileDefaults(profile); !ok {
			return nil, fmt.Errorf("LOAD_SCENARIO step %q uses unknown profile %q", part, profile)
		}

		duration, err := time.ParseDuration(strings.TrimSpace(pieces[1]))
		if err != nil {
			return nil, fmt.Errorf("LOAD_SCENARIO step %q has invalid duration: %w", part, err)
		}

		if duration <= 0 {
			return nil, fmt.Errorf("LOAD_SCENARIO step %q must have duration > 0", part)
		}

		steps = append(steps, ScenarioStep{
			Profile:  profile,
			Duration: duration,
		})
	}

	if len(steps) == 0 {
		return nil, nil
	}

	return steps, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be int: %w", key, err)
	}

	return parsed, nil
}

func getEnvFloat(key string, fallback float64) (float64, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be float: %w", key, err)
	}

	return parsed, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be duration: %w", key, err)
	}

	return parsed, nil
}
