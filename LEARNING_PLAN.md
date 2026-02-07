# Learning Plan

This document tracks the learning progress for the image processing pipeline system design project. Each chapter has a short goal statement so progress is clear and measurable.

## 1. Project Setup & Local Runtime (Completed)
Goal: Bootstrap the repo and local infra so the system can run end-to-end in Docker.

## 2. Database Schema & State Machine (Completed)
Goal: Implement `media` and `processing_task` tables and clarify state transitions.

## 3. API Server Fundamentals
Goal: Build the minimal API (`/upload-url`, `/complete-upload`, `/media/{id}`) with proper validation and persistence.

## 4. Object Storage & Presigned Uploads
Goal: Integrate MinIO/S3 for direct uploads and deterministic object keys.

## 5. Messaging & Task Enqueue
Goal: Publish processing tasks to RabbitMQ and ensure reliable delivery.

## 6. Worker Processing Loop
Goal: Implement claim/lease logic, step execution order, and safe ACK behavior.

## 7. Idempotency & Retry Strategy
Goal: Add deterministic outputs, retry/backoff rules, and max retry handling.

## 8. Observability Basics
Goal: Add structured logging, essential metrics, and traceable identifiers.

## 9. Test Scenarios & Validation
Goal: Run happy path, idempotency, crash recovery, and ACK-order tests.

## 10. Deployment Path (Cloudflare)
Goal: Map services to Cloudflare options and document a practical deployment approach.
