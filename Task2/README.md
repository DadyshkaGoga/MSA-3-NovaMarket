# Task 2 — Агрегированные данные на фронте: история заказов (CQRS)

Реализация сценария «Просмотр истории покупок»: обновлённая контейнерная диаграмма (C2) и ADR с
выбором паттерна агрегации данных.

## Состав

| Файл | Назначение |
|------|-----------|
| [ADR.md](./ADR.md) | Архитектурное решение: выбор CQRS (read-model) против API Composition и Event Sourcing — со взвешенной матрицей, плюсами/минусами и последствиями |
| [c2-eda-history.puml](./c2-eda-history.puml) / [c2-eda-history.png](./c2-eda-history.png) | Обновлённый C2: добавлен Order History Service (CQRS read-model) и его read-store |

## Краткий вывод

- Для read-heavy сценария истории под NFR отклика (p98 0,3 c / p99.99 ≤ 1 c) выбран **CQRS**:
  выделенный **Order History Service** с денормализованной проекцией, наполняемой доменными
  событиями из Task1 (`OrderCreated`, `PaymentSucceeded`, `DeliveryScheduled`, `OrderDelivered`,
  `OrderCancelled`).
- Чтение истории — один быстрый запрос к проекции, без read-time fan-out к write-сервисам.
- CQRS — единственный вариант, прошедший gate-критерий по NFR отклика (см. матрицу в ADR);
  API Composition и Event Sourcing отклонены (детали и баллы — в [ADR.md](./ADR.md)).
- Решение переиспользует уже существующий событийный поток: read-model — ещё один потребитель тех
  же топиков, новых требований к write-сервисам нет.

## Как воспроизвести (рендер диаграммы)

```bash
cd MSA-3-NovaMarket
docker run --rm -v "$PWD/Task2:/work" -w /work plantuml/plantuml -tpng "*.puml"
```

## Замечания

- Eventual consistency проекции (секунды) — осознанный компромисс; живой статус заказа покупатель
  видит в Order Service, история — постфактум. Митигации (идемпотентность, перестроение проекции из
  лога событий) описаны в ADR, раздел «Compromises».
- Чек и отзыв не дублируются в проекцию: история отдаёт ссылки, чек берётся из Payment, создание
  отзыва — в Review (точечная API Composition для редких действий).
