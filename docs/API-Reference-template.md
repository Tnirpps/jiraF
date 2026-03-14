
! Пишет Разработчик

# API Reference: [Название сервиса]

> **Что это?** Контракт вашего API для потребителей.  
> Документ должен ответить на вопрос: «Как мне вызвать ваш сервис?»  
> Пиши для разработчика другой команды, который видит ваш API впервые.

---

| Поле | Значение |
|---|---|
| **Версия API** | v1 |
| **Base URL (prod)** | `https://api.wb.ru/your-service/v1` |
| **Base URL (staging)** | `https://api-staging.wb.ru/your-service/v1` |
| **Swagger/OpenAPI** | [ссылка](https://...) |
| **Обновлено** | ГГГГ-ММ-ДД |

---

## Аутентификация

> Как клиент должен аутентифицироваться?

**Тип:** Bearer Token / API Key / mTLS / internal (без авторизации)

```http
Authorization: Bearer <token>
```

> Где получить токен: _внутренняя документация WB Auth_

---

## Общие правила

| Параметр | Значение |
|---|---|
| Формат запроса | `application/json` |
| Формат ответа | `application/json` |
| Кодировка | UTF-8 |
| Таймаут | 30 секунд |

### Коды ответов

| Код | Значение |
|---|---|
| 200 OK | Успех |
| 201 Created | Ресурс создан |
| 400 Bad Request | Ошибка в запросе (валидация) |
| 401 Unauthorized | Не аутентифицирован |
| 403 Forbidden | Нет прав |
| 404 Not Found | Ресурс не найден |
| 409 Conflict | Конфликт (дубль) |
| 429 Too Many Requests | Rate limit превышен |
| 500 Internal Server Error | Ошибка сервера |

### Формат ошибки

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Поле 'name' обязательно",
    "details": [
      { "field": "name", "message": "не может быть пустым" }
    ]
  }
}
```

---

## Эндпоинты

### GET /items

Получить список элементов.

**Query параметры:**

| Параметр | Тип | Обязательный | Описание |
|---|---|---|---|
| `page` | integer | нет | Номер страницы (default: 1) |
| `limit` | integer | нет | Размер страницы (default: 20, max: 100) |
| `filter` | string | нет | Фильтр по _полю_ |

**Пример запроса:**

```http
GET /items?page=1&limit=20
Authorization: Bearer <token>
```

**Пример ответа (200 OK):**

```json
{
  "items": [
    {
      "id": "123",
      "name": "Название",
      "created_at": "2026-01-15T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150
  }
}
```

---

### POST /items

Создать новый элемент.

**Тело запроса:**

```json
{
  "name": "string, обязательно, max 255 символов",
  "description": "string, опционально",
  "type": "TYPE_A | TYPE_B"
}
```

| Поле | Тип | Обязательное | Валидация |
|---|---|---|---|
| `name` | string | да | 1–255 символов |
| `description` | string | нет | max 1000 символов |
| `type` | enum | да | `TYPE_A`, `TYPE_B` |

**Пример запроса:**

```http
POST /items
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Новый элемент",
  "type": "TYPE_A"
}
```

**Пример ответа (201 Created):**

```json
{
  "id": "456",
  "name": "Новый элемент",
  "type": "TYPE_A",
  "created_at": "2026-03-01T12:00:00Z"
}
```

**Возможные ошибки:**

| Код | Когда возникает |
|---|---|
| 400 | Не прошла валидация (`name` пустой, `type` неверный) |
| 409 | Элемент с таким `name` уже существует |

---

### GET /items/{id}

Получить элемент по ID.

**Path параметры:**

| Параметр | Тип | Описание |
|---|---|---|
| `id` | string | ID элемента |

**Пример ответа (200 OK):**

```json
{
  "id": "456",
  "name": "Новый элемент",
  "type": "TYPE_A",
  "created_at": "2026-03-01T12:00:00Z"
}
```

---

### DELETE /items/{id}

Удалить элемент.

**Пример ответа (200 OK):**

```json
{
  "success": true
}
```

---

## Rate Limiting

| Уровень | Лимит |
|---|---|
| Глобальный | 1000 RPS |
| На пользователя | 100 RPS |

При превышении: `429 Too Many Requests` + заголовок `Retry-After: <seconds>`

---

## Changelog

| Версия | Дата | Изменения |
|---|---|---|
| v1.0 | 2026-01-01 | Первый релиз |
