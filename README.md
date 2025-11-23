# Semantic Video Daemon

API server for managing frame extraction and cloud upload state.

## Prerequisites
- Go 1.22+
- FFmpeg installed (required for frame extraction)

## Setup
1) Install dependencies:
```bash
go mod tidy
```
2) (Optional) Set `FRAMES_ROOT` to change where extracted frames are stored (defaults to `frames/`).

## Run the daemon
```bash
go run cmd/daemon/main.go
```
The server listens on `:8080`.

## Swagger / OpenAPI docs
1) Generate docs (after updating handlers or models):
```bash
swag init -g cmd/daemon/main.go -o internal/docs
```
2) Start the server and open the Swagger UI:
```
http://localhost:8080/swagger
```
The UI serves `doc.json` from `internal/docs` via `http-swagger`.
