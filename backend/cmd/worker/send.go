package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"safe-notify/internal/email"
	"safe-notify/internal/models"
)

// attemptSend returns:
// - ok = true if delivery succeeded
// - ok = false if delivery failed, with an error message
//
// It first applies chaos injection (demo), then sends a real email via AWS SES.
func attemptSend(ctx context.Context, sender email.Sender, task models.Task) (bool, string) {
	// Chaos simulation (demo feature)
	p := task.ChaosFailPercent
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if r.Intn(100) < p {
		return false, "CHAOS injected failure"
	}

	// Real email via SES
	subject := fmt.Sprintf("[Safe-Notify] %s (%s)", task.EventType, task.EntityID)
	body := fmt.Sprintf(
		"TaskID: %s\nEventType: %s\nEntityID: %s\nPriority: %s\nChannel: %s\n",
		task.TaskID, task.EventType, task.EntityID, task.Priority, task.Channel,
	)

	if err := sender.Send(ctx, task.RecipientEmail, subject, body); err != nil {
		return false, "SES send failed: " + err.Error()
	}

	return true, ""
}
