package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/joho/godotenv"

	"safe-notify/internal/email"
	kafkaproducer "safe-notify/internal/queue"
	"safe-notify/internal/store"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	workerID := getenv("WORKER_ID", "worker-1")

	// Dynamo store (source of truth)
	st, err := store.NewDynamoStore(ctx)
	if err != nil {
		log.Fatal("worker: init dynamo:", err)
	}

	// SES sender (delivery)
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal("worker: load aws cfg:", err)
	}
	sender, err := email.NewSESSender(awsCfg)
	if err != nil {
		log.Fatal("worker: init ses:", err)
	}

	// Kafka config
	brokersCSV := getenv("KAFKA_BROKERS", "localhost:9092")
	mainTopic := getenv("KAFKA_TOPIC_MAIN", "safe-notify-tasks")
	retryTopic := getenv("KAFKA_TOPIC_RETRY", "safe-notify-retry")
	groupID := getenv("KAFKA_GROUP_ID", "safe-notify-workers")

	// Consume main topic (work queue)
	mainConsumer := kafkaproducer.NewConsumer(splitCSV(brokersCSV), mainTopic, groupID)
	defer mainConsumer.Close()

	// Produce retry messages (delayed retry queue)
	retryProducer := kafkaproducer.NewProducer(brokersCSV, retryTopic)
	defer retryProducer.Close()

	log.Println("worker: started",
		"workerID=", workerID,
		"mainTopic=", mainTopic,
		"retryTopic=", retryTopic,
		"brokers=", brokersCSV,
	)

	for {
		// 1) Read one task message from the MAIN topic
		tm, commit, err := mainConsumer.ReadTask(ctx)
		if err != nil {
			log.Println("worker: read error:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// 2) Process the task
		if err := processOne(ctx, st, sender, workerID, tm.TaskID, retryProducer); err != nil {
			log.Println("worker: process error:", err)
			// IMPORTANT: if system fails BEFORE scheduling retry, DO NOT commit.
			// Kafka will redeliver this same message later.
			continue
		}

		// 3) Commit offset ONLY after success (or successful retry scheduling / DLQ marking)
		if err := commit(ctx); err != nil {
			log.Println("worker: commit error:", err)
			// not fatal; could cause reprocessing; idempotency/claim protects you
		}
	}
}

func processOne(
	ctx context.Context,
	st *store.DynamoStore,
	sender email.Sender,
	workerID, taskID string,
	retryProducer *kafkaproducer.Producer,
) error {
	// Load full task from Dynamo (truth)
	task, err := st.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		// Task missing/deleted: nothing to do; safe to commit Kafka message
		return nil
	}

	// OPTIONAL SAFETY:
	// If retry was scheduled, don't process before NextRetryAt
	now := time.Now().UnixMilli()
	if task.NextRetryAt > 0 && now < task.NextRetryAt {
		// This can happen if a duplicate/early message exists.
		// Ignore it if it happens; scheduler will publish later.
		return nil
	}

	// Claim the task so only this worker can process it (prevents double-send of messages)
	claimed, err := st.ClaimTask(ctx, task.TaskID, workerID, now)
	if err != nil {
		return err
	}
	if !claimed {
		// Someone else owns it / it already moved states; safe to commit message
		return nil
	}

	// Attempt delivery (chaos + SES)
	ok, errMsg := attemptSend(ctx, sender, *task)

	newAttempt := task.AttemptCount + 1

	// Success path
	if ok {
		return st.UpdateAfterAttempt(ctx, task.TaskID, "SENT", newAttempt, "", time.Now().UnixMilli())
	}

	// Failure path
	max := task.MaxAttempts
	if max <= 0 {
		max = 3
	}

	// Terminal failure => DLQ state in Dynamo (NO Kafka DLQ topic)
	if newAttempt >= max {
		return st.UpdateAfterAttempt(ctx, task.TaskID, "DLQ", newAttempt, errMsg, time.Now().UnixMilli())
	}

	// Not terminal => schedule retry via retry topic
	backoff := computeBackoffMs(newAttempt)         // e.g. attempt1=2s, attempt2=5s
	nextRetryAt := time.Now().UnixMilli() + backoff // epoch ms

	// Update Dynamo so UI shows FAILED + next_retry_at
	if err := st.UpdateForRetry(ctx, task.TaskID, newAttempt, errMsg, nextRetryAt, time.Now().UnixMilli()); err != nil {
		return err
	}

	// Publish retry message so scheduler can re-enqueue later
	if err := retryProducer.PublishRetry(ctx, task.TaskID, nextRetryAt); err != nil {
		// If Kafka publish fails, return error so we DON'T commit.
		// Kafka will redeliver the main message and we'll try scheduling again.
		return err
	}

	return nil
}

func computeBackoffMs(attempt int) int64 {
	// Simple + defensible for demo
	// attempt=1 => 2s, attempt=2 => 5s, attempt=3 => terminal DLQ handled above
	switch attempt {
	case 1:
		return 2000
	case 2:
		return 5000
	default:
		return 10000
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
