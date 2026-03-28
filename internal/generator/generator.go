/*
RPS-based generator для отправки bid-запросов.

Работа:
- interval = 1s / RPS
- ticker генерирует события
- на каждый тик запускается goroutine (fire)
- внутри создаётся context с timeout → sender.Send(ctx, req)

Свойства:
- не блокируется медленными запросами
- соблюдает дедлайн на каждый запрос
- корректно обрабатывает shutdown через parent context

Ограничения:
- ticker не даёт точный RPS под нагрузкой
- нет ограничения на количество goroutine

Прод-улучшения:
- worker pool
- rate limiter
- метрики
*/

package generator

import (
	"context"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/Ishee11/exchange/internal/model"
)

type Sender interface {
	Send(ctx context.Context, req model.BidRequest)
}

type Generator struct {
	rps     int
	timeout time.Duration
	sender  Sender
}

func New(rps int, timeout time.Duration, sender Sender) *Generator {
	if rps <= 0 {
		log.Fatal("rps must be > 0")
	}

	return &Generator{
		rps:     rps,
		timeout: timeout,
		sender:  sender,
	}
}

func (g *Generator) Start(ctx context.Context) {
	interval := time.Second / time.Duration(g.rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("generator started: rps=%d\n", g.rps)

	for {
		select {
		case <-ctx.Done():
			log.Println("generator stopped")
			return

		case <-ticker.C:
			go g.fire(ctx)
		}
	}
}

func (g *Generator) fire(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, g.timeout)
	defer cancel()

	req := buildRequest()
	g.sender.Send(ctx, req)
}

// ---- request generation ----

func buildRequest() model.BidRequest {
	return model.BidRequest{
		RequestID:   genID(),
		ImpID:       genID(),
		SiteID:      randomSite(),
		PlacementID: randomPlacement(),
		FloorPrice:  randomFloor(),
		UserID:      genID(),
		DeviceType:  randomDevice(),
		Timestamp:   time.Now().UnixMilli(),
	}
}

var sites = []string{"1", "2", "3"}
var placements = []string{"banner_top", "sidebar", "footer"}
var devices = []string{"mobile", "desktop"}

func genID() string {
	return strconv.FormatInt(time.Now().UnixNano()+rand.Int63n(1000), 36)
}

func randomSite() string {
	return sites[rand.Intn(len(sites))]
}

func randomPlacement() string {
	return placements[rand.Intn(len(placements))]
}

func randomDevice() string {
	return devices[rand.Intn(len(devices))]
}

func randomFloor() float64 {
	return 0.5 + rand.Float64()*2
}
