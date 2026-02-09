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

## Quickstart (5 min)
1. Get upload URL:
```
curl -s -X POST http://localhost:8080/upload-url \
  -H "Content-Type: application/json" \
  -d '{"content_type":"image/jpeg","file_name":"test.jpg"}'
```
2. Upload file (use the returned `upload_url`):
```
curl -X PUT -H "Content-Type: image/jpeg" --data-binary @./test.jpg "<upload_url>"
```
3. Complete upload:
```
curl -X POST http://localhost:8080/complete-upload \
  -H "Content-Type: application/json" \
  -d '{"media_id":"<id>","original_key":"media/<id>/original.jpg"}'
```
4. Check status:
```
curl http://localhost:8080/media/<id>
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

## Environment
Copy `.env.example` to `.env` and adjust if needed.

Key vars:
- `DATABASE_URL`
- `RABBITMQ_URL`
- `MINIO_ENDPOINT`
- `MINIO_PUBLIC_ENDPOINT`
- `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY`

## Testing Focus
- Happy path: upload -> processing -> READY
- Idempotency: repeated `complete-upload`
- Worker crash recovery
- ACK/DB ordering correctness

## Learning Plan
See `LEARNING_PLAN.md` for the structured course outline and progress tracking.

## Stack
Go, RabbitMQ, Postgres, MinIO, Docker Compose
