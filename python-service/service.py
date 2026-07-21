"""
Internal microservice wrapping the original AI-Video-Assistant pipeline
(audio download/chunk -> Whisper transcription -> Mistral summarization/
extraction -> Chroma RAG). The Go API/worker never talks to Whisper, Mistral,
or Chroma directly — it calls this service over HTTP, which keeps the ML
stack isolated from the backend and lets each side scale independently.

Run standalone:
    uvicorn service:app --host 0.0.0.0 --port 8000
"""
import os
import traceback

from dotenv import load_dotenv
from fastapi import FastAPI
from pydantic import BaseModel

from utils.audio_processor import process_input
from core.transcriber import transcribe_all
from core.summarizer import summarize, generate_title
from core.extractor import extract_action_items, extract_key_decisions, extract_questions
from core.rag_engine import build_rag_chain, ask_question

load_dotenv()

app = FastAPI(title="AI Video Assistant - Pipeline Service")

# In-memory registry of built RAG chains, keyed by video_id, so /chat can
# reuse the vector store without re-embedding on every question.
_RAG_CHAINS = {}


class ProcessRequest(BaseModel):
    source: str
    language: str = "english"
    video_id: str | None = None


class ProcessResponse(BaseModel):
    title: str = ""
    transcript: str = ""
    summary: str = ""
    action_items: str = ""
    key_decisions: str = ""
    open_questions: str = ""
    error: str = ""


class ChatRequest(BaseModel):
    video_id: str
    question: str


class ChatResponse(BaseModel):
    answer: str = ""
    error: str = ""


@app.get("/health")
def health():
    return {"status": "ok"}


@app.post("/process", response_model=ProcessResponse)
def process(req: ProcessRequest):
    """Runs the full pipeline: download/convert -> transcribe -> summarize ->
    extract -> build RAG chain. Mirrors main.run_pipeline() from the original
    Streamlit app, just called over HTTP instead of a CLI/UI."""
    try:
        chunks = process_input(req.source)
        transcript = transcribe_all(chunks, req.language)

        title = generate_title(transcript)
        summary = summarize(transcript)
        action_items = extract_action_items(transcript)
        key_decisions = extract_key_decisions(transcript)
        open_questions = extract_questions(transcript)

        rag_chain = build_rag_chain(transcript)
        if req.video_id:
            _RAG_CHAINS[req.video_id] = rag_chain

        return ProcessResponse(
            title=title,
            transcript=transcript,
            summary=summary,
            action_items=action_items,
            key_decisions=key_decisions,
            open_questions=open_questions,
        )
    except Exception as e:
        traceback.print_exc()
        return ProcessResponse(error=str(e))


@app.post("/chat", response_model=ChatResponse)
def chat(req: ChatRequest):
    """Chat-with-your-meeting endpoint, backed by the RAG chain built during /process."""
    chain = _RAG_CHAINS.get(req.video_id)
    if chain is None:
        return ChatResponse(error="no RAG chain found for this video_id — call /process first")
    try:
        answer = ask_question(chain, req.question)
        return ChatResponse(answer=answer)
    except Exception as e:
        traceback.print_exc()
        return ChatResponse(error=str(e))
