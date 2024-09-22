package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	
)

var ctx = context.Background()

func newRedisClient() *redis.Client {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})
	return rdb
}

func PersistMessage(c *gin.Context) {
	var msg Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	rdb := newRedisClient()
	key := fmt.Sprintf("messages:%s", msg.Receiver)

	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist message"})
		return
	}

	if err := rdb.LPush(ctx, key, msgData).Err(); err != nil {
		log.Printf("Error saving message to Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message Persisted"})
}

func RetrieveMessageHistory(c *gin.Context) {
	receiver := c.Param("receiver")

	rdb := newRedisClient()

	key := fmt.Sprintf("messages:%s", receiver)
	messages, err := rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		log.Printf("Error retrieving message history from Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	var msgList []Message
	for _, msgData := range messages {
		var msg Message
		if err := json.Unmarshal([]byte(msgData), &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}
		msgList = append(msgList, msg)
	}

	c.JSON(http.StatusOK, gin.H{"messages": msgList})
}

func main() {
	router := gin.Default()
	router.POST("/persist-message", PersistMessage)
	router.GET("/message-history/:receiver", RetrieveMessageHistory)

	port := ":8081"
	if err := router.Run(port); err != nil {
		log.Fatalf("Unable to start server. Error: %v", err)
	}
}
