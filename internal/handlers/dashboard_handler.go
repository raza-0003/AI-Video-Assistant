package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raza-0003/ai-video-backend/internal/auth"
	"github.com/raza-0003/ai-video-backend/internal/models"
)

type DashboardHandler struct {
	DB *pgxpool.Pool
}

func NewDashboardHandler(db *pgxpool.Pool) *DashboardHandler {
	return &DashboardHandler{DB: db}
}

// Stats godoc
// @Summary      Get dashboard stats for the authenticated user
// @Tags         dashboard
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} models.DashboardStats
// @Router       /api/v1/dashboard/stats [get]
func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	var stats models.DashboardStats
	err := h.DB.QueryRow(r.Context(), `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'processing') AS processing,
			COUNT(*) FILTER (WHERE status = 'completed')  AS completed,
			COUNT(*) FILTER (WHERE status = 'failed')     AS failed
		FROM videos
		WHERE user_id = $1
	`, userID).Scan(&stats.TotalVideos, &stats.Processing, &stats.Completed, &stats.Failed)

	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to compute dashboard stats")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
