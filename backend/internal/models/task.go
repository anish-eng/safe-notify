package models

type Task struct {
	// Keys
	TaskID         string `dynamodbav:"task_id" json:"task_id"`
	IdempotencyKey string `dynamodbav:"idempotency_key" json:"idempotency_key"`

	// Business
	EventType      string `dynamodbav:"event_type" json:"event_type"`
	EntityID       string `dynamodbav:"entity_id" json:"entity_id"`
	Channel        string `dynamodbav:"channel" json:"channel"`
	RecipientEmail string `dynamodbav:"recipient_email" json:"recipient_email"`
	Priority       string `dynamodbav:"priority" json:"priority"`

	// Processing/Status
	Status       string `dynamodbav:"status" json:"status"`
	AttemptCount int    `dynamodbav:"attempt_count" json:"attempt_count"`
	MaxAttempts  int    `dynamodbav:"max_attempts" json:"max_attempts"`
	LastError    string `dynamodbav:"last_error" json:"last_error"`

	// Demo-only (chaos)
	ChaosFailPercent int `dynamodbav:"chaos_fail_percent" json:"chaos_fail_percent"`

	// Timestamps (store as epoch ms, easy + consistent)
	CreatedAt           int64  `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt           int64  `dynamodbav:"updated_at" json:"updated_at"`
	WorkerID            string `dynamodbav:"worker_id" json:"worker_id"`
	ProcessingStartedAt int64  `dynamodbav:"processing_started_at" json:"processing_started_at"`
	NextRetryAt         int64  `dynamodbav:"next_retry_at" json:"next_retry_at"`
}
