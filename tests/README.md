# Тест-кейсы jiraF

Ручные тест-кейсы для Telegram-бота. Каждый файл - отдельная область.

Формат каждого кейса: предусловия, шаги (только действия), итоговый ожидаемый результат.

Для тестов с несколькими пользователями нужны два Telegram-аккаунта в одном групповом чате.

---

| Тег | Файл | Область |
|-----|------|---------|
| TC-ST | [01_start.md](features/01_start.md) | `/start` - онбординг |
| TC-HLP | [02_help.md](features/02_help.md) | `/help` |
| TC-SP | [03_set_project.md](features/03_set_project.md) | `/set_project` - выбор проекта |
| TC-SD | [04_start_discussion.md](features/04_start_discussion.md) | `/start_discussion` |
| TC-MSG | [05_messages.md](features/05_messages.md) | Сохранение сообщений в сессии |
| TC-CT | [06_create_task.md](features/06_create_task.md) | `/create_task` - инициация черновика |
| TC-TMPL | [07_task_templates.md](features/07_task_templates.md) | Типы задач и шаблонные поля |
| TC-PRI | [08_priority.md](features/08_priority.md) | Приоритет задачи |
| TC-DT | [09_due_date.md](features/09_due_date.md) | Срок выполнения - парсинг дат |
| TC-LNK | [10_links.md](features/10_links.md) | Ссылки - извлечение и нормализация |
| TC-ASGN | [11_assignee.md](features/11_assignee.md) | Исполнитель задачи |
| TC-CB | [12_callbacks.md](features/12_callbacks.md) | Callbacks: Подтвердить / Редактировать / Отменить |
| TC-ED | [13_edit_flow.md](features/13_edit_flow.md) | Edit flow - редактирование через reply |
| TC-CL | [14_cancel.md](features/14_cancel.md) | `/cancel` - завершение обсуждения |
| TC-LS | [15_list.md](features/15_list.md) | `/list` - задачи и проекты |
| TC-FMT | [16_formatting.md](features/16_formatting.md) | Форматирование preview и Todoist-контента |
| TC-PA | [17_pending_action.md](features/17_pending_action.md) | Управление action-сообщением |
| TC-EDGE | [18_edge_cases.md](features/18_edge_cases.md) | Граничные случаи, изоляция чатов, устойчивость |
