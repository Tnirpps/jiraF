# jiraF

A Telegram bot that integrates with Todoist for task
## Run with Docker

```bash
docker-compose up -d
```

## Project Structure

- `cmd/bot/main.go`: Application entry point
- `internal/bot/`: Bot core implementation
- `internal/commands/`: Command implementations
- `internal/todoist/`: Todoist API client
