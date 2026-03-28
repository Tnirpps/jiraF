# ADR: jiraF "Telegram → Todoist" (MVP)

## Status
**Updated: March 2026** | **Original: Draft**

## Context
- **Боль**: Обсуждения задач в Telegram теряются; нет структурированного переноса в трекер. Нужен быстрый и минимально трудозатратный способ превращать чат-дискуссию в конкретную задачу на доске.
- **Цели**:
  - Собирать контекст переписки между `/start_discussion` и `/create_task`.
  - Автоматически формировать черновик задачи (title, description, due, priority, assignee).
  - Показывать предпросмотр с подтверждением и создавать задачу в Todoist.
  - Дать возможность корректировать черновик через внешний AI/ML-эндпоинт.
- **Затронутые компоненты/интеграции**:
  - Telegram Bot API (long polling).
  - Внешний Todoist API (создание задач).
  - Внешний AI API (суммаризация и редактирование черновика; YandexGPT или OpenRouter).
  - Внутренняя БД (чаты, сессии, сообщения, черновики, созданные задачи, аудит правок).

## Participants
- **Команда**:
  - UX/UI — дизайнер — Овчинников Дмитрий
  - QA — тестирование — Талалыкин Дмитрий
  - Бэкенд-разработка — Занозин Александр, Ткаченко Егор
  - Project Manager — Лемещенко Мария
  - Product Manager — Лемещенко Мария
- **Автор**: Команда разработки jiraF
- **Reviewers**: Технический лидер, Архитектор, DevOps инженер, QA инженер

## Decision / Solution

### Общая архитектура (MVP)

#### Telegram Bot Handler
- **Long polling** через `GetUpdatesChan` с таймаутом 60 секунд (реализация в `internal/bot/bot.go`, метод `Start()`).
- Команды: `/set_project`, `/start_discussion`, `/create_task`, `/cancel`, `/help`, `/start`.
- При активной сессии сохраняет все текстовые сообщения в БД.
- **Privacy mode**: бот видит сообщения в групповых чатах (требуется настройка через @BotFather).

#### Session Manager
- Инвариант: одна активная сессия на чат.
- `/start_discussion` закрывает старую и открывает новую сессию.
- `/cancel` закрывает текущую сессию.
- Сессия имеет владельца (`owner_id`) — только он может создать задачу по итогам обсуждения.

#### Message Store (DB)
- Хранит все сообщения без удаления (для воспроизводимости контекста и аудита).
- Индексы по `chat_id`, `session_id`, `ts`.

#### Draft Builder (AI)
- **Суммаризация**: вызов AI API (YandexGPT или OpenRouter) для анализа контекста.
- `extract_title`: короткий заголовок на основе суммаризации от AI.
- `extract_due`: RU-разговорные даты/время → нормализация в ISO (TZ=Europe/Moscow).
- `extract_priority`: словарь → Todoist priority (1..4).
- `extract_assignee_note`: по @упоминаниям и фразам ("назначь/ответственный/Иван" и т.п.).
- `summarize → description`: 3–5 ключевых предложений из контекста.
- Сохраняет результат в `draft_tasks`.

#### Preview & Callbacks
- `/create_task` строит черновик из сообщений активной сессии.
- Отправляет предпросмотр с кнопками:
  - `[✅ Подтвердить]` — создаёт задачу в Todoist
  - `[✏️ Редактировать]` — запрашивает инструкцию и вызывает AI для редактирования
  - `[❌ Отменить]` — отменяет создание задачи
- Идемпотентность callback-кнопок по ключу `chat_id + session_id`.
- Кнопки удаляются после нажатия (только если нажал владелец сессии).

#### Todoist Client
- Авторизация по `TODOIST_API_TOKEN`.
- Создание задачи в проекте, указанном в `chat_settings`.
- **Маппинг**:
  - `content` ← `draft.title`
  - `description` ← `draft.description + "\n\nПредлагаемый исполнитель: <assignee_note>"`
  - `priority` ← `draft.priority` (1..4)
  - `due_datetime` ← `draft.due_iso` (UTC ISO, если есть)
  - `assignee` — **не маппится автоматически** (требуется ручная корректировка в Todoist)
- По подтверждению пишет запись в `created_tasks` (task_id, url) и закрывает сессию.

#### AI Client (редактирование)
- По нажатию `[Редактировать]` бот запрашивает у пользователя одно сообщение-инструкцию.
- Вызов AI API с `{ draft, instruction, lang: "ru", timezone: "Europe/Moscow" }`.
- Валидация строгого JSON-ответа `{ title, description, due_date, priority, priority_text, labels }`.
- Обновляет `draft_tasks`, пишет аудит (в `audit_edits`), повторно показывает предпросмотр.
- При ошибке/невалидности — короткое понятное сообщение; исходный draft не меняется.

#### Error Handling & Idempotency
- Простые ретраи (1–2) при 5xx/сетевых сбоях Todoist/AI.
- Защита от повторных нажатий: проверка `created_tasks` по `session_id` до вызова Todoist.

#### Логирование и наблюдаемость
- INFO для ключевых шагов; ERROR для исключений.
- Логирование запросов к AI API с промптами.

#### Конфигурация (.env)
- `TELEGRAM_BOT_TOKEN` — токен бота от @BotFather
- `TODOIST_API_TOKEN` — токен Todoist
- `DATABASE_URL` — PostgreSQL connection string
- `APP_BASE_URL` — базовый URL приложения (для вебхуков, если будут)
- `TZ=Europe/Moscow` — часовой пояс по умолчанию
- `AI_PROVIDER` — провайдер AI (yandex или openrouter)
- `YANDEX_FOLDER_ID` — ID каталога Yandex Cloud (для YandexGPT)
- `OPENROUTER_MODEL` — модель OpenRouter (по умолчанию: openai/gpt-4o-mini)

