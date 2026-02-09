# Image Processing Pipeline (System Design Practice)

Small, realistic system-design practice project focused on async pipelines, MQ, retries, idempotency, failure recovery, consistency, and observability. No fancy UI.

## Goals
- Asynchronous upload and processing
- Reliable task execution (crash-safe)
- Retry with backoff
- Idempotent processing
- Horizontal scaling for workers
- Observability (logs + metrics)

## Out of Scope
- Fancy UI
- Social features
- Search/recommendation/tagging
- Full auth system

## Architecture (High Level)
Client -> API (Go) -> Object Storage (MinIO/S3)
             -> RabbitMQ -> Workers (Go) -> Postgres

## Local Setup
Start all services:
```
docker compose up -d
```

## API (Minimal)
1. `POST /upload-url`
Request:
```json
{ "content_type": "image/jpeg", "file_name": "a.jpg" }
```
Response:
```json
{ "media_id": "01J...", "upload_url": "...", "original_key": "media/{id}/original.jpg" }
```

2. `POST /complete-upload`
```json
{ "media_id": "01J...", "original_key": "media/{id}/original.jpg" }
```

3. `GET /media/{media_id}`
```json
{ "media_id": "01J...", "status": "READY", "final_url": "..." }
```

## Metrics
- API: `GET /metrics` on port `8080`
- Worker: `GET /metrics` on port `9091`

## Testing Focus
- Happy path: upload -> processing -> READY
- Idempotency: repeated `complete-upload`
- Worker crash recovery
- ACK/DB ordering correctness

## Learning Plan
See `LEARNING_PLAN.md` for the structured course outline and progress tracking.

## Stack
Go, RabbitMQ, Postgres, MinIO, Docker Compose
