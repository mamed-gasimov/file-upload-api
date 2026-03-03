# file-service

Go 1.25 REST API for managing files with PostgreSQL 16, S3-compatible object storage (MinIO), RabbitMQ-based async AI analysis, and direct OpenAI integration.

Files are streamed directly to MinIO without disk buffering; metadata is persisted in PostgreSQL. After upload, an analysis request is published to RabbitMQ and processed asynchronously by **ai-service**, which returns a translation summary stored alongside the file metadata.

## Project structure

```
file-service/
├── cmd/server/main.go                           # entry point – wires everything together
├── internal/
│   ├── config/config.go                         # .env → Config struct (caarlos0/env)
│   ├── modules/
│   │   ├── files/
│   │   │   ├── model.go                         # File entity
│   │   │   ├── messages.go                      # RabbitMQ message types (AnalyzeRequest, AnalysisReply)
│   │   │   ├── repository.go                    # pgx database layer
│   │   │   ├── result_consumer.go               # RabbitMQ consumer for analysis results
│   │   │   ├── service.go                       # business logic (upload, delete, analyze)
│   │   │   └── handler.go                       # Echo HTTP handlers
│   │   └── analysis/
│   │       ├── analysis.go                      # Provider interface
│   │       └── openai/openai.go                 # OpenAI implementation (sync path)
│   ├── messaging/
│   │   ├── messaging.go                         # Publisher/Consumer interfaces
│   │   └── rabbitmq/rabbitmq.go                 # RabbitMQ client (publish + consume)
│   ├── server/server.go                         # Echo router + middleware
│   └── storage/
│       ├── storage.go                           # Storage interface (Upload, Download, Delete)
│       └── minio/minio.go                       # MinIO implementation (streaming)
├── migrations/
│   ├── 001_create_files.sql                     # initial schema
│   ├── 002_add_resume_column.sql                # adds resume (AI summary) column
│   └── 003_add_translation_summary_column.sql   # adds translation_summary column (async path)
├── docs/
│   └── openapi.yaml                             # OpenAPI 3.0.3 specification
├── docker-compose.yml                           # PostgreSQL 16 + MinIO + RabbitMQ
├── .env.example                                 # environment variable template
└── go.mod
```

## Quick start

```bash
# 1. Start infrastructure
docker compose up -d

# 2. Copy env (adjust if needed)
cp .env.example .env

# 3. Run the server (migrations run automatically)
go run ./cmd/server
```

The server starts on `http://localhost:8080`.

## Configuration

All settings are loaded from environment variables (or an `.env` file via godotenv).
See `.env.example` for the full list.

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8080` | HTTP server port |
| `POSTGRES_HOST` | `localhost` | PostgreSQL host |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_USER` | `fileuser` | PostgreSQL user |
| `POSTGRES_PASSWORD` | `filepass` | PostgreSQL password |
| `POSTGRES_DB` | `filedb` | PostgreSQL database name |
| `POSTGRES_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `MINIO_ENDPOINT` | `localhost:9000` | MinIO API endpoint |
| `MINIO_ACCESS_KEY` | `minioadmin` | MinIO access key |
| `MINIO_SECRET_KEY` | `minioadmin` | MinIO secret key |
| `MINIO_BUCKET` | `files` | MinIO bucket name |
| `MINIO_USE_SSL` | `false` | Use SSL for MinIO |
| `OPENAI_API_KEY` | — | OpenAI API key (required) |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1/` | OpenAI-compatible API base URL |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection URL |

## API

Full OpenAPI 3.0 specification is available in [`docs/openapi.yaml`](docs/openapi.yaml).

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/files` | List all uploaded files (newest first) |
| `POST` | `/api/files` | Upload a file (multipart/form-data, field `file`) |
| `POST` | `/api/files/:id/analyze` | Trigger async AI analysis of a file |
| `DELETE` | `/api/files/:id` | Delete a file by ID |

### Upload

```bash
curl -X POST http://localhost:8080/api/files \
  -F "file=@./myfile.pdf"
