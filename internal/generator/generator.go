/*
Load generator для отправки неоднородных bid-запросов в DSP.

Работа:
- traffic profile задаёт базовые параметры нагрузки
- stages управляют ramp-up / plateau / ramp-down
- для каждого запроса выбирается request class из request mix
- отправка ограничена concurrency limit
- на каждый запрос создаётся context с timeout

Свойства:
- поддерживает jitter между запросами
- создаёт более правдоподобный mix трафика, чем однотипный fixed-RPS loop
- корректно обрабатывает shutdown через parent context
*/

package generator

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/Ishee11/exchange/internal/client"
	"github.com/Ishee11/exchange/internal/metrics"
	"github.com/Ishee11/exchange/internal/model"
)

type RequestClass string

const (
	RequestClassHot        RequestClass = "hot_path"
	RequestClassNoBidProne RequestClass = "no_bid_prone"
	RequestClassExpensive  RequestClass = "expensive"
	RequestClassInvalid    RequestClass = "invalid"
)

type RequestMix struct {
	InvalidShare    float64
	ExpensiveShare  float64
	NoBidProneShare float64
}

type Config struct {
	TargetRPS        int
	Timeout          time.Duration
	ConcurrencyLimit int
	Jitter           time.Duration
	RampUpDuration   time.Duration
	PlateauDuration  time.Duration
	RampDownDuration time.Duration
	RequestMix       RequestMix
}

type Sender interface {
	Send(ctx context.Context, req model.BidRequest) client.SendResult
}

type Generator struct {
	cfg    Config
	sender Sender
	sem    chan struct{}
}

func New(cfg Config, sender Sender) *Generator {
	if err := validateConfig(cfg); err != nil {
		log.Fatal(err)
	}

	return &Generator{
		cfg:    cfg,
		sender: sender,
		sem:    make(chan struct{}, cfg.ConcurrencyLimit),
	}
}

func (g *Generator) Start(ctx context.Context) {
	logConfig("generator started", g.cfg)

	if err := g.runProfile(ctx, g.cfg); err != nil && err != context.Canceled {
		log.Printf("generator stopped with error: %v\n", err)
	}

	log.Println("generator stopped")
}

func (g *Generator) runProfile(ctx context.Context, cfg Config) error {
	minRPS := max(1, cfg.TargetRPS/4)

	stages := []stage{
		{
			name:     "ramp_up",
			duration: cfg.RampUpDuration,
			rpsAt: func(progress float64) int {
				return interpolateRPS(minRPS, cfg.TargetRPS, progress)
			},
		},
		{
			name:     "plateau",
			duration: cfg.PlateauDuration,
			rpsAt: func(_ float64) int {
				return cfg.TargetRPS
			},
		},
	}

	if cfg.RampDownDuration > 0 && cfg.PlateauDuration > 0 {
		stages = append(stages, stage{
			name:     "ramp_down",
			duration: cfg.RampDownDuration,
			rpsAt: func(progress float64) int {
				return interpolateRPS(cfg.TargetRPS, minRPS, progress)
			},
		})
	}

	for _, stage := range stages {
		if stage.duration == 0 && stage.name != "plateau" {
			continue
		}

		log.Printf("generator stage started: name=%s duration=%v\n", stage.name, stage.duration)

		if err := g.runStage(ctx, cfg, stage); err != nil {
			return err
		}

		if stage.name == "plateau" && stage.duration == 0 {
			return context.Canceled
		}
	}

	return nil
}

func (g *Generator) runStage(ctx context.Context, cfg Config, stage stage) error {
	stageStart := time.Now()

	for {
		progress := stage.progress(stageStart)
		if stage.done(progress) {
			return nil
		}

		wait := nextInterval(cfg, stage.rpsAt(progress))
		if remaining := stage.remaining(stageStart); remaining > 0 && wait > remaining {
			wait = remaining
		}
		timer := time.NewTimer(wait)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		if stage.done(stage.progress(stageStart)) {
			return nil
		}

		if err := g.acquire(ctx); err != nil {
			return err
		}

		go g.fire(ctx, cfg)
	}
}

func (g *Generator) fire(parentCtx context.Context, cfg Config) {
	defer g.release()

	ctx, cancel := context.WithTimeout(parentCtx, cfg.Timeout)
	defer cancel()

	req := buildRequest(nextRequestClass(cfg.RequestMix))
	metrics.IncGeneratedRequest()
	g.sender.Send(ctx, req)
}

func (g *Generator) acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case g.sem <- struct{}{}:
		return nil
	}
}

func (g *Generator) release() {
	<-g.sem
}

func nextInterval(cfg Config, rps int) time.Duration {
	if rps <= 0 {
		rps = 1
	}

	base := time.Second / time.Duration(rps)
	if cfg.Jitter == 0 {
		return base
	}

	jitter := time.Duration(rand.Int63n(int64(cfg.Jitter)*2+1)) - cfg.Jitter
	if base+jitter < time.Millisecond {
		return time.Millisecond
	}

	return base + jitter
}

