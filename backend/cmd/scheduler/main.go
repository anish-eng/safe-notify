package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	kafkaproducer "safe-notify/internal/queue"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	brokersCSV := getenv("KAFKA_BROKERS", "localhost:9092")
	retryTopic := getenv("KAFKA_TOPIC_RETRY", "safe-notify-retry")
	mainTopic := getenv("KAFKA_TOPIC_MAIN", "safe-notify-tasks")

	groupID := getenv("KAFKA_SCHEDULER_GROUP", "safe-notify-scheduler")

	retryConsumer := kafkaproducer.NewConsumer(splitCSV(brokersCSV), retryTopic, groupID)
	defer retryConsumer.Close()

	mainProducer := kafkaproducer.NewProducer(brokersCSV, mainTopic)
	defer mainProducer.Close()

	log.Println("scheduler: started retryTopic=", retryTopic, "mainTopic=", mainTopic)

	for {
		rm, commit, err := retryConsumer.ReadRetry(ctx)
		if err != nil {
			log.Println("scheduler: read error:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		now := time.Now().UnixMilli()
		if rm.NextRetryAt > now {
			time.Sleep(time.Duration(rm.NextRetryAt-now) * time.Millisecond)
		}

		// publish back to main topic
		if err := mainProducer.PublishTask(ctx, rm.TaskID); err != nil {
			log.Println("scheduler: publish main failed:", err)
			// do not commit; will retry
			continue
		}

		if err := commit(ctx); err != nil {
			log.Println("scheduler: commit error:", err)
		}
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
