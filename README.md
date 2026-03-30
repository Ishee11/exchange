# Exchange (Traffic Simulator)

Минимальная реализация Ad Exchange, генерирующая bid-запросы к DSP с заданным RPS и timeout.

## Архитектура

Сервис состоит из трёх основных слоёв:

- generator — отвечает за генерацию нагрузки (RPS) и дедлайны
- client — отвечает за HTTP-взаимодействие с DSP
- model — описывает структуру bid-запроса

Общий поток:

generator → client → DSP

## Как работает

1. Generator запускает ticker с интервалом 1/RPS
2. На каждый тик создаётся goroutine
3. Для каждого запроса создаётся context с timeout
4. Client отправляет HTTP POST в DSP
5. Результаты отправки и latency экспортируются в Prometheus-метрики

## Конфигурация

Основные параметры задаются в main:

- RPS (запросов в секунду)
- timeout (дедлайн на запрос)
- URL DSP
- `METRICS_ADDR` (по умолчанию `:2112`)

## Запуск

```bash
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
