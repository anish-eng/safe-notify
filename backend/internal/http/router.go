package httpapi

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, app *App) {
	r.Get("/healthz", healthHandler)
	r.Get("/notifications", app.listNotificationsHandler)
	r.Post("/events", app.createEvent)
	r.Post("/tasks/{task_id}/replay", app.ReplayTaskHandler)
}
