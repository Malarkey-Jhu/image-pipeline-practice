CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE media_status AS ENUM ('INIT', 'UPLOADED', 'PROCESSING', 'READY', 'FAILED');
CREATE TYPE task_status AS ENUM ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'RETRY');
CREATE TYPE task_step AS ENUM ('resize', 'compress', 'webp');

CREATE TABLE IF NOT EXISTS media (
  id TEXT PRIMARY KEY,
  status media_status NOT NULL,
  original_key TEXT,
  final_key TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS processing_task (
  id TEXT PRIMARY KEY,
  media_id TEXT NOT NULL REFERENCES media(id) ON DELETE CASCADE,
  step task_step NOT NULL,
  status task_status NOT NULL,
  retry_count INT NOT NULL DEFAULT 0,
  lock_by TEXT,
  lock_until TIMESTAMPTZ,
  input_key TEXT NOT NULL,
  output_key TEXT NOT NULL,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(media_id, step)
);

CREATE INDEX IF NOT EXISTS idx_processing_task_status_lock ON processing_task(status, lock_until);
