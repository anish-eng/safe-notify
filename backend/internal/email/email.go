package email

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type Sender interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

type SESSender struct {
	client    *sesv2.Client
	fromEmail string
}

func NewSESSender(cfg aws.Config) (*SESSender, error) {
	from := os.Getenv("SES_FROM_EMAIL")
	if from == "" {
		return nil, fmt.Errorf("SES_FROM_EMAIL is not set")
	}
	return &SESSender{
		client:    sesv2.NewFromConfig(cfg),
		fromEmail: from,
	}, nil
}

func (s *SESSender) Send(ctx context.Context, to, subject, body string) error {
	_, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.fromEmail),
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body)},
				},
			},
		},
	})
	return err
}
