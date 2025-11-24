# jiraF

A Telegram bot that integrates with Todoist for task management. It provides a discussion workflow that collects messages and creates tasks from them.

## Features

- Set a Todoist project for each chat
- Start discussion sessions to collect messages
- Create tasks from discussion context
- Store message history in PostgreSQL database

## Setup

### Environment Variables

Copy the example environment file and fill in your values:

```bash
cp .env.example .env
```

Required variables:
- `TELEGRAM_BOT_TOKEN`: Get from [@BotFather](https://t.me/BotFather)
- `TODOIST_API_TOKEN`: Get from [Todoist Integrations](https://todoist.com/app/settings/integrations)
- `DATABASE_URL`: PostgreSQL connection string

### Run with Docker

The easiest way to run the bot is with Docker Compose:

```bash
docker-compose up -d
```

This will start:
- PostgreSQL database with the required schema
- Telegram bot connected to the database

## Available Commands

- `/start` - Get started with the bot
- `/help` - Display available commands
- `/set_project <id|url>` - Set Todoist project for the chat
- `/start_discussion` - Start collecting messages
- `/cancel` - Cancel current discussion

## Project Structure

- `cmd/bot/main.go`: Application entry point
- `internal/bot/`: Bot core implementation
- `internal/commands/`: Command implementations
- `internal/todoist/`: Todoist API client
- `internal/db/`: Database models and operations

## Development

### Running Tests

```bash
go test ./...
```

### Database Schema

The database schema is defined in `internal/db/schema.sql` and includes tables for:
- Chats and their settings
- Discussion sessions
- Messages within sessions
- Tasks created from discussions