func nextRequestClass(mix RequestMix) RequestClass {
	value := rand.Float64()

	if value < mix.InvalidShare {
		return RequestClassInvalid
	}

	value -= mix.InvalidShare
	if value < mix.ExpensiveShare {
		return RequestClassExpensive
	}

	value -= mix.ExpensiveShare
	if value < mix.NoBidProneShare {
		return RequestClassNoBidProne
	}

	return RequestClassHot
}

// ---- request generation ----

func buildRequest(class RequestClass) model.BidRequest {
	req := model.BidRequest{
		RequestID: genID(),
		ImpID:     genID(),
		UserID:    genID(),
		Timestamp: time.Now().UnixMilli(),
	}

	switch class {
	case RequestClassExpensive:
		req.SiteID = randomFrom(expensiveSites)
		req.PlacementID = randomFrom(expensivePlacements)
		req.FloorPrice = 2.5 + rand.Float64()*4
		req.DeviceType = "desktop"
	case RequestClassNoBidProne:
		req.SiteID = randomFrom(noBidProneSites)
		req.PlacementID = randomFrom(noBidPronePlacements)
		req.FloorPrice = 4 + rand.Float64()*3
		req.DeviceType = randomFrom(devices)
	case RequestClassInvalid:
		req.SiteID = ""
		req.PlacementID = "broken_slot"
		req.FloorPrice = -1
		req.UserID = ""
		req.DeviceType = "unknown"
		req.Timestamp = time.Now().Add(-24 * time.Hour).UnixMilli()
	default:
		req.SiteID = randomFrom(hotSites)
		req.PlacementID = randomFrom(hotPlacements)
		req.FloorPrice = 0.2 + rand.Float64()*1.5
		req.DeviceType = randomFrom(devices)
	}

	return req
}

var hotSites = []string{"news", "sports", "social"}
var expensiveSites = []string{"finance", "streaming", "marketplace"}
var noBidProneSites = []string{"remnant", "long_tail", "gaming"}
var hotPlacements = []string{"banner_top", "feed_inline", "sidebar"}
var expensivePlacements = []string{"premium_video", "homepage_takeover", "above_the_fold"}
var noBidPronePlacements = []string{"footer", "below_the_fold", "sidebar_small"}
var devices = []string{"mobile", "desktop"}

func genID() string {
	return strconv.FormatInt(time.Now().UnixNano()+rand.Int63n(1000), 36)
}

func randomFrom(values []string) string {
	return values[rand.Intn(len(values))]
}

type stage struct {
	name     string
	duration time.Duration
	rpsAt    func(progress float64) int
}

func (s stage) progress(start time.Time) float64 {
	if s.duration <= 0 {
		return 0
	}

	progress := float64(time.Since(start)) / float64(s.duration)
	if progress > 1 {
		return 1
	}

	return progress
}

func (s stage) done(progress float64) bool {
	return s.duration > 0 && progress >= 1
}

func (s stage) remaining(start time.Time) time.Duration {
	if s.duration <= 0 {
		return 0
	}

	remaining := s.duration - time.Since(start)
	if remaining < 0 {
		return 0
	}

	return remaining
}

func interpolateRPS(from, to int, progress float64) int {
	if progress < 0 {
		progress = 0
	}

	if progress > 1 {
		progress = 1
	}

	value := float64(from) + float64(to-from)*progress
	if value < 1 {
		return 1
	}

	return int(value)
}

func validateConfig(cfg Config) error {
	if cfg.TargetRPS <= 0 {
		return fmt.Errorf("target rps must be > 0")
	}

	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0")
	}

	if cfg.ConcurrencyLimit <= 0 {
		return fmt.Errorf("concurrency limit must be > 0")
	}

	totalShare := cfg.RequestMix.InvalidShare + cfg.RequestMix.ExpensiveShare + cfg.RequestMix.NoBidProneShare
	if totalShare > 1 {
		return fmt.Errorf("request mix shares must sum to <= 1")
	}

	if cfg.Jitter < 0 || cfg.RampUpDuration < 0 || cfg.PlateauDuration < 0 || cfg.RampDownDuration < 0 {
		return fmt.Errorf("durations must be >= 0")
	}

	return nil
}

func logConfig(prefix string, cfg Config) {
	log.Printf(
		"%s: target_rps=%d timeout=%v concurrency_limit=%d jitter=%v ramp_up=%v plateau=%v ramp_down=%v invalid_share=%.2f expensive_share=%.2f no_bid_prone_share=%.2f\n",
		prefix,
		cfg.TargetRPS,
		cfg.Timeout,
		cfg.ConcurrencyLimit,
		cfg.Jitter,
		cfg.RampUpDuration,
		cfg.PlateauDuration,
		cfg.RampDownDuration,
		cfg.RequestMix.InvalidShare,
		cfg.RequestMix.ExpensiveShare,
		cfg.RequestMix.NoBidProneShare,
	)
}
