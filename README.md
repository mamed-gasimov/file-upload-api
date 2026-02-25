# file-service

Go 1.25 REST API for file uploads with PostgreSQL 16 (pgx), S3-compatible object storage (MinIO), and AI-powered file analysis (OpenAI).
Files are streamed directly to MinIO without disk buffering; metadata is persisted in PostgreSQL.
Uploaded files can be analyzed via OpenAI to generate a brief summary stored alongside the metadata.

## Project structure

```
file-service/
├── cmd/server/main.go                       # entry point – wires everything together
├── internal/
│   ├── config/config.go                     # .env → Config struct (caarlos0/env)
│   ├── modules/
│   │   ├── files/
│   │   │   ├── model.go                     # File entity
│   │   │   ├── repository.go               # pgx database layer
│   │   │   ├── service.go                   # business logic (upload, delete, analyze)
│   │   │   └── handler.go                   # Echo HTTP handlers
│   │   └── analysis/
│   │       ├── analysis.go                  # Provider interface
│   │       └── openai/openai.go             # OpenAI implementation
│   ├── server/server.go                     # Echo router + middleware
│   └── storage/
│       ├── storage.go                       # Storage interface (Upload, Download, Delete)
│       └── minio/minio.go                   # MinIO implementation (streaming)
├── migrations/
│   ├── 001_create_files.sql                 # initial schema
│   └── 002_add_resume_column.sql            # adds resume (AI summary) column
├── docs/
│   └── openapi.yaml                         # OpenAPI 3.0.3 specification
├── docker-compose.yml                       # PostgreSQL 16 + MinIO
├── .env.example                             # environment variable template
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

## API

Full OpenAPI 3.0 specification is available in [`docs/openapi.yaml`](docs/openapi.yaml).

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/files` | List all uploaded files (newest first) |
| POST | `/api/files` | Upload a file (multipart/form-data, field `file`) |
| POST | `/api/files/:id/analyze` | Analyze a file with AI and store a summary |
| DELETE | `/api/files/:id` | Delete a file by ID |

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
  "resume": null
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
  "resume": "This PDF document contains ... (AI-generated summary)"
}
```

The file content is downloaded from MinIO, truncated to 100 000 characters, and sent to OpenAI (GPT-4o Mini) which returns a 2–3 sentence summary. The summary is persisted in the `resume` column.

### Delete

```bash
curl -X DELETE http://localhost:8080/api/files/1
```

Response `204 No Content` on success.

## Architecture

```
HTTP request
  → Echo (Logger, Recover, CORS middleware)
    → FileHandler
      → FileService
        → FileRepository (PostgreSQL via pgx)
        → Storage interface → MinIO implementation
        → Analysis Provider → OpenAI implementation
```

- **Streaming uploads** — files are piped directly from the HTTP request to MinIO, avoiding temporary files on disk.
- **AI-powered analysis** — uploaded files can be analyzed on demand via the OpenAI API; summaries are stored in PostgreSQL.
- **Automatic migrations** — [goose](https://github.com/pressly/goose) runs pending SQL migrations on startup.
- **Graceful shutdown** — the server handles SIGINT/SIGTERM and drains connections with a 10-second timeout.
- **Best-effort cleanup** — if saving metadata fails after a successful upload, the object is deleted from MinIO.

## Infrastructure (Docker Compose)

`docker compose up -d` starts two containers:

| Service | Image | Ports |
|---|---|---|
| PostgreSQL | `postgres:16` | `5432` |
| MinIO | `minio/minio:latest` | `9000` (API), `9001` (Console) |

Data is persisted in local volumes (`pg_data/` and `minio_data/`).

## Dependencies

| Package | Purpose |
|---|---|
| [echo/v4](https://github.com/labstack/echo) | HTTP framework |
| [pgx/v5](https://github.com/jackc/pgx) | PostgreSQL driver & connection pool |
| [minio-go/v7](https://github.com/minio/minio-go) | S3-compatible object storage client |
| [openai-go](https://github.com/openai/openai-go) | OpenAI API client for file analysis |
| [goose/v3](https://github.com/pressly/goose) | Database migrations |
| [caarlos0/env](https://github.com/caarlos0/env) | Struct-based env var parsing |
| [godotenv](https://github.com/joho/godotenv) | `.env` file loader |
| [google/uuid](https://github.com/google/uuid) | UUID generation for object keys |
