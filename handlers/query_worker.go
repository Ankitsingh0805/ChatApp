package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	. "bewanderic/chatapp/shared"
)

var ctx = context.Background()

// RetrieveMessageHistory fetches chat history for a specific receiver.
func RetrieveMessageHistory(c *gin.Context, rdb *redis.Client) {
	receiver := c.Param("receiver")
	key := fmt.Sprintf("messages:%s", receiver)

	messages, err := rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		log.Printf("Error retrieving message history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	var msgList []Message
	for _, msgData := range messages {
		var msg Message
		json.Unmarshal([]byte(msgData), &msg)
		msgList = append(msgList, msg)
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgList})
}
