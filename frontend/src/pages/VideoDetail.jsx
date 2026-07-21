import { useEffect, useState, useCallback, useRef } from "react";
import { useParams, Link } from "react-router-dom";
import { api } from "../api";
import StatusBadge from "../components/StatusBadge";

export default function VideoDetail() {
  const { id } = useParams();
  const [video, setVideo] = useState(null);
  const [messages, setMessages] = useState([]);
  const [question, setQuestion] = useState("");
  const [sending, setSending] = useState(false);
  const chatEndRef = useRef(null);

  const load = useCallback(async () => {
    const v = await api.getVideo(id);
    setVideo(v);
    if (v.status === "completed") {
      const history = await api.chatHistory(id);
      setMessages(history);
    }
  }, [id]);

  useEffect(() => {
    load();
    const interval = setInterval(() => {
      // Stop polling once the job is done or failed — nothing left to update.
      setVideo((current) => {
        if (current && (current.status === "completed" || current.status === "failed" || current.status === "cancelled")) {
          clearInterval(interval);
          return current;
        }
        return current;
      });
      load();
    }, 3000);
    return () => clearInterval(interval);
  }, [load]);

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  async function handleSend(e) {
    e.preventDefault();
    if (!question.trim()) return;
    const q = question.trim();
    setQuestion("");
    setMessages((m) => [...m, { id: `local-${Date.now()}`, role: "user", content: q }]);
    setSending(true);
    try {
      const reply = await api.sendChat(id, q);
      setMessages((m) => [...m, reply]);
    } catch (err) {
      setMessages((m) => [...m, { id: `err-${Date.now()}`, role: "assistant", content: `Error: ${err.message}` }]);
    } finally {
      setSending(false);
    }
  }

  async function handleCancel() {
    try {
      await api.cancelVideo(id);
      await load();
    } catch (err) {
      // Surface via the failed/cancelled panel on next load — good enough
      // for this scope; a toast would be a nice next iteration.
      console.error(err);
    }
  }

  if (!video) {
    return (
      <div className="page">
        <div className="container empty-state">
          <span className="spinner" />
        </div>
      </div>
    );
  }

  return (
    <div className="page">
      <div className="container">
        <Link to="/" className="back-link">
          ← Back to dashboard
        </Link>

        <div className="page-header">
          <div>
            <h1 className="page-title">{video.title || "Untitled meeting"}</h1>
            <p className="page-subtitle">{video.source}</p>
          </div>
          <StatusBadge status={video.status} />
        </div>

        {video.status === "processing" && (
          <div className="panel detail-panel" style={{ marginBottom: 20 }}>
            <div className="detail-block-title">Progress</div>
            <div className="progress-hero">
              <span className="progress-hero-value">{video.progress_percent}%</span>
              <div className="progress-hero-track">
                <div className="progress-hero-fill" style={{ width: `${video.progress_percent}%` }} />
              </div>
              <button className="btn-cancel" onClick={handleCancel}>
                Cancel
              </button>
            </div>
          </div>
        )}

        {video.status === "pending" && (
          <div className="panel detail-panel" style={{ marginBottom: 20 }}>
            <div className="detail-block-title">Queued</div>
            <div className="detail-block-body" style={{ color: "var(--text-muted)", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
              <span>Waiting for a worker to pick this up.</span>
              <button className="btn-cancel" onClick={handleCancel}>
                Cancel
              </button>
            </div>
          </div>
        )}

        {video.status === "cancelled" && (
          <div className="panel detail-panel" style={{ marginBottom: 20 }}>
            <div className="detail-block-title">Cancelled</div>
            <div className="detail-block-body" style={{ color: "var(--text-muted)" }}>
              This job was stopped before it finished.
            </div>
          </div>
        )}

        {video.status === "failed" && (
          <div className="panel detail-panel" style={{ marginBottom: 20 }}>
            <div className="detail-block-title">Error</div>
            <div className="detail-block-body" style={{ color: "var(--danger)" }}>
              {video.error_message || "Processing failed."}
            </div>
          </div>
        )}

        {video.status === "completed" && (
          <div className="detail-grid">
            <div>
              <div className="panel detail-panel">
                <div className="detail-block-title">Summary</div>
                <div className="detail-block-body">{video.summary || "—"}</div>
              </div>
              <div className="panel detail-panel">
                <div className="detail-block-title">Action items</div>
                <div className="detail-block-body">{video.action_items || "None identified."}</div>
              </div>
              <div className="panel detail-panel">
                <div className="detail-block-title">Key decisions</div>
                <div className="detail-block-body">{video.key_decisions || "None identified."}</div>
              </div>
              <div className="panel detail-panel">
                <div className="detail-block-title">Open questions</div>
                <div className="detail-block-body">{video.open_questions || "None identified."}</div>
              </div>
            </div>

            <div className="panel chat-panel">
              <div className="chat-header">Ask about this meeting</div>
              <div className="chat-messages">
                {messages.length === 0 && (
                  <div className="chat-empty">Ask a question about the transcript to get started.</div>
                )}
                {messages.map((m) => (
                  <div key={m.id} className={`chat-msg ${m.role}`}>
                    {m.content}
                  </div>
                ))}
                <div ref={chatEndRef} />
              </div>
              <form className="chat-input-row" onSubmit={handleSend}>
                <input
                  type="text"
                  value={question}
                  onChange={(e) => setQuestion(e.target.value)}
                  placeholder="What were the key decisions?"
                  disabled={sending}
                />
                <button className="btn btn-primary" disabled={sending}>
                  {sending ? <span className="spinner" /> : "Send"}
                </button>
              </form>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
