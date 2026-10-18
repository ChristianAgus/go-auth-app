# auth-app

A small REST API in Go with user authentication (register, login, JWT) and a per-user notes CRUD. Built with a layered structure (handler → service → repository) and SQL migrations.

## Features

- Register and login with bcrypt-hashed passwords
- JWT authentication (Bearer token) with a protected `/api/me` endpoint
- Notes CRUD where every user can only access their own notes
- Search and pagination on the notes list
- Database migrations with [goose](https://github.com/pressly/goose)
- Config via `.env`, connection pooling, and graceful shutdown
- Health check endpoint

## Tech Stack

- Go 1.22+ (uses the standard `net/http` router)
- MySQL 8 / MariaDB
- `database/sql` with `go-sql-driver/mysql`
- `golang-jwt/jwt`, `golang.org/x/crypto/bcrypt`, `pressly/goose`

## Project Structure

```
auth-app/
├── cmd/
│   ├── api/            # HTTP server entry point
│   └── migrate/        # migration runner
├── internal/
│   ├── config/         # environment config
│   ├── database/       # MySQL connection and migration runner
│   ├── model/          # data structs
│   ├── repository/     # SQL queries
│   ├── service/        # business logic
│   ├── handler/        # HTTP handlers
│   └── middleware/     # auth middleware
├── migrations/         # SQL migration files (embedded in the binary)
└── docs/postman/       # Postman collection
```

## Getting Started

### 1. Prerequisites

- [Go](https://go.dev/dl/) 1.22 or newer
- A running MySQL or MariaDB server

### 2. Clone and configure

```bash
git clone https://github.com/USERNAME/auth-app.git
cd auth-app
cp .env.example .env
```

Edit `.env` and set your database credentials and a strong `JWT_SECRET` (at least 32 characters).

| Variable | Description | Default |
|---|---|---|
| `APP_PORT` | Server port | `8080` |
| `DB_HOST` | Database host | `127.0.0.1` |
| `DB_PORT` | Database port | `3306` |
| `DB_USER` | Database user | `root` |
| `DB_PASSWORD` | Database password | *(empty)* |
| `DB_NAME` | Database name | `auth_app` |
| `JWT_SECRET` | Secret used to sign tokens (min. 32 chars) | *(required)* |
| `JWT_EXPIRY_HOURS` | Token lifetime in hours | `24` |

### 3. Create the database

```sql
CREATE DATABASE auth_app CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 4. Run migrations and start the server

```bash
go mod tidy
go run ./cmd/migrate up
go run ./cmd/api
```

The server runs at `http://localhost:8080`. Check it with `GET /health`.

## Migrations

| Command | Description |
|---|---|
| `go run ./cmd/migrate up` | Apply all pending migrations |
| `go run ./cmd/migrate down` | Roll back the last migration |
| `go run ./cmd/migrate reset` | Roll back all migrations |
| `go run ./cmd/migrate status` | Show migration status |

To change the schema, add a new file in `migrations/` (for example `00003_add_column.sql`) instead of editing existing ones.

## API Endpoints

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/health` | No | Health check |
| POST | `/api/auth/register` | No | Create an account |
| POST | `/api/auth/login` | No | Log in and receive a token |
| GET | `/api/me` | Yes | Current user profile |
| POST | `/api/notes` | Yes | Create a note |
| GET | `/api/notes` | Yes | List notes (`?q=`, `?page=`, `?limit=`) |
| GET | `/api/notes/{id}` | Yes | Get a note |
| PUT | `/api/notes/{id}` | Yes | Update a note |
| DELETE | `/api/notes/{id}` | Yes | Delete a note |

Protected endpoints require the header `Authorization: Bearer <token>`.

### Example

```bash
# Register
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"christian","email":"chris@mail.com","password":"rahasia123"}'

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"chris@mail.com","password":"rahasia123"}'

# Create a note
curl -X POST http://localhost:8080/api/notes \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"Learning Go","content":"JWT and migrations"}'
```

### Status Codes

| Code | Meaning |
|---|---|
| 200 / 201 / 204 | Success |
| 400 | Invalid input |
| 401 | Missing or invalid credentials or token |
| 404 | Resource not found (or not owned by the user) |
| 409 | Username or email already taken |
| 500 | Server error |

## Testing with Postman

Import `docs/postman/auth-app.postman_collection.json` into Postman. Tokens and note IDs are stored automatically after login and create requests.

## License

Add a license of your choice (for example MIT) in a `LICENSE` file.