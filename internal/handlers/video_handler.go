package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raza-0003/ai-video-backend/internal/auth"
	"github.com/raza-0003/ai-video-backend/internal/models"
	"github.com/raza-0003/ai-video-backend/internal/queue"
)

type VideoHandler struct {
	DB          *pgxpool.Pool
	QueueClient *asynq.Client
	Inspector   *asynq.Inspector
}

func NewVideoHandler(db *pgxpool.Pool, qc *asynq.Client, inspector *asynq.Inspector) *VideoHandler {
	return &VideoHandler{DB: db, QueueClient: qc, Inspector: inspector}
}

type submitVideoRequest struct {
	Source   string `json:"source"`   // YouTube URL or an already-uploaded S3/file key
	Language string `json:"language"` // "english" | "hinglish"
}

// Submit godoc
// @Summary      Submit a video/meeting for AI analysis
// @Tags         videos
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body submitVideoRequest true "Submission payload"
// @Success      202 {object} models.Video
// @Router       /api/v1/videos [post]
func (h *VideoHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var req submitVideoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Source == "" {
		httpError(w, http.StatusBadRequest, "source is required")
		return
	}
	if req.Language == "" {
		req.Language = "english"
	}

	id := uuid.NewString()
	ctx := r.Context()
	_, err := h.DB.Exec(ctx,
		`INSERT INTO videos (id, user_id, source, language, status)
		 VALUES ($1, $2, $3, $4, 'pending')`,
		id, userID, req.Source, req.Language,
	)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to create video record")
		return
	}

	// Hand off the heavy lifting (download/transcribe/RAG) to the async worker
	// so the API responds immediately instead of blocking the HTTP request.
	taskInfo, err := queue.Enqueue(ctx, h.QueueClient, queue.ProcessVideoPayload{
		VideoID:  id,
		Source:   req.Source,
		Language: req.Language,
	})
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to enqueue processing job")
		return
	}

	// Store the task ID so a later "cancel" request knows what to remove/signal.
	_, _ = h.DB.Exec(ctx, `UPDATE videos SET task_id = $1 WHERE id = $2`, taskInfo.ID, id)

	writeJSON(w, http.StatusAccepted, map[string]string{
		"id":     id,
		"status": "pending",
	})
}

// Get godoc
// @Summary      Get a single video's status/results
// @Tags         videos
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Video ID"
// @Success      200 {object} models.Video
// @Router       /api/v1/videos/{id} [get]
func (h *VideoHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	v, err := scanVideo(h.DB.QueryRow(r.Context(), videoSelectQuery+` WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		if err == pgx.ErrNoRows {
			httpError(w, http.StatusNotFound, "video not found")
			return
		}
		httpError(w, http.StatusInternalServerError, "failed to fetch video")
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// History godoc
// @Summary      List the authenticated user's video history
// @Tags         videos
// @Security     BearerAuth
// @Produce      json
// @Success      200 {array} models.Video
// @Router       /api/v1/videos [get]
func (h *VideoHandler) History(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	rows, err := h.DB.Query(r.Context(), videoSelectQuery+` WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to fetch history")
		return
	}
	defer rows.Close()

	videos := []models.Video{}
	for rows.Next() {
		v, err := scanVideoRows(rows)
		if err != nil {
			httpError(w, http.StatusInternalServerError, "failed to read history")
			return
		}
		videos = append(videos, v)
	}
	writeJSON(w, http.StatusOK, videos)
}

// Cancel godoc
// @Summary      Cancel a pending or in-progress video job
// @Tags         videos
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Video ID"
// @Success      200 {object} models.Video
// @Router       /api/v1/videos/{id}/cancel [post]
func (h *VideoHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	var status string
	var taskID *string
	err := h.DB.QueryRow(ctx,
		`SELECT status, task_id FROM videos WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&status, &taskID)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpError(w, http.StatusNotFound, "video not found")
			return
		}
		httpError(w, http.StatusInternalServerError, "failed to fetch video")
		return
	}

	if status != string(models.StatusPending) && status != string(models.StatusProcessing) {
		httpError(w, http.StatusConflict, "only pending or processing videos can be cancelled")
		return
	}

	if taskID != nil && *taskID != "" {
		var cancelErr error
		if status == string(models.StatusProcessing) {
			// Already running: signal its context to cancel. The worker's HTTP
			// call to the Python service is built with this context, so it
			// aborts mid-flight instead of running to completion.
			cancelErr = queue.CancelRunningTask(h.Inspector, *taskID)
		} else {
			// Still sitting in the queue, never picked up: just remove it.
			cancelErr = queue.CancelQueuedTask(h.Inspector, "default", *taskID)
		}
		// Best-effort: if the task already finished/moved on between our status
		// read and this call, that's fine — we still mark the row cancelled below.
		_ = cancelErr
	}

	_, err = h.DB.Exec(ctx,
		`UPDATE videos SET status = 'cancelled', updated_at = now() WHERE id = $1`, id,
	)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to update video status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": "cancelled"})
}

const videoSelectQuery = `
SELECT id, user_id, source, language, status, progress_percent, title, summary,
       transcript_s3_key, action_items, key_decisions, open_questions, error_message,
       created_at, updated_at, completed_at
FROM videos`

// scanner abstracts pgx.Row vs pgx.Rows so both Get and History can share scan logic.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanVideo(row pgx.Row) (models.Video, error) {
	return scanVideoRow(row)
}

func scanVideoRows(rows pgx.Rows) (models.Video, error) {
	return scanVideoRow(rows)
}

func scanVideoRow(s scanner) (models.Video, error) {
	var v models.Video
	err := s.Scan(
		&v.ID, &v.UserID, &v.Source, &v.Language, &v.Status, &v.ProgressPercent,
		&v.Title, &v.Summary, &v.TranscriptS3Key, &v.ActionItems, &v.KeyDecisions,
		&v.OpenQuestions, &v.ErrorMessage, &v.CreatedAt, &v.UpdatedAt, &v.CompletedAt,
	)
	return v, err
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
