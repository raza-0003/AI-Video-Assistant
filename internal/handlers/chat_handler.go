package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raza-0003/ai-video-backend/internal/models"
)

type ChatHandler struct {
	DB           *pgxpool.Pool
	PythonSvcURL string
	Client       *http.Client
}

func NewChatHandler(db *pgxpool.Pool, pythonSvcURL string) *ChatHandler {
	return &ChatHandler{DB: db, PythonSvcURL: pythonSvcURL, Client: &http.Client{Timeout: 60 * time.Second}}
}

type chatRequest struct {
	Question string `json:"question"`
}

type pythonChatResponse struct {
	Answer string `json:"answer"`
	Error  string `json:"error,omitempty"`
}

// Chat godoc
// @Summary      Ask a question about a processed video (RAG chat)
// @Tags         chat
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Video ID"
// @Param        body body chatRequest true "Question payload"
// @Success      200 {object} models.ChatMessage
// @Router       /api/v1/videos/{id}/chat [post]
func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "id")

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Question == "" {
		httpError(w, http.StatusBadRequest, "question is required")
		return
	}

	ctx := r.Context()

	// Persist the user's question first so history is never lost even if the
	// downstream pipeline call fails.
	h.saveMessage(ctx, videoID, "user", req.Question)

	body, _ := json.Marshal(map[string]string{"video_id": videoID, "question": req.Question})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.PythonSvcURL+"/chat", bytes.NewReader(body))
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to build pipeline request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.Client.Do(httpReq)
	if err != nil {
		httpError(w, http.StatusBadGateway, "pipeline service unreachable")
		return
	}
	defer resp.Body.Close()

	var pr pythonChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil || pr.Error != "" {
		httpError(w, http.StatusInternalServerError, "chat pipeline error")
		return
	}

	msg := h.saveMessage(ctx, videoID, "assistant", pr.Answer)
	writeJSON(w, http.StatusOK, msg)
}

// History godoc
// @Summary      Get chat history for a video
// @Tags         chat
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Video ID"
// @Success      200 {array} models.ChatMessage
// @Router       /api/v1/videos/{id}/chat [get]
func (h *ChatHandler) History(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "id")
	rows, err := h.DB.Query(r.Context(),
		`SELECT id, video_id, role, content, created_at FROM chat_messages WHERE video_id = $1 ORDER BY created_at ASC`,
		videoID,
	)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to fetch chat history")
		return
	}
	defer rows.Close()

	messages := []models.ChatMessage{}
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(&m.ID, &m.VideoID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			httpError(w, http.StatusInternalServerError, "failed to read chat history")
			return
		}
		messages = append(messages, m)
	}
	writeJSON(w, http.StatusOK, messages)
}

func (h *ChatHandler) saveMessage(ctx context.Context, videoID, role, content string) models.ChatMessage {
	id := uuid.NewString()
	now := time.Now()
	h.DB.Exec(ctx, `INSERT INTO chat_messages (id, video_id, role, content) VALUES ($1, $2, $3, $4)`,
		id, videoID, role, content)
	return models.ChatMessage{ID: id, VideoID: videoID, Role: role, Content: content, CreatedAt: now}
}
