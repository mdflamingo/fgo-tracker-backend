# Roadmap fgo-tracker-backend

## Приоритет 1: Фундамент

- [ ] Убрать мёртвый код (комментарии subscriptions, `fmt.Println(uuid)`)
- [ ] Исправить `.env`/config несоответствие (`RUN_ADDR` vs `RUN_ADDRESS`)
- [ ] Добавить валидацию DTO (`go-playground/validator`)
- [ ] Стандартные ошибки — единый JSON-формат `{"error": "...", "code": 400}`
- [ ] Recovery middleware (`chi.Recoverer`)

## Приоритет 2: CRUD операции

- [ ] Update task (`PUT /api/task/{id}`) — модель, репозиторий, обработчик
- [ ] Delete task (`DELETE /api/task/{id}`) — каскадное удаление через FK
- [ ] List tasks (`GET /api/task/list`) — пагинация, фильтрация, `GetList()`

## Приоритет 3: Auth

- [ ] User registration + login (`POST /api/user/register`, `POST /api/user/login`) → JWT
- [ ] JWT middleware — извлекает user ID из токена, передаёт в контекст
- [ ] Убрать хардкод creator ID (`task_service.go:166`)
- [ ] User/Project CRUD — хендлеры для уже существующих таблиц

## Приоритет 4: Вложения

- [ ] Attachment upload — handler + storage (локальный или S3)
- [ ] GET attachments for task — в TaskResponse или отдельный эндпоинт

## Приоритет 5: Инфраструктура

- [ ] CORS middleware
- [ ] Dockerfile — multi-stage build
- [ ] docker-compose с app-сервисом
- [ ] README — setup, env vars, API overview

## Приоритет 6: Качество

- [ ] Unit-тесты репозитория (testcontainers)
- [ ] Integration-тесты хендлеров (httptest + мок репозитория)
- [ ] CI/CD — GitHub Actions: lint, test, build, Docker push

## Приоритет 7 (опционально): GraphQL

- [ ] gqlgen + схема (параллельно с REST)
- [ ] Резолверы — Task, User, Project
