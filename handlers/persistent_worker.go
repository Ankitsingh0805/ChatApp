package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	. "bewanderic/chatapp/shared"
)

// PersistMessage stores a message in Redis.
func PersistMessage(c *gin.Context, rdb *redis.Client) {
	var msg Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	msg.Timestamp = time.Now()
	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist message"})
		return
	}

	key := "messages:" + msg.Receiver
	rdb.LPush(context.Background(), key, msgData)
	c.JSON(http.StatusOK, gin.H{"message": "Message Persisted"})
}
