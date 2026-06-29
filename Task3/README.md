# Task 3 — Масштабирование под нагрузку (Kubernetes HPA)

Динамическое масштабирование тестового приложения в Minikube: HPA по утилизации памяти (часть 1)
и по количеству запросов в секунду через Prometheus + prometheus-adapter (часть 2).

## Состав

| Файл | Назначение |
|------|-----------|
| [deployment.yaml](./deployment.yaml) | Deployment тест-приложения: 1 реплика, лимит памяти 30Mi, порт 8080 |
| [service.yaml](./service.yaml) | ClusterIP Service на 8080 |
| [hpa-memory.yaml](./hpa-memory.yaml) | Часть 1 — HPA по памяти: целевая утилизация 80%, 1..10 реплик |
| [hpa-rps.yaml](./hpa-rps.yaml) | Часть 2 — HPA по RPS на под (custom-метрика `http_requests_per_second`) |
| [prometheus/prometheus-adapter-values.yaml](./prometheus/prometheus-adapter-values.yaml) | Правило prometheus-adapter: `rate(http_requests_total[1m])` → custom-метрика для HPA |
| [locustfile.py](./locustfile.py) | Сценарий нагрузки (из задания) |
| [local-loadapp/](./local-loadapp) | arm64-аналог тест-приложения для живого прогона (см. «Замечания») |
| [verification/](./verification) | Логи живого прогона: изменение числа реплик под нагрузкой |

## Ключевые решения

- **HPA по памяти требует `requests.memory`.** Утилизация считается как процент от
  `requests` — поэтому в Deployment задан `requests.memory: 20Mi` (и `limits.memory: 30Mi` по
  условию). Без `requests` метрика `Utilization` не вычисляется.
- **Источник метрик памяти** — `metrics-server` (`minikube addons enable metrics-server`).
- **RPS-метрика** в Kubernetes нет «из коробки»: `metrics-server` отдаёт только CPU/память.
  Поэтому для части 2 поднимается Prometheus (скрейпит `/metrics` по аннотациям
  `prometheus.io/scrape`) и `prometheus-adapter`, который публикует
  `http_requests_per_second = sum(rate(http_requests_total[1m])) by (pod)` в
  `custom.metrics.k8s.io`. Её потребляет `hpa-rps.yaml` (цель — 10 rps/под).

## Как запустить и проверить

```bash
# 0. Кластер и метрики
minikube start --driver=docker
minikube addons enable metrics-server

# Часть 1 — HPA по памяти
kubectl apply -f deployment.yaml -f service.yaml -f hpa-memory.yaml
kubectl rollout status deploy/scaletestapp
kubectl get hpa scaletestapp-mem -w        # наблюдаем рост REPLICAS под нагрузкой
# нагрузка (через port-forward):
kubectl port-forward svc/scaletestapp 8080:8080 &
locust -f locustfile.py --host http://localhost:8080 --headless -u 200 -r 20 -t 3m

# Часть 2 — HPA по RPS (Prometheus + adapter)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/prometheus -n monitoring --create-namespace
helm install prometheus-adapter prometheus-community/prometheus-adapter \
  -n monitoring -f prometheus/prometheus-adapter-values.yaml
# проверить, что custom-метрика поднялась:
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/default/pods/*/http_requests_per_second"
kubectl apply -f hpa-rps.yaml
kubectl get hpa scaletestapp-rps -w        # REPLICAS растут при росте RPS
```

## Доказательства масштабирования

Реальные логи живого прогона (Minikube, нагрузка locust) — в [verification/](./verification):

| Файл | Что показывает |
|------|----------------|
| [00-minikube-start.log](./verification/00-minikube-start.log) | старт кластера |
| [10-mem-hpa-watch.log](./verification/10-mem-hpa-watch.log) | часть 1: утилизация памяти под нагрузкой `145%/80%`, число реплик растёт `1 → 2` |
| [11-mem-hpa-describe.txt](./verification/11-mem-hpa-describe.txt) | часть 1: `SuccessfulRescale … New size: 2; reason: memory resource utilization … above target` |
| [20-rps-hpa-watch.log](./verification/20-rps-hpa-watch.log) | часть 2: число реплик растёт `1 → 4 → 8 → 10` под нагрузкой и снижается после её снятия |
| [21-rps-hpa-describe.txt](./verification/21-rps-hpa-describe.txt) | часть 2: custom.metrics API отдаёт `http_requests_per_second` на под; события `New size: 4/8/10; reason: pods metric http_requests_per_second above target` |

**Итог.** Часть 1 — HPA по памяти масштабировал приложение `1 → 2` при превышении порога 80%
(`145%/80%`). Часть 2 — HPA по RPS через Prometheus + prometheus-adapter масштабировал `1 → 10`
(до максимума) под нагрузкой ~100 rps и затем сворачивал реплики после снятия нагрузки.

## Замечания

- **Опечатка в задании.** В тексте сказано «количество реплик **базы данных**» — в тестовом
  стенде базы данных нет; масштабируется само тестовое приложение (его Deployment). Доказательства
  показывают изменение числа реплик `scaletestapp`.
- **arm64-аналог для живого прогона.** Официальный образ `ghcr.io/yandex-practicum/scaletestapp`
  собран только под **amd64** и не запускается на Apple Silicon (arm64) без эмуляции (эмуляция в
  minikube нестабильна). Поэтому боевые манифесты ссылаются на официальный образ (корректно для
  amd64-кластера ревьюера), а для живого прогона на arm64 используется функционально идентичный
  локальный образ [local-loadapp/](./local-loadapp) (те же `GET /` и `GET /metrics` с
  `http_requests_total`). Запуск с ним: `kubectl apply -f local-loadapp/deployment.local.yaml`
  вместо `deployment.yaml` (Service и оба HPA — без изменений). GUI-скриншоты дашборда/Prometheus
  Web UI добавляются отдельно; для доказательства достаточно логов в `verification/`.
- **GUI-скриншоты** (дашборд Minikube, Prometheus Web UI) снимаются вручную; задание принимает
  «скриншоты ИЛИ логи», и логи в `verification/` фиксируют факт масштабирования.
