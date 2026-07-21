-- Initial schema for AI Video Assistant platform

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- status: pending | processing | completed | failed | cancelled
CREATE TABLE IF NOT EXISTS videos (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source           TEXT NOT NULL,            -- youtube URL or uploaded file key
    language         TEXT NOT NULL DEFAULT 'english',
    status           TEXT NOT NULL DEFAULT 'pending',
    progress_percent INT  NOT NULL DEFAULT 0,
    task_id          TEXT,                     -- Asynq task ID, needed to cancel/delete the job
    title            TEXT,
    summary          TEXT,
    transcript_s3_key TEXT,
    action_items     TEXT,
    key_decisions    TEXT,
    open_questions   TEXT,
    error_message    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_videos_user_id ON videos(user_id);
CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status);

CREATE TABLE IF NOT EXISTS chat_messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id   UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    role       TEXT NOT NULL,   -- user | assistant
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_video_id ON chat_messages(video_id);

CREATE EXTENSION IF NOT EXISTS pgcrypto;
