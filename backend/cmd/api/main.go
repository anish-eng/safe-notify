package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"os"
	httpapi "safe-notify/internal/http"
	kafkaproducer "safe-notify/internal/queue"
	"safe-notify/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func main() {
	_ = godotenv.Load()
	log.Println("DYNAMO_TABLE =", os.Getenv("DYNAMO_TABLE"))
	log.Println("DYNAMO_ENDPOINT =", os.Getenv("DYNAMO_ENDPOINT"))
	ctx := context.Background()
	st, err := store.NewDynamoStore(ctx)
	if err != nil {
		log.Fatal("failed to init dynamo store:", err)
	}
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	kafkaTopic := os.Getenv("KAFKA_TOPIC_TASKS")
	fmt.Println(kafkaTopic)
	prod := kafkaproducer.NewProducer(kafkaBrokers, kafkaTopic)
	defer prod.Close()

	app := &httpapi.App{
		Store:         st,
		TasksProducer: prod,
	}
	// 2) create your app container (dependencies)
	// app := &httpapi.App{
	// 	Store: st,
	// }
	r := chi.NewRouter()
	// rand.Seed(time.Now().UnixNano())
	// basic middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	httpapi.RegisterRoutes(r, app)

	log.Println("API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))

}
