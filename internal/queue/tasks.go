package queue

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

// ProcessVideoPayload is enqueued when a user submits a new video/meeting for analysis.
type ProcessVideoPayload struct {
	VideoID  string `json:"video_id"`
	Source   string `json:"source"`
	Language string `json:"language"`
}

// NewProcessVideoTask builds an Asynq task for the background worker to pick up.
func NewProcessVideoTask(payload ProcessVideoPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeProcessVideo, data), nil
}

// CancelQueuedTask removes a task that hasn't started processing yet.
func CancelQueuedTask(inspector *asynq.Inspector, queueName, taskID string) error {
	return inspector.DeleteTask(queueName, taskID)
}

// CancelRunningTask signals an actively-processing task's context to cancel.
// The worker handler must be built with that context (e.g. via
// http.NewRequestWithContext) for the cancellation to actually interrupt it.
func CancelRunningTask(inspector *asynq.Inspector, taskID string) error {
	return inspector.CancelProcessing(taskID)
}

// Enqueue submits a new video-processing job onto the default queue and
// returns the resulting TaskInfo, whose ID is needed later to cancel the job.
func Enqueue(ctx context.Context, client *asynq.Client, payload ProcessVideoPayload) (*asynq.TaskInfo, error) {
	task, err := NewProcessVideoTask(payload)
	if err != nil {
		return nil, err
	}
	return client.EnqueueContext(ctx, task, asynq.Queue("default"), asynq.MaxRetry(3))
}
