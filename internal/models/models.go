package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// VideoStatus enumerates the lifecycle of a submitted video/meeting job.
type VideoStatus string

const (
	StatusPending    VideoStatus = "pending"
	StatusProcessing VideoStatus = "processing"
	StatusCompleted  VideoStatus = "completed"
	StatusFailed     VideoStatus = "failed"
	StatusCancelled  VideoStatus = "cancelled"
)

type Video struct {
	ID               string      `json:"id"`
	UserID           string      `json:"user_id"`
	Source           string      `json:"source"`
	Language         string      `json:"language"`
	Status           VideoStatus `json:"status"`
	ProgressPercent  int         `json:"progress_percent"`
	TaskID           *string     `json:"-"`
	Title            *string     `json:"title,omitempty"`
	Summary          *string     `json:"summary,omitempty"`
	TranscriptS3Key  *string     `json:"transcript_s3_key,omitempty"`
	ActionItems      *string     `json:"action_items,omitempty"`
	KeyDecisions     *string     `json:"key_decisions,omitempty"`
	OpenQuestions    *string     `json:"open_questions,omitempty"`
	ErrorMessage     *string     `json:"error_message,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	CompletedAt      *time.Time  `json:"completed_at,omitempty"`
}

type ChatMessage struct {
	ID        string    `json:"id"`
	VideoID   string    `json:"video_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type DashboardStats struct {
	TotalVideos int `json:"total_videos"`
	Processing  int `json:"processing"`
	Completed   int `json:"completed"`
	Failed      int `json:"failed"`
}