```

Response `201 Created`:

```json
{
  "id": 1,
  "name": "myfile.pdf",
  "size": 204800,
  "mime_type": "application/pdf",
  "object_key": "2026/02/16/550e8400-e29b-41d4-a716-446655440000_myfile.pdf",
  "created_at": "2026-02-16T12:00:00Z",
  "updated_at": "2026-02-16T12:00:00Z",
  "resume": null,
  "translation_summary": null
}
```

### List

```bash
curl http://localhost:8080/api/files
```

Response `200 OK`: JSON array of file objects.

### Analyze

```bash
curl -X POST http://localhost:8080/api/files/1/analyze
```

Response `200 OK`:

```json
{
  "id": 1,
  "name": "myfile.pdf",
  "size": 204800,
  "mime_type": "application/pdf",
  "object_key": "2026/02/16/550e8400-e29b-41d4-a716-446655440000_myfile.pdf",
  "created_at": "2026-02-16T12:00:00Z",
  "updated_at": "2026-02-16T12:05:00Z",
  "resume": null,
  "translation_summary": null
}
```

The endpoint downloads the file from MinIO, sends its content to OpenAI synchronously, and stores the result in the `resume` column in PostgreSQL.

The async RabbitMQ flow is triggered automatically on **upload** (`POST /api/files`): an `AnalyzeRequest` message is published to the `file.analyze` queue. **ai-service** processes it asynchronously, sends the content to OpenAI (GPT-4o Mini), and publishes the result back to `file.analysis.result`. This service consumes the result and updates the `translation_summary` column in PostgreSQL.

### Delete

```bash
curl -X DELETE http://localhost:8080/api/files/1
```

Response `204 No Content` on success.

## Architecture

### Sync (HTTP path)

```
HTTP request
  → Echo (Logger, Recover, CORS middleware)
    → FileHandler
      → FileService
        → FileRepository (PostgreSQL via pgx)
        → Storage → MinIO

POST /api/files/:id/analyze
  → FileService.AnalyzeFile()
    → Storage → MinIO (download)
    → OpenAI (FileResume)
    → FileRepository.UpdateResume() → PostgreSQL
```

### Async (RabbitMQ path)

```
POST /api/files
  → FileService.UploadFile()
    → Publish AnalyzeRequest → [file.analyze queue]
                                      ↓
                               ai-service (worker)
                                      ↓
                        Publish AnalysisReply → [file.analysis.result queue]
                                      ↓
                       result_consumer.go (goroutine)
                         → FileRepository.UpdateTranslationSummary()
```

### Key design points

- **Streaming uploads** — files are piped directly from the HTTP request to MinIO, avoiding temporary disk writes.
- **Async analysis** — analysis requests are non-blocking; file upload succeeds regardless of broker availability.
- **Manual ACK** — the result consumer acknowledges messages only after a successful database update.
- **Automatic migrations** — [goose](https://github.com/pressly/goose) runs pending SQL migrations on startup.
- **Graceful shutdown** — the server handles `SIGINT`/`SIGTERM` and drains connections with a 10-second timeout.
- **Best-effort cleanup** — if saving metadata fails after a successful MinIO upload, the object is deleted from MinIO.

## RabbitMQ message contracts

Both queues are declared `durable` with `persistent` delivery mode to survive broker restarts.

### `file.analyze` (published by file-service, consumed by ai-service)

```json
{
  "file_id": 1,
  "object_key": "2026/02/16/uuid_myfile.pdf",
  "content_type": "application/pdf",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### `file.analysis.result` (published by ai-service, consumed by file-service)

```json
{
  "file_id": 1,
  "translation_summary": "This document describes...",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "error": ""
}
```

## Database schema

```sql
CREATE TABLE files (
    id                  BIGSERIAL    PRIMARY KEY,
    name                TEXT         NOT NULL,
    size                BIGINT       NOT NULL DEFAULT 0,
    mime_type           TEXT         NOT NULL DEFAULT 'application/octet-stream',
    object_key          TEXT         NOT NULL UNIQUE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    resume              TEXT,                          -- sync OpenAI path (nullable)
    translation_summary TEXT                           -- async ai-service path (nullable)
);
```

## Infrastructure (Docker Compose)

`docker compose up -d` starts three containers:

| Service | Image | Ports |
|---|---|---|
| PostgreSQL | `postgres:16` | `5432` (mapped to `5433` on host) |
| MinIO | `minio/minio:latest` | `9000` (API), `9001` (Console) |
| RabbitMQ | `rabbitmq:3.13-management` | `5672` (AMQP), `15672` (Management UI) |

Data is persisted in local volumes (`pg_data/`, `minio_data/`, `rabbitmq_data/`).

## Dead Letter Exchange (DLX) — not yet implemented

Currently, messages that fail during processing (unmarshal error, DB update failure) are acknowledged and silently dropped. To add DLX/DLQ support, see the [ai-service README](../ai-service/README.md#dead-letter-exchange-dlx--dead-letter-queue-dlq) for a full implementation guide that applies equally to both services.

## Dependencies

| Package | Purpose |
|---|---|
| [echo/v4](https://github.com/labstack/echo) | HTTP framework |
| [pgx/v5](https://github.com/jackc/pgx) | PostgreSQL driver & connection pool |
| [minio-go/v7](https://github.com/minio/minio-go) | S3-compatible object storage client |
| [amqp091-go](https://github.com/rabbitmq/amqp091-go) | RabbitMQ AMQP client |
| [openai-go](https://github.com/openai/openai-go) | OpenAI API client |
| [goose/v3](https://github.com/pressly/goose) | Database migrations |
| [caarlos0/env](https://github.com/caarlos0/env) | Struct-based env var parsing |
| [godotenv](https://github.com/joho/godotenv) | `.env` file loader |
| [google/uuid](https://github.com/google/uuid) | UUID generation for object keys |
