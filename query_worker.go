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

func ReadAllMessages(c *gin.Context) {
	receiver := c.Param("receiver")

	rdb := newRedisClient()
	defer rdb.Close()

	key := fmt.Sprintf("messages:%s", receiver)
	messages, err := rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		log.Printf("Error retrieving messages for %s: %v", receiver, err)
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
	router.GET("/messages/:receiver", ReadAllMessages)

	port := ":1122"
	if err := router.Run(port); err != nil {
		log.Fatalf("Unable to start server. Error: %v", err)
	}
}
