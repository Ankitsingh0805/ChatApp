package main

import (
	"context" // Import the context package
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"bewanderic/chatapp/client"   // Corrected the import to match client package
	"bewanderic/chatapp/handlers"
)

func main() {
	// Setup Redis client
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})

	// Background goroutine for Redis Pub/Sub broadcasting
	go func() {
		sub := rdb.Subscribe(context.Background(), "chat_room")
		for {
			msg, err := sub.ReceiveMessage(context.Background())
			if err != nil {
				log.Println("Error receiving message:", err)
				continue
			}
			client.Broadcast([]byte(msg.Payload)) // Broadcast to WebSocket clients
		}
	}()

	router := gin.Default()
	router.StaticFile("/", "./static/index.html")

	// WebSocket route
	router.GET("/ws", func(c *gin.Context) {
		client.HandleWebSocket(c, rdb) // Correct package name: client.HandleWebSocket
	})

	// REST routes for message persistence and retrieval
	router.POST("/persist-message", func(c *gin.Context) {
		handlers.PersistMessage(c, rdb)
	})
	router.GET("/message-history/:receiver", func(c *gin.Context) {
		handlers.RetrieveMessageHistory(c, rdb)
	})

	// Start the server
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Unable to start server. Error: %v", err)
	}
}





