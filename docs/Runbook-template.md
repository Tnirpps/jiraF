

! Пишет: Разработчик

# Runbook: [Название сервиса / фичи]

> **Что это?** Операционное руководство для дежурного инженера (on-call).  
> Документ должен отвечать на вопрос: «Что делать, когда что-то сломалось?»  
> Пиши как будто читателю звонят в 3 ночи — никаких лишних слов, только конкретные шаги.

---

| Поле | Значение |
|---|---|
| **Сервис** | _название_ |
| **Команда** | _название команды_ |
| **Владелец** | @username |
| **On-call контакт** | @username / Telegram: @handle |
| **Обновлено** | ГГГГ-ММ-ДД |

---

## Быстрый старт (Getting Started)

### Что делает этот сервис

> 2-3 предложения. Какую задачу решает? Кто его вызывает?

### Где запущен

| Окружение | Адрес / ссылка |
|---|---|
| Production | https://... |
| Staging | https://... |
| Grafana dashboard | https://grafana.wb.ru/... |
| Логи (Kibana/Loki) | https://... |

### Как запустить локально

```bash
# 1. Клонировать репозиторий
git clone https://git.wb.ru/...

# 2. Установить зависимости
make deps   # или: go mod download / npm install

# 3. Настроить переменные окружения
cp .env.example .env
# Заполнить: DATABASE_URL, KAFKA_BROKERS, ...

# 4. Поднять зависимости (docker-compose)
docker compose up -d postgres redis kafka

# 5. Запустить сервис
make run    # или: go run ./cmd/server / npm start
```

### Запуск тестов

```bash
make test           # unit tests
make test-integration  # integration tests (нужен docker compose)
```

---

## Архитектура (кратко)

```
[Caller] --> [Этот сервис] --> [DB]
                           --> [Kafka topic: events]
                           --> [External WB Service]
```

> Добавь диаграмму если нужно. Главное — понять основные потоки данных.

---

## Конфигурация

| Переменная | Описание | Пример |
|---|---|---|
| `DATABASE_URL` | Строка подключения к PostgreSQL | `postgres://user:pass@host:5432/db` |
| `KAFKA_BROKERS` | Адреса брокеров Kafka | `kafka1:9092,kafka2:9092` |
| `LOG_LEVEL` | Уровень логирования | `info` / `debug` / `error` |
| `PORT` | Порт HTTP-сервера | `8080` |

> Конфиги хранятся в: _Vault / ConfigMap / .env (staging only)_

---

## Эксплуатация

### Рестарт сервиса

```bash
# Kubernetes
kubectl rollout restart deployment/<service-name> -n <namespace>

# Или через ArgoCD / деплой-систему WB
```

### Проверить статус

```bash
# Health check
curl https://<service-host>/health

# Ожидаемый ответ:
# {"status": "ok", "version": "1.2.3"}
```

### Посмотреть логи

```bash
# Kubernetes logs
kubectl logs -l app=<service-name> -n <namespace> --tail=100 -f

# Через Kibana/Loki: ссылка на saved search
```

---

## Алерты и инциденты

### Список алертов

| Алерт | Что означает | Первое действие |
|---|---|---|
| `HighErrorRate` | Error rate > 5% | Проверить логи, см. [Error rate вырос](#error-rate-вырос) |
| `HighLatency` | p99 > 500ms | Проверить БД и внешние зависимости |
| `ServiceDown` | Health check упал | Рестарт пода, эскалировать если не помогло |

### Error rate вырос

1. Открыть дашборд: [Grafana](https://grafana.wb.ru/...)
2. Посмотреть последние логи: `kubectl logs ... | grep ERROR`
3. Проверить статус зависимостей (БД, Kafka, внешние сервисы)
4. Если причина — деплой: откатить (`helm rollback` или ArgoCD)
5. Если не ясно: эскалировать в @on-call

### Сервис не отвечает (Health check failed)

1. `kubectl get pods -n <namespace>` — проверить статус подов
2. Если pod в `CrashLoopBackOff`: `kubectl describe pod <pod-name>` — причина
3. Рестарт: `kubectl rollout restart deployment/<name>`
4. Если не помогло за 5 минут — эскалировать

### Медленные запросы к БД

1. Проверить активные запросы:
```sql
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';
```
2. Убить зависший запрос: `SELECT pg_cancel_backend(<pid>);`
3. Проверить индексы — возможно, нужен REINDEX

---

## Откат (Rollback)

```bash
# Helm
helm rollback <release-name> <revision>

# Проверить текущую ревизию
helm history <release-name>
```

**Откат миграции БД:**

```bash
# Пример с goose
goose -dir ./migrations postgres "$DATABASE_URL" down

# Пример с Flyway
flyway -url=... undo
```

> ⚠️ Откат миграции возможен только если миграция была написана обратно совместимой.  
> Если нет — согласуй с DBA перед откатом.

---

## Контакты и эскалация

| Кому | Когда | Контакт |
|---|---|---|
| On-call разработчик | Любой инцидент | @username / Telegram |
| Тимлид | Не можем устранить за 30 мин | @username |
| DBA | Проблемы с БД | @dba-oncall |
| DevOps / Infra | Проблемы с инфраструктурой | #infra-oncall в Slack |

