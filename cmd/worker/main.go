// Package main is the Asynq worker that consumes video-processing jobs,
// delegates the actual transcription/summarization/RAG work to the Python
// microservice (the original AI-Video-Assistant pipeline), and writes
// progress + results back to Postgres.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/raza-0003/ai-video-backend/internal/config"
	"github.com/raza-0003/ai-video-backend/internal/db"
	"github.com/raza-0003/ai-video-backend/internal/queue"
)

type Worker struct {
	DB           *pgxpool.Pool
	PythonSvcURL string
	HTTPClient   *http.Client
}

type pythonPipelineResponse struct {
	Title         string `json:"title"`
	Transcript    string `json:"transcript"`
	Summary       string `json:"summary"`
	ActionItems   string `json:"action_items"`
	KeyDecisions  string `json:"key_decisions"`
	OpenQuestions string `json:"open_questions"`
	Error         string `json:"error,omitempty"`
}

// HandleProcessVideo is invoked by Asynq for every "video:process" task.
// It walks the pipeline stages, persisting progress after each one so the
// frontend can poll /api/v1/videos/{id} and show a live percentage instead
// of a static "Generating..." spinner.
func (w *Worker) HandleProcessVideo(ctx context.Context, t *asynq.Task) error {
	var p queue.ProcessVideoPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("invalid task payload: %w", err)
	}

	log.Printf("processing video %s (source=%s)", p.VideoID, p.Source)

	w.setStatus(ctx, p.VideoID, "processing", 5, "")

	reqBody, _ := json.Marshal(map[string]string{
		"source":   p.Source,
		"language": p.Language,
		"video_id": p.VideoID,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.PythonSvcURL+"/process", bytes.NewReader(reqBody))
	if err != nil {
		return w.fail(ctx, p.VideoID, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// The pipeline (download -> whisper transcribe -> mistral summarize/extract
	// -> chroma embed) can legitimately take minutes for long meetings.
	client := &http.Client{Timeout: 20 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		// If the context was cancelled (via CancelRunningTask from the API),
		// this is a user-initiated stop, not a real failure — record it as such
		// instead of marking the job "failed".
		if ctx.Err() != nil {
			w.setStatus(context.Background(), p.VideoID, "cancelled", 0, "")
			log.Printf("video %s processing cancelled by user", p.VideoID)
			return nil
		}
		return w.fail(ctx, p.VideoID, err)
	}
	defer resp.Body.Close()

	w.setStatus(ctx, p.VideoID, "processing", 60, "")

	var result pythonPipelineResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return w.fail(ctx, p.VideoID, err)
	}
	if resp.StatusCode != http.StatusOK || result.Error != "" {
		return w.fail(ctx, p.VideoID, fmt.Errorf("pipeline error: %s", result.Error))
	}

	w.setStatus(ctx, p.VideoID, "processing", 90, "")

	_, err = w.DB.Exec(ctx, `
		UPDATE videos
		SET status = 'completed', progress_percent = 100,
		    title = $1, summary = $2, action_items = $3,
		    key_decisions = $4, open_questions = $5,
		    updated_at = now(), completed_at = now()
		WHERE id = $6
	`, result.Title, result.Summary, result.ActionItems,
		result.KeyDecisions, result.OpenQuestions, p.VideoID)
	if err != nil {
		return fmt.Errorf("failed to persist results: %w", err)
	}

	log.Printf("video %s completed", p.VideoID)
	return nil
}

func (w *Worker) setStatus(ctx context.Context, videoID, status string, progress int, errMsg string) {
	_, err := w.DB.Exec(ctx, `
		UPDATE videos SET status = $1, progress_percent = $2, error_message = NULLIF($3, ''), updated_at = now()
		WHERE id = $4
	`, status, progress, errMsg, videoID)
	if err != nil {
		log.Printf("failed to update status for video %s: %v", videoID, err)
	}
}

func (w *Worker) fail(ctx context.Context, videoID string, err error) error {
	w.setStatus(ctx, videoID, "failed", 0, err.Error())
	return err
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	worker := &Worker{
		DB:           pool,
		PythonSvcURL: cfg.PythonSvcURL,
		HTTPClient:   &http.Client{Timeout: 20 * time.Minute},
	}

	srv := queue.NewServer(cfg.RedisAddr)
	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TypeProcessVideo, worker.HandleProcessVideo)

	log.Println("AI Video Assistant worker started, waiting for jobs...")
	if err := srv.Run(mux); err != nil {
		log.Fatalf("worker failed: %v", err)
	}
}
