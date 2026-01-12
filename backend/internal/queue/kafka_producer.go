// package kafkaproducer

// import (
// 	"context"
// 	"encoding/json"
// 	"strings"
// 	"time"

// 	kgo "github.com/segmentio/kafka-go"
// )

// type Producer struct {
// 	topic   string
// 	writer  *kgo.Writer
// 	timeout time.Duration
// }

// type TaskMessage struct {
// 	TaskID string `json:"task_id"`
// }

// func NewProducer(brokersCSV, topic string) *Producer {
// 	brokers := splitCSV(brokersCSV)

// 	w := &kgo.Writer{
// 		Addr:         kgo.TCP(brokers...),
// 		Topic:        topic,
// 		Balancer:     &kgo.LeastBytes{},
// 		RequiredAcks: kgo.RequireOne, // simple + reliable for demo
// 	}

// 	return &Producer{
// 		topic:   topic,
// 		writer:  w,
// 		timeout: 3 * time.Second,
// 	}
// }

// func (p *Producer) Close() error {
// 	return p.writer.Close()
// }

// func (p *Producer) PublishTaskID(ctx context.Context, taskID string) error {
// 	msg := TaskMessage{TaskID: taskID}
// 	b, err := json.Marshal(msg)
// 	if err != nil {
// 		return err
// 	}

// 	// small timeout so API doesn’t hang forever if Kafka is down
// 	cctx, cancel := context.WithTimeout(ctx, p.timeout)
// 	defer cancel()

// 	return p.writer.WriteMessages(cctx, kgo.Message{
// 		Key:   []byte(taskID), // helps ordering for same task ID
// 		Value: b,
// 		Time:  time.Now(),
// 	})
// }

// func splitCSV(s string) []string {
// 	parts := strings.Split(s, ",")
// 	out := make([]string, 0, len(parts))
// 	for _, p := range parts {
// 		p = strings.TrimSpace(p)
// 		if p != "" {
// 			out = append(out, p)
// 		}
// 	}
// 	return out
// }

// new code for egenric producer starts here
package kafkaproducer

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	kgo "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer  *kgo.Writer
	timeout time.Duration
}

func NewProducer(brokersCSV, topic string) *Producer {
	brokers := splitCSV(brokersCSV)
	if topic == "" {
		panic("KAFKA_TOPIC is required (topic cannot be empty)")
		// or return nil and handle error, but panic is fine for early dev
	}

	w := &kgo.Writer{
		Addr:         kgo.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kgo.LeastBytes{},
		RequiredAcks: kgo.RequireOne,
	}

	return &Producer{
		writer:  w,
		timeout: 3 * time.Second,
	}
}

func (p *Producer) Close() error { return p.writer.Close() }

func (p *Producer) PublishTask(ctx context.Context, taskID string) error {
	msg := TaskMessage{TaskID: taskID}
	return p.publishJSON(ctx, taskID, msg)
}

func (p *Producer) PublishRetry(ctx context.Context, taskID string, nextRetryAt int64) error {
	msg := RetryMessage{TaskID: taskID, NextRetryAt: nextRetryAt}
	return p.publishJSON(ctx, taskID, msg)
}

func (p *Producer) publishJSON(ctx context.Context, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	cctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	return p.writer.WriteMessages(cctx, kgo.Message{
		Key:   []byte(key),
		Value: b,
		Time:  time.Now(),
	})
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
