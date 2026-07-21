-- Adds support for cancelling an in-flight or queued video-processing job.
-- (Migration runner just re-executes every .sql file on startup with no
-- version tracking, so ALTER TABLE ... IF NOT EXISTS keeps this idempotent
-- and safe to run against a database that already has the 001 schema.)

ALTER TABLE videos ADD COLUMN IF NOT EXISTS task_id TEXT;
