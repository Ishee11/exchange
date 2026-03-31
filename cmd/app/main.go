package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/Ishee11/exchange/internal/client"
	"github.com/Ishee11/exchange/internal/config"
	"github.com/Ishee11/exchange/internal/generator"
	"github.com/Ishee11/exchange/internal/metrics"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	go serveMetrics(cfg.MetricsAddr)

	c := client.New(cfg.DSPURL)

	genCfg := toGeneratorConfig(cfg)
	scenario, err := toGeneratorScenario(cfg.LoadScenario)
	if err != nil {
		log.Fatalf("scenario config error: %v", err)
	}

	gen := generator.NewWithScenario(genCfg, scenario, c)

	gen.Start(context.Background())
}

func serveMetrics(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())

	log.Printf("metrics server started: addr=%s path=/metrics\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("metrics server failed: %v", err)
	}
}

func toGeneratorConfig(cfg config.Config) generator.Config {
	return generator.Config{
		TargetRPS:        cfg.TargetRPS,
		Timeout:          cfg.RequestTimeout,
		ConcurrencyLimit: cfg.ConcurrencyLimit,
		Jitter:           cfg.Jitter,
		RampUpDuration:   cfg.RampUpDuration,
		PlateauDuration:  cfg.PlateauDuration,
		RampDownDuration: cfg.RampDownDuration,
		RequestMix: generator.RequestMix{
			InvalidShare:    cfg.InvalidShare,
			ExpensiveShare:  cfg.ExpensiveShare,
			NoBidProneShare: cfg.NoBidProneShare,
		},
	}
}

func toGeneratorScenario(steps []config.ScenarioStep) ([]generator.ScenarioStep, error) {
	if len(steps) == 0 {
		return nil, nil
	}

	result := make([]generator.ScenarioStep, 0, len(steps))

	for _, step := range steps {
		profileCfg, ok := config.ProfileDefaults(step.Profile)
		if !ok {
			return nil, fmt.Errorf("unknown scenario profile %q", step.Profile)
		}

		result = append(result, generator.ScenarioStep{
			Name:     step.Profile,
			Duration: step.Duration,
			Config:   toGeneratorConfig(profileCfg),
		})
	}

	return result, nil
}
