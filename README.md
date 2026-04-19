# jiraF

Telegram-бот для интеграции с Todoist, преобразующий обсуждения в чате в структурированные задачи.

## 📖 Документация

| Документ | Описание |
|----------|----------|
| [ADR.md](ADR.md) | Архитектурные решения и обоснования |
| [RUNBOOK.md](RUNBOOK.md) | Инструкции по эксплуатации и реагированию на инциденты |
| [DECISION_LOG.md](DECISION_LOG.md) | Журнал ключевых решений проекта |
| [GLOSSARY.md](GLOSSARY.md) | Словарь терминов и сокращений |

---

## Возможности

- **Сессии обсуждения** — сбор контекста переписки между `/start_discussion` и `/create_task`
- **AI-суммаризация** — автоматическое формирование черновика задачи (заголовок, описание, срок, приоритет)
- **Предпросмотр** — подтверждение или редактирование черновика перед созданием задачи
- **Todoist интеграция** — создание задач в указанном проекте
- **История сообщений** — хранение в PostgreSQL для аудита и воспроизводимости

---

## Быстрый старт

### 1. Настройка окружения

```bash
cp .env.example .env
```

**Требуемые переменные:**

| Переменная | Описание |
|------------|----------|
| `TELEGRAM_BOT_TOKEN` | Токен от [@BotFather](https://t.me/BotFather) |
| `TODOIST_API_TOKEN` | Токен из [Todoist Integrations](https://todoist.com/app/settings/integrations) |
| `DATABASE_URL` | PostgreSQL connection string |
| `AI_PROVIDER` | Провайдер AI: `yandex` или `openrouter` |

### 2. Запуск

```bash
docker-compose up -d
```

Запустятся:
- PostgreSQL с необходимой схемой
- Telegram-бот

### 3. Проверка

```bash
docker-compose ps
docker-compose logs bot
```

Отправьте `/help` боту в Telegram.

---

## Команды

| Команда | Описание |
|---------|----------|
| `/start` | Начало работы с ботом |
| `/help` | Список доступных команд |
| `/set_project <id|url>` | Установить Todoist-проект для чата |
| `/start_discussion` | Начать сбор сообщений |
| `/cancel` | Отменить текущее обсуждение |
| `/create_task` | Создать задачу из обсуждения |

---

## ⚠️ Важные ограничения

1. **Нельзя запускать два экземпляра бота** с одним токеном (конфликт long polling)
2. **Telegram может быть заблокирован** в некоторых регионах — выбирайте сервер в другой локации (EU, Asia)

Подробности: [RUNBOOK.md](RUNBOOK.md#1-запуск-бота)

---

## Архитектура

**Стек:**
- Go 1.21
- PostgreSQL 14
- Telegram Bot API (long polling)
- Todoist API (REST v2)
- AI API (YandexGPT / OpenRouter)

**Схема БД:**
- `chats`, `chat_settings`
- `sessions` (с `owner_id` для контроля доступа)
- `messages`
- `draft_tasks`, `created_tasks`
- `audit_edits`

Подробности: [ADR.md](ADR.md)

---

## Разработка

### Тесты

```bash
go test ./...
```

### Структура проекта

```
jiraF/
├── cmd/bot/main.go        # Точка входа
├── internal/
│   ├── bot/               # Ядро бота
│   ├── commands/          # Обработчики команд
│   ├── ai/                # AI-клиент (YandexGPT, OpenRouter)
│   ├── todoist/           # Todoist API клиент
│   ├── db/                # Модели и репозиторий БД
│   └── httpclient/        # HTTP-клиент для внешних API
└── configs/               # Конфигурационные файлы
```

---

## TODO

### Features
- [ ] Webhook вместо long polling (production)
- [ ] Автоматический маппинг assignee в Todoist
- [ ] Полная валидация ошибок AI API

### Improvements
- [ ] Улучшенный парсинг RU-дат с AI
- [ ] Шаблоны задач для типовых обсуждений
- [ ] Поддержка вложений (файлы, изображения)
- [ ] Привести пользовательские текстовки и сообщения бота к единому короткому стилю

### DevOps
- [ ] CI/CD пайплайн
- [ ] Автоматические тесты с coverage
