// package httpapi

// import "safe-notify/internal/store"

// type App struct {
// 	Store *store.DynamoStore
// }

package httpapi

import (
	kafkaproducer "safe-notify/internal/queue"
	"safe-notify/internal/store"
)

type App struct {
	Store         *store.DynamoStore
	TasksProducer *kafkaproducer.Producer // publishes to safe-notify-tasks
}
