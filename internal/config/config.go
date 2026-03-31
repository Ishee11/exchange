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
	TargetRPS        int
	RequestTimeout   time.Duration
	ConcurrencyLimit int
	Jitter           time.Duration
	RampUpDuration   time.Duration
	PlateauDuration  time.Duration
	RampDownDuration time.Duration
	InvalidShare     float64
	ExpensiveShare   float64
	NoBidProneShare  float64
}

func Load() (Config, error) {
	profile := strings.ToLower(getEnv("TRAFFIC_PROFILE", ProfileNormal))

	cfg, ok := profileDefaults(profile)
	if !ok {
		return Config{}, fmt.Errorf("unknown traffic profile %q", profile)
	}

	var err error

	cfg.TrafficProfile = profile
	cfg.DSPURL = getEnv("DSP_URL", "http://localhost:8080/bid")
	cfg.MetricsAddr = getEnv("METRICS_ADDR", ":2112")

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

	totalShare := c.InvalidShare + c.ExpensiveShare + c.NoBidProneShare
	if c.InvalidShare < 0 || c.ExpensiveShare < 0 || c.NoBidProneShare < 0 {
		return fmt.Errorf("request class shares must be >= 0")
	}

	if totalShare > 1 {
		return fmt.Errorf("request class shares must sum to <= 1")
	}

	return nil
}

func profileDefaults(profile string) (Config, bool) {
	switch profile {
	case ProfileNormal:
		return Config{
			TargetRPS:        200,
			RequestTimeout:   120 * time.Millisecond,
			ConcurrencyLimit: 64,
			Jitter:           5 * time.Millisecond,
			RampUpDuration:   5 * time.Second,
			PlateauDuration:  0,
			RampDownDuration: 0,
			InvalidShare:     0.01,
			ExpensiveShare:   0.15,
			NoBidProneShare:  0.25,
		}, true
	case ProfileBurst:
		return Config{
			TargetRPS:        1000,
			RequestTimeout:   90 * time.Millisecond,
			ConcurrencyLimit: 256,
			Jitter:           2 * time.Millisecond,
			RampUpDuration:   2 * time.Second,
			PlateauDuration:  15 * time.Second,
			RampDownDuration: 3 * time.Second,
			InvalidShare:     0.02,
			ExpensiveShare:   0.25,
			NoBidProneShare:  0.3,
		}, true
	case ProfileHeavy:
		return Config{
			TargetRPS:        3000,
			RequestTimeout:   70 * time.Millisecond,
			ConcurrencyLimit: 1024,
			Jitter:           1 * time.Millisecond,
			RampUpDuration:   5 * time.Second,
			PlateauDuration:  30 * time.Second,
			RampDownDuration: 5 * time.Second,
			InvalidShare:     0.03,
			ExpensiveShare:   0.35,
			NoBidProneShare:  0.35,
		}, true
	default:
		return Config{}, false
	}
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
