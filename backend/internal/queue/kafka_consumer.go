// package kafkaproducer

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"time"

// 	kgo "github.com/segmentio/kafka-go"
// )

// type Consumer struct {
// 	reader *kgo.Reader
// }

// func NewConsumer(brokers []string, topic, groupID string) *Consumer {
// 	r := kgo.NewReader(kgo.ReaderConfig{
// 		Brokers:  brokers,
// 		Topic:    topic,
// 		GroupID:  groupID,
// 		MinBytes: 1,
// 		MaxBytes: 10e6,
// 		// CommitInterval: 0 means we commit manually by calling CommitMessages
// 		CommitInterval: 0,
// 	})

// 	return &Consumer{reader: r}
// }

// func (c *Consumer) Close() error {
// 	return c.reader.Close()
// }

// // ReadTaskID blocks until it gets a message.
// // It returns the task_id and a commit function you should call AFTER successful processing.
// func (c *Consumer) ReadTaskID(ctx context.Context) (taskID string, commit func(context.Context) error, err error) {
// 	m, err := c.reader.FetchMessage(ctx)
// 	if err != nil {
// 		return "", nil, err
// 	}

// 	var tm TaskMessage
// 	if err := json.Unmarshal(m.Value, &tm); err != nil || tm.TaskID == "" {
// 		// Bad message: commit it so we don't get stuck re-reading it forever
// 		_ = c.reader.CommitMessages(ctx, m)
// 		if err == nil {
// 			err = errors.New("invalid message: missing task_id")
// 		}
// 		return "", nil, err
// 	}

// 	commitFn := func(cctx context.Context) error {
// 		// add a small timeout for commit
// 		cc, cancel := context.WithTimeout(cctx, 3*time.Second)
// 		defer cancel()
// 		return c.reader.CommitMessages(cc, m)
// 	}

//		return tm.TaskID, commitFn, nil
//	}
//
// working new code
package kafkaproducer

import (
	"context"
	"encoding/json"
	"time"

	kgo "github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kgo.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	r := kgo.NewReader(kgo.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0, // manual commits
	})

	return &Consumer{reader: r}
}

func (c *Consumer) Close() error { return c.reader.Close() }

// ReadTask consumes TaskMessage.
func (c *Consumer) ReadTask(ctx context.Context) (TaskMessage, func(context.Context) error, error) {
	m, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return TaskMessage{}, nil, err
	}

	var tm TaskMessage
	if err := json.Unmarshal(m.Value, &tm); err != nil {
		// commit bad messages so you don't get stuck forever
		_ = c.reader.CommitMessages(ctx, m)
		return TaskMessage{}, nil, err
	}

	commit := func(ctx context.Context) error {
		// small safety timeout
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		return c.reader.CommitMessages(cctx, m)
	}

	return tm, commit, nil
}

// ReadRetry consumes RetryMessage.
func (c *Consumer) ReadRetry(ctx context.Context) (RetryMessage, func(context.Context) error, error) {
	m, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return RetryMessage{}, nil, err
	}

	var rm RetryMessage
	if err := json.Unmarshal(m.Value, &rm); err != nil {
		_ = c.reader.CommitMessages(ctx, m)
		return RetryMessage{}, nil, err
	}

	commit := func(ctx context.Context) error {
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		return c.reader.CommitMessages(cctx, m)
	}

	return rm, commit, nil
}
