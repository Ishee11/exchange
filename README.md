# Exchange (Traffic Simulator)

Минимальная реализация Ad Exchange, генерирующая неоднородные bid-запросы к DSP с управляемым профилем нагрузки.

## Архитектура

Сервис состоит из трёх основных слоёв:

- generator — отвечает за профили нагрузки, stages, request mix и дедлайны
- client — отвечает за HTTP-взаимодействие с DSP
- model — описывает структуру bid-запроса
- config — собирает runtime-конфигурацию из env

Общий поток:

generator → client → DSP

## Как работает

1. Generator читает traffic profile и runtime config
2. Нагрузка проходит через stages: `ramp-up`, `plateau`, `ramp-down`
3. Для каждого запроса выбирается request class: `hot_path`, `no_bid_prone`, `expensive`, `invalid`
4. Генерация ограничивается `concurrency_limit`, а между запросами добавляется jitter
5. Для каждого запроса создаётся context с timeout
6. Client отправляет HTTP POST в DSP
7. Результаты отправки и latency экспортируются в Prometheus-метрики

## Конфигурация

Основные параметры задаются через env:

- `DSP_URL` — URL DSP, по умолчанию `http://localhost:8080/bid`
- `METRICS_ADDR` — адрес HTTP endpoint с метриками, по умолчанию `:2112`
- `TRAFFIC_PROFILE` — `normal`, `burst`, `heavy`
- `TARGET_RPS` — целевой RPS профиля
- `CONCURRENCY_LIMIT` — максимум одновременных in-flight запросов
- `REQUEST_TIMEOUT` — дедлайн на один запрос
- `REQUEST_JITTER` — случайное отклонение интервала между запросами
- `RAMP_UP_DURATION` — длительность ramp-up
- `PLATEAU_DURATION` — длительность plateau, `0` означает бесконечную steady-state нагрузку
- `RAMP_DOWN_DURATION` — длительность ramp-down
- `INVALID_SHARE` — доля частично сломанных запросов
- `EXPENSIVE_SHARE` — доля expensive запросов
- `NO_BID_PRONE_SHARE` — доля no-bid-prone запросов

## Запуск

```bash
go run ./cmd/app
```

Пример запуска c burst-профилем:

```bash
TRAFFIC_PROFILE=burst \
TARGET_RPS=1500 \
CONCURRENCY_LIMIT=300 \
REQUEST_TIMEOUT=80ms \
PLATEAU_DURATION=20s \
go run ./cmd/app
```

Метрики доступны на `http://localhost:2112/metrics`, если не переопределять `METRICS_ADDR`.

## Prometheus

Экспортируемые метрики:

- `exchange_generator_requests_total` — сколько bid-запросов сгенерировано
- `exchange_client_requests_total{status,http_status,http_status_class}` — чем закончилась отправка во внешний DSP
- `exchange_client_request_duration_seconds{status}` — latency исходящего запроса
- `exchange_client_requests_in_flight` — сколько исходящих запросов одновременно в работе

Пример `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: exchange
    static_configs:
      - targets:
          - localhost:2112
```
