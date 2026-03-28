# Runbook: jiraF Telegram Bot Operations

## Описание

Этот документ содержит пошаговые инструкции для эксплуатации, обслуживания и реагирования на инциденты бота jiraF.

---

## Содержание

1. [Запуск бота](#1-запуск-бота)
2. [Остановка бота](#2-остановка-бота)
3. [Проверка состояния](#3-проверка-состояния)
4. [Перезапуск бота](#4-перезапуск-бота)
5. [Обновление конфигурации](#5-обновление-конфигурации)
6. [Работа с базой данных](#6-работа-с-базой-данных)
7. [Реагирование на инциденты](#7-реагирование-на-инциденты)
8. [Логирование и диагностика](#8-логирование-и-диагностика)

---

## 1. Запуск бота

### ⚠️ Важные предупреждения перед запуском

#### Запрещено запускать несколько экземпляров бота одновременно
**Причина:** Telegram Bot API не поддерживает распределённую обработку обновлений через long polling. Если запущены два бота с одним токеном:
- Оба экземпляра будут получать одни и те же обновления
- Возникнут конфликты обработки сообщений
- Задачи могут создаваться дублями
- Поведение бота станет непредсказуемым

**Как избежать:**
- Перед запуском нового экземпляра убедитесь, что старый остановлен:
  ```bash
  docker-compose ps
  docker-compose down  # если нужно остановить существующий
  ```
- Для масштабирования используйте webhook вместо long polling (требуется доработка)

#### Блокировка Telegram в регионе
**Проблема:** В некоторых регионах (включая РФ) Telegram может быть заблокирован на уровне провайдера.

**Признаки:**
- Бот не получает обновления
- В логах ошибки соединения с `api.telegram.org`
- Таймауты при подключении

**Решения:**
1. **Выбрать сервер для хостинга в другом регионе:**
   - Европа: Netherlands, Germany, Finland
   - Азия: Kazakhstan, Armenia, Georgia
   - Примеры провайдеров: Hetzner, DigitalOcean, Vultr, Linode

2. **Использовать proxy для бота:**
   - Настроить SOCKS5 или HTTP proxy в контейнере
   - Добавить переменную окружения `HTTPS_PROXY`

3. **Использовать Telegram CDN (если доступно):**
   - Альтернативные эндпоинты для API

**Проверка доступности Telegram:**
```bash
docker-compose exec bot curl -I https://api.telegram.org
```
**Ожидаемый результат:** HTTP 200 OK

---

### Ситуация
Первый запуск или запуск после остановки

### Когда использовать
- Первоначальное развёртывание
- После обслуживания инфраструктуры
- После обновления кода

### Prerequisites
- Docker и Docker Compose установлены
- Файл `.env` настроен с необходимыми переменными
- Образ бота собран или доступен в registry

### Шаги

#### 1.1 Проверить переменные окружения
```bash
cat .env
```

**Ожидаемые переменные:**
- `TELEGRAM_BOT_TOKEN` — токен от @BotFather
- `TODOIST_API_TOKEN` — токен Todoist
- `DATABASE_URL` — PostgreSQL connection string
- `AI_PROVIDER` — провайдер AI (yandex/openrouter)

#### 1.2 Запустить стек
```bash
docker-compose up -d
```

#### 1.3 Проверить статус контейнеров
```bash
docker-compose ps
```

**Ожидаемый результат:**
- `jiraf-db` — статус `Up`
- `jiraf-bot` — статус `Up`

#### 1.4 Проверить логи бота
```bash
docker-compose logs bot
```

**Ожидаемый результат:**
```
Starting bot...
```

### Проверка результата
1. Отправить боту команду `/help` в Telegram
2. Получить список доступных команд

### Если что-то пошло не так
| Проблема | Решение |
|----------|---------|
| Container exited | Проверить логи: `docker-compose logs bot` |
| Ошибка подключения к БД | Проверить `DATABASE_URL` в `.env` |
| Ошибка токена Telegram | Проверить `TELEGRAM_BOT_TOKEN` |

---

## 2. Остановка бота

### Ситуация
Плановая остановка для обслуживания

### Шаги

#### 2.1 Остановить стек
```bash
docker-compose down
```

#### 2.2 Проверить остановку
```bash
docker-compose ps
```

**Ожидаемый результат:** Все контейнеры остановлены

### Сохранение данных
Данные базы данных сохраняются в volume `jiraf_db_data` и не будут потеряны.

---

## 3. Проверка состояния

### Ситуация
Мониторинг работоспособности бота

### Признаки проблем
- Бот не отвечает на команды
- Ошибки при создании задач
- Задержки в обработке сообщений

### Шаги

#### 3.1 Проверить статус контейнеров
```bash
docker-compose ps
```

#### 3.2 Проверить логи на ошибки
```bash
docker-compose logs --tail=100 bot | grep -i error
```

#### 3.3 Проверить подключение к БД
```bash
docker-compose exec db psql -U postgres -c "SELECT COUNT(*) FROM sessions WHERE status='open';"
```

#### 3.4 Проверить активные сессии
```bash
docker-compose exec db psql -U postgres -c "SELECT chat_id, started_at FROM sessions WHERE status='open';"
```

### Ожидаемый результат
- Все контейнеры в статусе `Up`
- В логах нет повторяющихся ошибок
- БД доступна и отвечает

---

## 4. Перезапуск бота

### Ситуация
Бот работает некорректно, требуется перезапуск

### Шаги

#### 4.1 Перезапустить контейнер бота
```bash
docker-compose restart bot
```

#### 4.2 Проверить логи после перезапуска
```bash
docker-compose logs --follow bot
```

#### 4.3 Проверить работоспособность
Отправить команду `/help` в Telegram

### Если не помогло
```bash
# Полная пересборка
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

---

## 5. Обновление конфигурации

### Ситуация
Изменение переменных окружения или настроек

### Шаги

#### 5.1 Изменить .env файл
```bash
nano .env
# или
vim .env
```

#### 5.2 Перезапустить бота
```bash
docker-compose up -d bot
```

#### 5.3 Проверить применение настроек
```bash
docker-compose exec bot env | grep TELEGRAM
```

---

## 6. Работа с базой данных

### 6.1 Подключение к БД

```bash
docker-compose exec db psql -U postgres -d jiraf
```

### 6.2 Проверка схемы
```sql
\dt
```

### 6.3 Проверка данных
```sql
-- Количество чатов
SELECT COUNT(*) FROM chats;

-- Активные сессии
SELECT * FROM sessions WHERE status = 'open';

-- Последние сообщения
SELECT * FROM messages ORDER BY ts DESC LIMIT 10;
```

### 6.4 Резервное копирование

#### Создать дамп
```bash
docker-compose exec db pg_dump -U postgres jiraf > backup_$(date +%Y%m%d_%H%M%S).sql
```

#### Восстановить из дампа
```bash
cat backup_20260328.sql | docker-compose exec -T db psql -U postgres jiraf
```

### 6.5 Очистка старых сессий

**Внимание!** Использовать только при необходимости.

```sql
-- Закрыть сессии старше 30 дней
UPDATE sessions 
SET status = 'closed', closed_at = NOW()
WHERE status = 'open' AND started_at < NOW() - INTERVAL '30 days';
```

---

## 7. Реагирование на инциденты

### 7.1 Бот не отвечает на команды

**Признаки:**
- Сообщения не обрабатываются
- `/help` не работает

**Шаги:**
1. Проверить статус контейнера
   ```bash
   docker-compose ps
   ```
2. Проверить логи
   ```bash
   docker-compose logs --tail=200 bot
   ```
3. Проверить подключение к Telegram API
   ```bash
   docker-compose exec bot curl -s https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getMe
   ```
4. Перезапустить бота
   ```bash
   docker-compose restart bot
   ```

### 7.2 Ошибки при создании задач в Todoist

**Признаки:**
- В логах ошибки Todoist API
- Задачи не создаются после подтверждения

**Шаги:**
1. Проверить токен Todoist
   ```bash
   docker-compose exec bot env | grep TODOIST
   ```
2. Проверить доступность Todoist API
   ```bash
   docker-compose exec bot curl -s https://api.todoist.com/rest/v2/projects \
     -H "Authorization: Bearer $TODOIST_API_TOKEN"
   ```
3. Проверить логи на конкретные ошибки
   ```bash
   docker-compose logs bot | grep -i todoist
   ```

### 7.3 Ошибки AI API

**Признаки:**
- В логах ошибки при суммаризации
- `/create_task` возвращает ошибку AI

**Шаги:**
1. Проверить переменные AI-провайдера
   ```bash
   docker-compose exec bot env | grep -E "AI_|YANDEX_|OPENROUTER_"
   ```
2. Проверить доступность API
   ```bash
   # Для OpenRouter
   docker-compose exec bot curl -s https://openrouter.ai/api/v1/auth/key \
     -H "Authorization: Bearer $OPENROUTER_API_KEY"
   ```
3. Временно отключить AI (если критично)
   - Изменить `.env`: `AI_PROVIDER=mock` (если реализовано)
   - Перезапустить бота

### 7.4 Переполнение базы данных

**Признаки:**
- Медленные запросы
- Ошибки записи

**Шаги:**
1. Проверить размер БД
   ```bash
   docker-compose exec db psql -U postgres -d jiraf -c \
     "SELECT pg_size_pretty(pg_database_size('jiraf'));"
   ```
2. Проверить количество записей
   ```sql
   SELECT 'messages' as table_name, COUNT(*) FROM messages
   UNION ALL
   SELECT 'sessions', COUNT(*) FROM sessions
   UNION ALL
   SELECT 'draft_tasks', COUNT(*) FROM draft_tasks;
   ```
3. Очистить старые данные (см. раздел 6.5)

---

## 8. Логирование и диагностика

### Просмотр логов

#### Последние строки
```bash
docker-compose logs --tail=100 bot
```

#### В реальном времени
```bash
docker-compose logs --follow bot
```

#### С фильтром по времени
```bash
docker-compose logs --since="2026-03-28T17:00:00" bot
```

#### Поиск ошибок
```bash
docker-compose logs bot | grep -iE "error|fatal|panic"
```

### Уровни логирования

В коде используются уровни:
- `INFO` — штатная работа
- `ERROR` — ошибки

### Экспорт логов
```bash
docker-compose logs bot > logs_$(date +%Y%m%d_%H%M%S).txt
```

---

## История изменений Runbook

| Дата | Изменение | Автор |
|------|-----------|-------|
| 2026-03-28 | Первоначальная версия | Команда jiraF |
