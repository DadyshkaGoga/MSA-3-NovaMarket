# Task 4 — Надёжность приложения (NGINX)

Повышение надёжности на NGINX: rate limiting с разными лимитами на клиентский канал и circuit
breaker для вызовов внешнего логистического сервиса.

## Состав

| Файл | Назначение |
|------|-----------|
| [rate_limiter.conf](./rate_limiter.conf) | Часть 1 — лимиты по каналу: web 50 r/s, mobile 30 r/s с одного IP |
| [circuit_breaker.conf](./circuit_breaker.conf) | Часть 2 — circuit breaker: таймаут 3 c, 5 ошибок → 30 c open + fallback |
| [nginx-test.conf](./nginx-test.conf) | Полный рабочий конфиг с тестовыми upstream/mock для прогона |
| [rate_limiter.py](./rate_limiter.py) | Locust-сценарий проверки rate limiter (web/mobile) |
| [circuit_breaker.py](./circuit_breaker.py) | Locust-сценарий проверки circuit breaker |
| [verification/](./verification) | Логи срабатывания Rate Limiter (429) и Circuit Breaker (fallback) |

## Ключевые решения

**Rate limiter по каналу (часть 1).** Open-source NGINX не выбирает зону лимита условно, поэтому
канал (заголовок `X-Client-Type: web|mobile`) через `map` задаёт ключ зоны, а **пустой ключ
отключает** лимит для неподходящего канала. Две зоны: `web` (50 r/s) и `mobile` (30 r/s), обе по
`$binary_remote_addr` (на один IP). Превышение → `429`.

**Circuit breaker (часть 2).** Активный `health_check` доступен только в NGINX Plus; на
open-source «размыкание» делает **пассивный** health-check апстрима: `max_fails=5 fail_timeout=30s`
помечает сервер недоступным на 30 c после 5 ошибок (затем half-open пробный запрос). Жёсткий
таймаут ответа — `proxy_connect/read/send_timeout 3s`. Контролируемый ответ при размыкании — через
`error_page 502 503 504 = @fallback`. Что считается ошибкой апстрима — задаёт `proxy_next_upstream`
(таймаут/соединение/5xx).

## Как запустить и проверить

```bash
# NGINX с полным тестовым конфигом
docker run --rm --name nginx-test -p 8080:8080 \
  -v "$PWD/nginx-test.conf:/etc/nginx/nginx.conf:ro" nginx:alpine

# Часть 1 — rate limiter: высокий RPS с одного IP -> 429 после 50 (web) / 30 (mobile) r/s
locust -f rate_limiter.py --host=http://localhost:8080 --headless -u 80 -r 80 -t 30s

# Часть 2 — circuit breaker: первые ~5 запросов ждут таймаут 3 c (логист недоступен),
# затем breaker размыкается -> мгновенный fallback (время ответа падает с ~3000 мс до единиц мс)
locust -f circuit_breaker.py --host=http://localhost:8080 --headless -u 20 -r 20 -t 60s
```

## Доказательства срабатывания

Логи живого прогона (NGINX в Docker + locust) — в [verification/](./verification):

| Файл | Что показывает |
|------|----------------|
| [30-rate-limiter-locust.log](./verification/30-rate-limiter-locust.log) | rate limiter: при обстреле ~1200 r/s с одного IP отклоняется **96.9% (web)** и **98.2% (mobile)** запросов кодом `429`; пропускная способность совпадает с лимитами (web ~50 r/s, mobile ~30 r/s) |
| [31-circuit-breaker-curl.log](./verification/31-circuit-breaker-curl.log) | circuit breaker, последовательные запросы (два недоступных сервера): первые ~5 запросов ловят по одному таймауту **~3.0 s** вперемешку с мгновенным fallback на уже помеченном сервере; после ~5 ошибок на сервер (≈запрос 10) breaker полностью открыт → запросы 11+ мгновенные **~0.001 s** |
| [32-circuit-breaker-locust.log](./verification/32-circuit-breaker-locust.log) | circuit breaker, locust: медиана **2 мс** (fallback при открытом breaker), хвост **3000 мс** на перцентиле 99.99% (таймауты до размыкания и периодические half-open пробы) |

## Замечания

- В `nginx-test.conf` апстрим `logistics_backend` указывает на недоступные адреса (blackhole),
  чтобы обращения реально упирались в таймаут 3 c — это и нужно для наблюдаемого срабатывания
  circuit breaker (return-based mock не может «висеть»). Боевой `circuit_breaker.conf` ссылается на
  реальные эндпоинты логиста (`logistics-1/2.company.com`).
- **Особенность инфраструктуры:** open-source NGINX **игнорирует** `max_fails` для upstream из
  одного сервера («such a server will never be considered unavailable»). Поэтому и боевой
  `circuit_breaker.conf`, и `nginx-test.conf` задают по ДВА эндпоинта логиста
  (+ `proxy_next_upstream_tries 1`, чтобы один запрос не упирался в 2×3 c) — только так пассивный
  health-check реально размыкает breaker. Наблюдаемое размыкание снято в `verification/31..32`.
- Скриншоты срабатывания (для сдачи) снимаются с веб-интерфейса locust / из логов; задание
  допускает и логи — они зафиксированы в `verification/`.
