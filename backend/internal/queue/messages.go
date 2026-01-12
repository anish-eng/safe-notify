package kafkaproducer

type TaskMessage struct {
	TaskID string `json:"task_id"`
}

// RetryMessage includes when it should be retried.
type RetryMessage struct {
	TaskID      string `json:"task_id"`
	NextRetryAt int64  `json:"next_retry_at"` // epoch ms
}
