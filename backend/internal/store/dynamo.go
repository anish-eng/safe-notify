package store

import (
	"context"
	"errors"
	"fmt"
	"os"

	"safe-notify/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoStore struct {
	db        *dynamodb.Client
	tableName string
}

func NewDynamoStore(ctx context.Context) (*DynamoStore, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-2"
	}

	table := os.Getenv("DYNAMO_TABLE")
	if table == "" {
		return nil, fmt.Errorf("DYNAMO_TABLE is required")
	}

	endpoint := os.Getenv("DYNAMO_ENDPOINT")
	fmt.Println("Dynamo endpoint:", endpoint)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return &DynamoStore{db: client, tableName: table}, nil
}

func (s *DynamoStore) PutTask(ctx context.Context, t models.Task) error {
	item, err := attributevalue.MarshalMap(t)
	if err != nil {
		return err
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	return err
}

func (s *DynamoStore) ListTasks(ctx context.Context, limit int32) ([]models.Task, error) {
	out, err := s.db.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
		Limit:     aws.Int32(limit),
	})
	if err != nil {
		return nil, err
	}

	var tasks []models.Task
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *DynamoStore) GetTaskByID(ctx context.Context, taskID string) (*models.Task, error) {
	out, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"task_id": &types.AttributeValueMemberS{Value: taskID},
		},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}

	var t models.Task
	if err := attributevalue.UnmarshalMap(out.Item, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *DynamoStore) FetchProcessableTasks(ctx context.Context, limit int32) ([]models.Task, error) {
	out, err := s.db.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
		Limit:     aws.Int32(limit),

		// Only pull tasks needing work
		FilterExpression: aws.String("#st = :pending OR #st = :failed"),
		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pending": &types.AttributeValueMemberS{Value: "PENDING"},
			":failed":  &types.AttributeValueMemberS{Value: "FAILED"},
		},
	})
	if err != nil {
		return nil, err
	}

	var tasks []models.Task
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *DynamoStore) ClaimTask(ctx context.Context, taskID string, workerID string, nowMs int64) (bool, error) {
	_, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),

		Key: map[string]types.AttributeValue{
			"task_id": &types.AttributeValueMemberS{Value: taskID},
		},

		// Only claim if it's still PENDING or FAILED
		ConditionExpression: aws.String("#st = :pending OR #st = :failed"),

		UpdateExpression: aws.String("SET #st = :processing, worker_id = :wid, processing_started_at = :psa, updated_at = :u"),

		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pending":    &types.AttributeValueMemberS{Value: "PENDING"},
			":failed":     &types.AttributeValueMemberS{Value: "FAILED"},
			":processing": &types.AttributeValueMemberS{Value: "PROCESSING"},
			":wid":        &types.AttributeValueMemberS{Value: workerID},
			":psa":        &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", nowMs)},
			":u":          &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", nowMs)},
		},
	})

	if err != nil {
		// If condition fails, someone else claimed it (or it changed state)
		var cfe *types.ConditionalCheckFailedException
		if errors.As(err, &cfe) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *DynamoStore) UpdateAfterAttempt(
	ctx context.Context,
	taskID string,
	newStatus string,
	attemptCount int,
	lastError string,
	nowMs int64,
) error {

	_, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"task_id": &types.AttributeValueMemberS{Value: taskID},
		},

		UpdateExpression: aws.String("SET #st = :st, attempt_count = :ac, last_error = :le, updated_at = :u"),
		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":st": &types.AttributeValueMemberS{Value: newStatus},
			":ac": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", attemptCount)},
			":le": &types.AttributeValueMemberS{Value: lastError},
			":u":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", nowMs)},
		},
	})

	return err
}

func (s *DynamoStore) UpdateForRetry(
	ctx context.Context,
	taskID string,
	attemptCount int,
	lastErr string,
	nextRetryAt int64,
	updatedAt int64,
) error {
	_, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"task_id": &types.AttributeValueMemberS{Value: taskID},
		},
		UpdateExpression: aws.String(
			"SET #st=:failed, attempt_count=:ac, last_error=:le, next_retry_at=:nra, updated_at=:ua " +
				"REMOVE worker_id, processing_started_at",
		),
		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":failed": &types.AttributeValueMemberS{Value: "FAILED"},
			":ac":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", attemptCount)},
			":le":     &types.AttributeValueMemberS{Value: lastErr},
			":nra":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", nextRetryAt)},
			":ua":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", updatedAt)},
		},
	})
	return err
}

func (s *DynamoStore) ResetForReplay(ctx context.Context, taskID string, updatedAt int64) error {
	_, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"task_id": &types.AttributeValueMemberS{Value: taskID},
		},
		UpdateExpression: aws.String(
			"SET #st=:pending, attempt_count=:zero, last_error=:empty, next_retry_at=:zr, updated_at=:ua " +
				"REMOVE worker_id, processing_started_at",
		),
		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pending": &types.AttributeValueMemberS{Value: "PENDING"},
			":zero":    &types.AttributeValueMemberN{Value: "0"},
			":empty":   &types.AttributeValueMemberS{Value: ""},
			":zr":      &types.AttributeValueMemberN{Value: "0"},
			":ua":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", updatedAt)},
		},
	})
	return err
}
