# AI Video Assistant

An AI-powered meeting assistant that processes YouTube videos or uploaded recordings, generates transcripts and summaries, extracts action items, and allows users to chat with the meeting using Retrieval-Augmented Generation (RAG).

## Features

- User authentication with JWT
- Upload YouTube links for processing
- Background job processing using Redis + Asynq
- Live processing progress updates
- Meeting transcript and summary generation
- Action item extraction
- Chat with processed meetings using RAG
- Dashboard showing processing statistics
- REST APIs with Swagger documentation

---

## Tech Stack

### Backend
- Go
- Chi Router
- PostgreSQL
- Redis
- Asynq
- JWT Authentication
- AWS SDK (S3 integration)

### AI Service
- Python
- FastAPI
- Whisper
- LangChain
- Mistral AI
- ChromaDB

### Frontend
- React
- Vite

---

## Architecture

```
React Frontend
       │
       ▼
    Go REST API
       │
       ├────────► PostgreSQL
       │
       ├────────► Redis Queue
       │
       ▼
    Go Worker
       │
       ▼
Python FastAPI Service
       │
       ├── Whisper
       ├── Mistral AI
       └── ChromaDB
```

The Go backend is responsible for authentication, APIs, database operations, and job scheduling. Long-running AI tasks are executed by a separate Python service through background workers, keeping API requests responsive.

---

## Running the Project

```bash
cp .env.example .env

docker compose up --build
```

Services:

- Go API → http://localhost:8080
- Swagger → http://localhost:8080/docs/index.html
- Python Service → http://localhost:8000
- React Frontend → http://localhost:5173

---

## API Endpoints

| Method | Endpoint | Description |
|---------|----------|-------------|
| POST | /api/v1/auth/register | Register user |
| POST | /api/v1/auth/login | Login |
| POST | /api/v1/videos | Submit video |
| GET | /api/v1/videos | Video history |
| GET | /api/v1/videos/{id} | Processing status |
| POST | /api/v1/videos/{id}/chat | Chat with meeting |
| GET | /api/v1/dashboard/stats | Dashboard statistics |

---

## Future Improvements

- Store transcripts in AWS S3
- Add rate limiting
- Improve logging
- Add unit and integration tests
