# Mini Admin — Go + PostgreSQL + Vanilla JS

Прототип административной панели с авторизацией (access/refresh tokens), CRUD пользователей, пагинацией, поиском, фильтрацией и сортировкой.

## Стек

- **Backend:** Go 1.22, стандартная библиотека `net/http`, `lib/pq`, `golang-jwt`, `bcrypt`
- **Frontend:** HTML + CSS + vanilla JavaScript (без фреймворков)
- **БД:** PostgreSQL 16
- **Инфраструктура:** Docker Compose

## Быстрый запуск (Docker)

```bash
cp .env.example .env
docker-compose up --build
```

Приложение будет доступно по адресу: **http://localhost:8080**

## Тестовый администратор

| Поле     | Значение            |
|----------|---------------------|
| Email    | admin@example.com   |
| Password | admin123            |

Создаётся автоматически при первом запуске (seed).  
Параметры можно изменить через `.env`.

## Запуск без Docker

Требования: Go 1.22+, PostgreSQL 16+

```bash
# 1. Создать БД
createdb miniadmin

# 2. Настроить переменные окружения
cp .env.example .env
# отредактировать .env

# 3. Экспортировать переменные
export $(cat .env | grep -v '^#' | xargs)

# 4. Запустить
go run ./cmd/server
```

## Структура проекта

```
mini-admin/
├── cmd/server/          # Точка входа
│   └── main.go
├── internal/
│   ├── config/          # Загрузка конфигурации из ENV
│   ├── database/        # Подключение, миграции, seed
│   ├── handlers/        # HTTP-обработчики (auth, users)
│   ├── middleware/       # JWT auth, CORS, проверка активности
│   ├── models/          # Модели и запросы к БД
│   └── utils/           # JSON-ответы, валидация
├── migrations/          # SQL-миграции
├── frontend/            # Статические файлы UI
│   ├── index.html
│   ├── css/style.css
│   └── js/
│       ├── api.js       # HTTP-клиент с автообновлением токена
│       └── app.js       # SPA-логика
├── docker-compose.yml
├── Dockerfile
├── .env.example
└── README.md
```

## API

### Авторизация

| Метод | Endpoint            | Описание                          | Auth |
|-------|---------------------|-----------------------------------|------|
| POST  | /api/auth/login     | Логин (email + password)          | —    |
| POST  | /api/auth/refresh   | Обновление access token           | —    |
| GET   | /api/auth/me        | Текущий пользователь              | ✓    |
| POST  | /api/auth/logout    | Выход (удаление refresh token)    | ✓    |

### Пользователи

| Метод  | Endpoint         | Описание             | Auth |
|--------|------------------|----------------------|------|
| GET    | /api/users       | Список (пагинация)   | ✓    |
| POST   | /api/users       | Создание             | ✓    |
| GET    | /api/users/{id}  | Просмотр             | ✓    |
| PUT    | /api/users/{id}  | Редактирование       | ✓    |
| DELETE | /api/users/{id}  | Удаление             | ✓    |

**Query-параметры списка:** `search`, `status`, `sort_by`, `order`, `page`, `per_page`

### Формат ошибок

```json
{
  "error": "Validation failed",
  "details": {
    "email": "Invalid email format",
    "password": "Password is required"
  }
}
```

## Авторизация (access + refresh tokens)

1. `POST /api/auth/login` → возвращает `access_token` (JWT, 15 мин) и `refresh_token` (opaque, 7 дней)
2. Access token передаётся в заголовке: `Authorization: Bearer <token>`
3. При истечении access token → `POST /api/auth/refresh` с refresh token → новая пара токенов (ротация)
4. Frontend автоматически обновляет токен при получении 401
5. При logout refresh token удаляется из БД

## Что реализовано

- [x] Авторизация (login / logout / me)
- [x] Access token (JWT) + Refresh token (opaque, в БД)
- [x] Ротация refresh token при обновлении
- [x] Проверка активности пользователя
- [x] CRUD пользователей
- [x] Валидация входных данных
- [x] Единый формат ошибок (JSON)
- [x] Пагинация
- [x] Поиск по name / email
- [x] Фильтрация по status
- [x] Сортировка по любому полю
- [x] Миграции (с отслеживанием применённых)
- [x] Seed тестового администратора
- [x] Frontend на vanilla JS (SPA без фреймворков)
- [x] Взаимодействие frontend ↔ backend только через API
