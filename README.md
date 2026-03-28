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
5. Логируется latency и статус ответа

## Конфигурация

Основные параметры задаются в main:

- RPS (запросов в секунду)
- timeout (дедлайн на запрос)
- URL DSP

## Запуск

```bash
go run ./cmd/app