### Данные/схема БД (PostgreSQL)

```sql
chats(id, created_at)
chat_settings(chat_id PK, todoist_project_id, updated_at)
sessions(id PK, chat_id, owner_id, status, started_at, closed_at)  -- owner_id добавлен для контроля доступа
messages(id PK, chat_id, session_id, message_id, user_id, username, text, ts)
draft_tasks(session_id PK, title, description, due_iso, priority, assignee_note, updated_at)
created_tasks(id PK, session_id, todoist_task_id, url, created_at)
audit_edits(id PK, session_id, instruction_text, diff_json, created_at)
```

**Индексы**: по `chat_id`, `session_id`, `ts` в соответствующих таблицах.

### Ключевые решения

| Решение | Обоснование |
|---------|-------------|
| **Long polling** вместо webhook | Проще для разработки и локального тестирования; webhook может быть добавлен позже |
| **AI API с поддержкой нескольких провайдеров** | Гибкость выбора; YandexGPT для RU-контекста, OpenRouter как альтернатива |
| **Таймзона Europe/Moscow** | По умолчанию для отображения дат; хранение в БД в UTC |
| **Идемпотентность callback-обработчиков** | Защита от дублей на уровне `created_tasks` |
| **Хранение всех сообщений** | Аудит, возможность повторной сборки черновика |
| **Владелец сессии (`owner_id`)** | Только автор обсуждения может создать задачу |

## Alternatives

### Получение апдейтов
| Вариант | Плюсы | Минусы | Выбор |
|---------|-------|--------|-------|
| **Webhook** | Масштабируемость, низкие задержки | Требует публичный URL, сложнее настройка | Отложен на потом |
| **Long polling** | Простота, не требует публичный URL | Задержки, менее масштабируемо | **Выбрано для MVP** |

### Хранение сессий
| Вариант | Плюсы | Минусы | Выбор |
|---------|-------|--------|-------|
| **In-memory cache/Redis** | Быстрый доступ | Нет трассируемости, сложность | Отклонено |
| **PostgreSQL** | Трассируемость, целостность | Медленнее кэша | **Выбрано** |

### AI/суммаризация
| Вариант | Плюсы | Минусы | Выбор |
|---------|-------|--------|-------|
| **Правила + эвристики** | Детерминированность, простота | Низкое качество | Отклонено |
| **Полная LLM-обработка** | Высокое качество | Стоимость, задержки | **Выбрано для MVP** |

## Constraints & Risks

### Технологические
- **RU-разговорные даты**: неоднозначности ("в следующий вторник", "к вечеру"). AI обрабатывает, но возможна неверная интерпретация.
- **Лимиты/ошибки Todoist/AI**: ретраи, таймауты, краткие сообщения об ошибках.
- **Privacy mode OFF**: бот видит сообщения — должен писать только при активной сессии.
- **AI API таймауты**: AI может отвечать долго; long polling Telegram имеет таймаут 60 секунд.

### Риски UX
- **Не назначаем assignee в Todoist** — может потребоваться ручная корректировка.
- **Возможная неверная интерпретация даты/приоритета** — компенсируем предпросмотром и режимом "Редактировать".
- **Стоимость AI-запросов** — каждый `/create_task` и редактирование стоят денег.

## Implementation Notes

### Реализовано (MVP)
- ✅ Команды `/set_project`, `/start_discussion`, `/create_task`, `/cancel`, `/help`
- ✅ Long polling для получения апдейтов
- ✅ Хранение сессий и сообщений в PostgreSQL
- ✅ AI-суммаризация через YandexGPT или OpenRouter
- ✅ Предпросмотр с inline-кнопками
- ✅ Редактирование черновика через AI
- ✅ Создание задачи в Todoist (базовая интеграция)
- ✅ Аудит правок (таблица `audit_edits`)

### Не реализовано / Отложено
- ⏳ Webhook вместо long polling
- ⏳ Автоматический маппинг assignee в Todoist (требует интеграции с пользователями Todoist)
- ⏳ Полная валидация и обработка ошибок AI API
- ⏳ Мониторинг и метрики (Prometheus, Grafana)

### Отображение времени
- Предпросмотр — MSK (Europe/Moscow)
- В БД — UTC ISO

## Appendix: Technology Stack

### Backend
- **Язык программирования**: Go 1.21
- **API интеграции**:
  - Telegram Bot API (`github.com/go-telegram-bot-api/telegram-bot-api/v5`)
  - Todoist API (REST v2)
  - AI API (YandexGPT или OpenRouter)
- **База данных**: PostgreSQL 14
- **HTTP клиент**: Стандартный `net/http` из Go stdlib с context поддержкой
- **Конфигурация**: Файлы `.env` (`github.com/joho/godotenv`), YAML (`gopkg.in/yaml.v3`)
- **Тестирование**: `github.com/stretchr/testify`

### Инфраструктура
- **Контейнеризация**: Docker + docker-compose
- **Базовый образ**: Alpine Linux

### Архитектура базы данных
Реляционная схема PostgreSQL с таблицами:
- `chats`, `chat_settings`
- `sessions` (с `owner_id` для контроля доступа)
- `messages`
- `draft_tasks`, `created_tasks`
- `audit_edits`

Индексы по ключевым полям (`chat_id`, `session_id`, `ts`).
Внешние ключи для обеспечения целостности данных.
