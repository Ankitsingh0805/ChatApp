package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	channel      = "chat_room"     // A single chat room for all users
	messageList  = "chat_messages" // Redis key for storing chat history
	maxMessages  = 300             // Max number of historical messages to store
	sessionStart = "session_start"
	sessionEnd   = "session_end"
)

var clients = make(map[*websocket.Conn]struct{})

type Message struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})

	go func() {
		ctx := context.Background()
		sub := rdb.Subscribe(ctx, channel)
		for {
			message, err := sub.ReceiveMessage(ctx)
			if err != nil {
				fmt.Println("Error receiving message", err)
			}
			if message != nil {
				broadcast([]byte(message.Payload))
			}
		}
	}()

	router := gin.Default()
	router.StaticFile("/", "./static/index.html")
	router.GET("/ws", serveWs(rdb)) // WebSocket endpoint
	err := router.Run(":8080")
	if err != nil {
		log.Fatalf("Unable to start server. Error %v", err)
	}
	log.Println("Server started successfully.")
}

func serveWs(rdb *redis.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Error upgrading WebSocket. Error: %v", err)
			return
		}

		sendHistory(conn, rdb)
		go handleClient(conn, rdb)
	}
}

func sendHistory(conn *websocket.Conn, rdb *redis.Client) {
	ctx := context.Background()
	messages, err := rdb.LRange(ctx, messageList, 0, maxMessages-1).Result()
	if err != nil {
		log.Println("Error retrieving chat history:", err)
		return
	}

	for _, msg := range messages {
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
	}
}

func broadcast(msgBytes []byte) {
	for conn := range clients {
		conn.WriteMessage(websocket.TextMessage, msgBytes)
	}
}

func handleClient(c *websocket.Conn, rdb *redis.Client) {
	defer func() {
		delete(clients, c)
		broadcastSessionChange("A user left the chat", sessionEnd, rdb)
		log.Println("Closing WebSocket")
		c.Close()
	}()
	clients[c] = struct{}{}

	broadcastSessionChange("A new user joined the chat", sessionStart, rdb)

	for {
		var msg Message
		err := c.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading JSON message. Error: %v", err)
			return
		}

		msg.Timestamp = time.Now()
		msg.Type = "chat"

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshaling:", err.Error())
			return
		}

		ctx := context.Background()
		err = rdb.LPush(ctx, messageList, msgBytes).Err()
		if err != nil {
			fmt.Println("Error storing message:", err.Error())
		}

		rdb.LTrim(ctx, messageList, 0, maxMessages-1)
		err = rdb.Publish(ctx, channel, string(msgBytes)).Err()
		if err != nil {
			fmt.Println("Error publishing:", err.Error())
		}
	}
}

func broadcastSessionChange(message string, messageType string, rdb *redis.Client) {
	sessionMessage := Message{
		Sender:    "System",
		Text:      message,
		Timestamp: time.Now(),
		Type:      messageType,
	}

	msgBytes, err := json.Marshal(sessionMessage)
	if err != nil {
		fmt.Println("Error marshaling session message:", err.Error())
		return
	}

	broadcast(msgBytes)
}




