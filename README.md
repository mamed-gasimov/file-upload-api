# file-service

Go 1.25 REST API for file uploads with PostgreSQL 16 (pgx) and S3-compatible object storage (MinIO).

## Project structure

```
file-service/
├── cmd/server/main.go          # entry point – wires everything together
├── internal/
│   ├── config/config.go        # .env → Config struct
│   ├── model/file.go           # File entity
│   ├── repository/             # pgx database layer
│   ├── storage/
│   │   ├── storage.go          # Storage interface
│   │   └── minio/minio.go      # MinIO implementation (streaming)
│   ├── handler/file_handler.go # Echo HTTP handlers
│   └── server/server.go        # Echo router + middleware
├── migrations/                 # SQL migrations
├── docker-compose.yml          # PostgreSQL 16 + MinIO
├── .env / .env.example         # environment variables
└── go.mod
```

## Quick start

```bash
# 1. Start infrastructure
docker compose up -d

# 2. Copy env (adjust if needed)
cp .env.example .env

# 3. Run the server
go run ./cmd/server

# Server starts on http://localhost:8080
```

## API

| Method | Endpoint          | Description             |
|--------|-------------------|-------------------------|
| GET    | `/api/files`      | List all uploaded files |
| POST   | `/api/files`      | Upload a file (multipart, field `file`) |
| DELETE | `/api/files/:id`  | Delete a file by ID     |

### Upload example

```bash
curl -X POST http://localhost:8080/api/files \
  -F "file=@./myfile.pdf"
```

### List example

```bash
curl http://localhost:8080/api/files
```

### Delete example

```bash
curl -X DELETE http://localhost:8080/api/files/1
```
