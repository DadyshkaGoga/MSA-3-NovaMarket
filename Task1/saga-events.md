# Реестр событий Saga-хореографии «Оформление заказа»

Оформление заказа — распределённая транзакция через несколько сервисов (Order → Inventory →
Payment → Delivery). Используется **Saga-хореография**: центрального оркестратора нет, каждый
сервис подписан на событие предыдущего шага, выполняет свою часть и публикует следующее событие.
При отказе на любом шаге запускаются **компенсации** — откат ранее сделанных действий в обратном
порядке.

## Типы событий

- **domain** — успешный бизнес-факт (шаг выполнен);
- **failure** — отказ шага (бизнес- или техническая ошибка);
- **compensation** — откат ранее выполненного действия.

## Таблица событий

| Этап | Тип события | Название |
|---|---|---|
| Корзина оформлена, инициировано создание заказа | domain | `CartCheckedOut` |
| Заказ создан (ожидает резервирования) | domain | `OrderCreated` |
| Товары зарезервированы на складе | domain | `InventoryReserved` |
| Не удалось зарезервировать (нет наличия) | failure | `InventoryReservationFailed` |
| Оплата прошла успешно | domain | `PaymentSucceeded` |
| Оплата отклонена (нет средств / ошибка шлюза) | failure | `PaymentFailed` |
| Заявка на доставку оформлена | domain | `DeliveryScheduled` |
| Не удалось оформить доставку | failure | `DeliverySchedulingFailed` |
| Снятие резерва со склада (компенсация) | compensation | `InventoryReleased` |
| Возврат средств покупателю (компенсация) | compensation | `PaymentRefunded` |
| Заказ подтверждён — терминальный успех | domain | `OrderConfirmed` |
| Заказ отменён — терминальный откат | compensation | `OrderCancelled` |
| Заказ доставлен (после отгрузки логистом) | domain | `OrderDelivered` |

## Компенсационная матрица

Какие компенсации запускает каждый `failure` (откатывается только то, что уже выполнено):

| Отказ | Что уже сделано к этому моменту | Компенсации (в обратном порядке) |
|---|---|---|
| `InventoryReservationFailed` | ничего не занято/списано | нет → `OrderCancelled` |
| `PaymentFailed` | товары зарезервированы | `InventoryReleased` → `OrderCancelled` |
| `DeliverySchedulingFailed` | резерв + оплата | `PaymentRefunded` → `InventoryReleased` → `OrderCancelled` |

## Топики Kafka и владельцы

| Топик | Продьюсер | Основные потребители |
|---|---|---|
| `order-events` | Order | Inventory, Notification, Seller |
| `inventory-events` | Inventory | Payment, Order, Catalog, Notification |
| `payment-events` | Payment | Delivery, Order, Notification |
| `delivery-events` | Delivery | Order, Notification, Seller |

## Семантика доставки (зафиксировано)

- Все доменные события — **at-least-once**; потребители **идемпотентны** по ключу заказа
  (`orderId` + тип события), что даёт эффективно exactly-once-эффект на бизнес-уровне.
- Событие публикуется в той же логической транзакции, что и изменение состояния сервиса
  (паттерн **Transactional Outbox** — как прод-апгрейд; в проектной схеме обозначен явно, без
  реализации).
- Имена событий — в прошедшем времени (событие фиксирует свершившийся факт).

## Соответствие сценарию задания

Этапы оформления из задания → события: «Подтвердить состав» → `CartCheckedOut`/`OrderCreated`;
«Проверить и зарезервировать» → `InventoryReserved`/`InventoryReservationFailed`; «Оплатить» →
`PaymentSucceeded`/`PaymentFailed`; «Подготовить к доставке» → `DeliveryScheduled`/
`DeliverySchedulingFailed`. Статусы личного кабинета («Ожидает оплаты» → «Оплачен и готовится» →
«Передан в доставку» → «Доставлен») Order Service выставляет, потребляя соответствующие события.
