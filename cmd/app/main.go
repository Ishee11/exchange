package main

import (
	"context"
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

	gen := generator.New(
		toGeneratorConfig(cfg),
		c,
	)

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
