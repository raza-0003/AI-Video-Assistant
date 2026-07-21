package queue

import (
	"github.com/hibiken/asynq"
)

const TypeProcessVideo = "video:process"

// NewClient creates an Asynq client used by the API server to enqueue jobs.
func NewClient(redisAddr string) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
}

// NewInspector creates an Asynq inspector used to cancel or delete jobs by ID —
// e.g. when a user hits "Cancel" on an in-progress or still-queued video.
func NewInspector(redisAddr string) *asynq.Inspector {
	return asynq.NewInspector(asynq.RedisClientOpt{Addr: redisAddr})
}

// NewServer creates an Asynq server used by the worker process to consume jobs.
// Three priority queues let critical work (e.g. retries) jump ahead of bulk processing.
func NewServer(redisAddr string) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 5,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)
}
