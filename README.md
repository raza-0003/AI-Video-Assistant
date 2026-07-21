# AI Video Assistant — Backend Platform

A production-style backend built around an existing Python meeting-intelligence
pipeline (audio ingestion → Whisper transcription → Mistral summarization/
extraction → Chroma RAG chat).

| Original | This version |
|---|---|
| No auth | JWT-based register/login |
| No persistence | Postgres: users, videos, chat history |
| Synchronous, blocks UI | Redis + Asynq background job queue |
| `"Generating..."` spinner | Live progress percentage, polled from DB |
| Local Chroma dir only | Pluggable S3 storage layer |
| No API | REST API + Swagger docs at `/docs` |
| Single Python process | Go API + Go worker + isolated Python ML microservice |

## Architecture

```
                     ┌──────────────┐
   Client / Frontend │              │
   ─────────────────▶│  Go API      │──┐
   (REST, JWT)        │  (chi)       │  │ enqueue job
                     └──────┬───────┘  │
                            │ Postgres  ▼
                            │        ┌────────┐
                            │        │ Redis  │
                            │        │ Asynq  │
                            │        └───┬────┘
                            ▼            │ consume
                     ┌──────────────┐    ▼
                     │  Postgres    │ ┌──────────────┐
                     │  users       │ │  Go Worker    │
                     │  videos      │ │              │
                     │  chat_msgs   │ └──────┬───────┘
                     └──────────────┘        │ HTTP
                                              ▼
                                    ┌──────────────────┐
                                    │ Python Service    │
                                    │ (FastAPI)         │
                                    │ yt-dlp/pydub      │
                                    │ Whisper            │
                                    │ Mistral (LangChain)│
                                    │ Chroma vector store │
                                    └──────────────────┘
```

The Go API never talks to Whisper/Mistral/Chroma directly — it enqueues a job,
the worker calls the Python service over HTTP, and results/progress are
written back to Postgres. This is the standard pattern for wrapping a
Python ML pipeline behind a real backend instead of rewriting the ML code
in Go.

## Stack

- **Go** (chi router, pgx, JWT, Asynq, AWS SDK v2)
- **PostgreSQL** — users, videos, chat history
- **Redis + Asynq** — background job queue for long-running pipeline jobs
- **AWS S3** — planned artifact storage (transcripts, exports)
- **Swagger** (swaggo) — API docs at `/docs/index.html`
- **Docker Compose** — one-command local environment
- **Python/FastAPI** — thin wrapper around the original transcription/RAG pipeline

## Frontend

A React (Vite) UI lives in `frontend/` — login/register, a dashboard to submit
meetings and watch live progress, and a per-meeting page with summary, action
items, and RAG chat.

```bash
cd frontend
npm install
cp .env.example .env    # points at http://localhost:8080 by default
npm run dev
```

Open `http://localhost:5173`. It talks to the Go API directly over REST — no
proxy needed, CORS is already open on the API side.

## Running locally

```bash
cp .env.example .env
# fill in MISTRAL_API_KEY at minimum

docker compose up --build
```

- API: `http://localhost:8080` (health check: `/health`, docs: `/docs/index.html`)
- Python pipeline service: `http://localhost:8000`

## API overview

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/auth/register` | — | Create account |
| POST | `/api/v1/auth/login` | — | Get JWT |
| POST | `/api/v1/videos` | ✅ | Submit YouTube URL/file for processing (enqueues job) |
| GET | `/api/v1/videos` | ✅ | Video history for the current user |
| GET | `/api/v1/videos/{id}` | ✅ | Status, progress %, and results for one video |
| POST | `/api/v1/videos/{id}/chat` | ✅ | Ask a question about a processed video (RAG) |
| GET | `/api/v1/videos/{id}/chat` | ✅ | Chat history for that video |
| GET | `/api/v1/dashboard/stats` | ✅ | Total / processing / completed / failed counts |

## Not yet wired up (next steps)

- Swagger doc comments are in place but need `swag init` run to generate `docs/`
- S3 upload of transcripts is scaffolded (`internal/storage/s3.go`) but not
  yet called from the worker — currently transcripts stay in Postgres
- Rate limiting, structured logging (zap/zerolog), and unit/API tests are
  not included in this scaffold yet
- `go mod tidy` needs to run with real internet access to the Go module proxy
