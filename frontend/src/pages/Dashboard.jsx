import { useEffect, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";
import StatusBadge from "../components/StatusBadge";

function timeAgo(iso) {
  const diff = (Date.now() - new Date(iso).getTime()) / 1000;
  if (diff < 60) return "just now";
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export default function Dashboard() {
  const [stats, setStats] = useState(null);
  const [videos, setVideos] = useState([]);
  const [source, setSource] = useState("");
  const [language, setLanguage] = useState("english");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    const [s, v] = await Promise.all([api.dashboardStats(), api.listVideos()]);
    setStats(s);
    setVideos(v);
    setLoading(false);
  }, []);

  useEffect(() => {
    load();
    // Poll every 4s so in-progress jobs update their status/percentage live,
    // without needing websockets for a project this size.
    const interval = setInterval(load, 4000);
    return () => clearInterval(interval);
  }, [load]);

  async function handleSubmit(e) {
    e.preventDefault();
    setError("");
    if (!source.trim()) return;
    setSubmitting(true);
    try {
      await api.submitVideo(source.trim(), language);
      setSource("");
      await load();
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  }

  async function handleCancel(e, id) {
    e.preventDefault();
    e.stopPropagation();
    try {
      await api.cancelVideo(id);
      await load();
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <div className="page">
      <div className="container">
        <div className="page-header">
          <div>
            <h1 className="page-title">Dashboard</h1>
            <p className="page-subtitle">Submit a meeting and track it through the pipeline.</p>
          </div>
        </div>

        <div className="stat-grid">
          <div className="panel stat-card">
            <div className="stat-label">Total videos</div>
            <div className="stat-value">{stats?.total_videos ?? "—"}</div>
          </div>
          <div className="panel stat-card">
            <div className="stat-label">Processing</div>
            <div className="stat-value processing">{stats?.processing ?? "—"}</div>
          </div>
          <div className="panel stat-card">
            <div className="stat-label">Completed</div>
            <div className="stat-value success">{stats?.completed ?? "—"}</div>
          </div>
          <div className="panel stat-card">
            <div className="stat-label">Failed</div>
            <div className="stat-value danger">{stats?.failed ?? "—"}</div>
          </div>
        </div>

        <div className="panel submit-panel">
          <div className="waveform" aria-hidden="true">
            {[40, 70, 45, 90, 55, 30, 65, 80, 40, 60, 35, 75].map((h, i) => (
              <span key={i} style={{ animationDelay: `${i * 0.08}s`, height: `${h}%` }} />
            ))}
          </div>
          <div className="section-label">Submit a meeting</div>
          {error && <div className="error-banner">{error}</div>}
          <form onSubmit={handleSubmit}>
            <div className="submit-row">
              <div className="field">
                <label htmlFor="source">YouTube URL or file key</label>
                <input
                  id="source"
                  type="text"
                  value={source}
                  onChange={(e) => setSource(e.target.value)}
                  placeholder="https://www.youtube.com/watch?v=..."
                  required
                />
              </div>
              <div className="field" style={{ flex: "0 0 160px" }}>
                <label htmlFor="language">Language</label>
                <select
                  id="language"
                  value={language}
                  onChange={(e) => setLanguage(e.target.value)}
                  style={{
                    width: "100%",
                    background: "var(--panel-raised)",
                    border: "1px solid var(--border)",
                    borderRadius: "var(--radius-sm)",
                    padding: "11px 13px",
                    color: "var(--text)",
                  }}
                >
                  <option value="english">English</option>
                  <option value="hinglish">Hinglish</option>
                </select>
              </div>
              <button className="btn btn-primary" disabled={submitting} style={{ marginTop: 22 }}>
                {submitting ? <span className="spinner" /> : "Analyze"}
              </button>
            </div>
          </form>
        </div>

        <div className="section-label">History</div>

        {loading ? (
          <div className="empty-state">
            <span className="spinner" />
          </div>
        ) : videos.length === 0 ? (
          <div className="panel empty-state">
            <div className="empty-state-title">No meetings yet</div>
            <p>Paste a YouTube URL above to run your first analysis.</p>
          </div>
        ) : (
          <div className="video-list">
            {videos.map((v) => (
              <Link to={`/videos/${v.id}`} key={v.id} className="panel video-row">
                <div className="video-row-main">
                  <div className="video-row-title">{v.title || "Untitled meeting"}</div>
                  <div className="video-row-source">{v.source}</div>
                </div>
                <div className="video-row-meta">
                  {v.status === "processing" && (
                    <div className="progress-track">
                      <div className="progress-fill" style={{ width: `${v.progress_percent}%` }} />
                    </div>
                  )}
                  <StatusBadge status={v.status} />
                  {(v.status === "pending" || v.status === "processing") && (
                    <button className="btn-cancel" onClick={(e) => handleCancel(e, v.id)}>
                      Cancel
                    </button>
                  )}
                  <span className="timestamp">{timeAgo(v.created_at)}</span>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
