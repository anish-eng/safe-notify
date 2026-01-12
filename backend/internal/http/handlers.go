package httpapi

import (
	"encoding/json"
	"net/http"
	"safe-notify/internal/models"
	"time"

	"fmt"
	"math/rand"

	"github.com/go-chi/chi/v5"
)

type CreateEventRequest struct {
	EventType        string `json:"eventType"`
	EntityID         string `json:"entityId"`
	RecipientEmail   string `json:"recipientEmail"`
	Priority         string `json:"priority"`
	ChaosFailPercent int    `json:"chaosFailPercent"`
}

type CreateEventResponse struct {
	TaskID         string `json:"task_id"`
	IdempotencyKey string `json:"idempotency_key"`
	EntityID       string `json:"entity_id"`
}

type ListNotificationsResponse struct {
	Items []any `json:"items"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *App) listNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := a.Store.ListTasks(r.Context(), 50)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load tasks"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tasks})
}

// func createEvent(w http.ResponseWriter, r *http.Request) {
// 	var req CreateEventRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
// 		return
// 	}

// 	// minimal validation defaults
// 	if req.EventType == "" {
// 		req.EventType = "ticket_escalated"
// 	}
// 	if req.EntityID == "" {
// 		req.EntityID = "TICKET-XXXX" // placeholder for now
// 	}
// 	if req.RecipientEmail == "" {
// 		req.RecipientEmail = "demo@example.com"
// 	}
// 	channel := "EMAIL"

// 	// For now: fake task id + deterministic idempotency key
// 	taskID := fmt.Sprintf("task_%04d", rand.Intn(10000))
// 	idKey := req.EventType + ":" + req.EntityID + ":" + channel + ":" + req.RecipientEmail

// 	writeJSON(w, http.StatusOK, CreateEventResponse{
// 		TaskID:         taskID,
// 		IdempotencyKey: idKey,
// 		EntityID:       req.EntityID,
// 	})
// }

func (a *App) createEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if req.EventType == "" {
		req.EventType = "ticket_escalated"
	}
	if req.EntityID == "" {
		req.EntityID = "TICKET-XXXX"
	}
	if req.RecipientEmail == "" {
		req.RecipientEmail = "demo@example.com"
	}
	if req.Priority == "" {
		req.Priority = "HIGH"
	}

	taskID := fmt.Sprintf("task_%04d", rand.Intn(10000))
	channel := "EMAIL"
	idKey := fmt.Sprintf("%s:%s:%s:%s", req.EventType, req.EntityID, channel, req.RecipientEmail)

	now := time.Now().UnixMilli()

	task := models.Task{
		TaskID:           taskID,
		IdempotencyKey:   idKey,
		EventType:        req.EventType,
		EntityID:         req.EntityID,
		Channel:          channel,
		RecipientEmail:   req.RecipientEmail,
		Priority:         req.Priority,
		Status:           "PENDING",
		AttemptCount:     0,
		MaxAttempts:      3,
		LastError:        "",
		ChaosFailPercent: req.ChaosFailPercent,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := a.Store.PutTask(r.Context(), task); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store task"})
		return
	}

	// Publish work item to Kafka
	if err := a.TasksProducer.PublishTask(r.Context(), taskID); err != nil {
		http.Error(w, "failed to publish to kafka: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, CreateEventResponse{
		TaskID:         task.TaskID,
		IdempotencyKey: task.IdempotencyKey,
		EntityID:       task.EntityID,
	})
}
func (a *App) ReplayTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "task_id")
	if taskID == "" {
		http.Error(w, "task_id required", http.StatusBadRequest)
		return
	}

	nowMs := time.Now().UnixMilli()

	// 1) Reset Dynamo record
	if err := a.Store.ResetForReplay(r.Context(), taskID, nowMs); err != nil {
		http.Error(w, "failed to reset task: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2) Publish to Kafka main topic so worker picks it up
	if err := a.TasksProducer.PublishTask(r.Context(), taskID); err != nil {
		http.Error(w, "failed to publish to kafka: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3) Return something useful to UI
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"task_id": taskID,
	})
}
