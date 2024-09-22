package client

import (
	"context"
	"encoding/json"
	"log"
	"time"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	. "bewanderic/chatapp/shared"
)

var (
    clients        = make(map[*websocket.Conn]struct{})
    channel        = "chat_room"
    maxChatMessages int = 300 // Changed name to make it unique
)

// HandleWebSocket upgrades HTTP connection to WebSocket and sends history.
func HandleWebSocket(c *gin.Context, rdb *redis.Client) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Send chat history to the newly connected client
	sendHistory(conn, rdb)

	// Add the client to the active client list
	clients[conn] = struct{}{}
	defer func() {
		delete(clients, conn)
		log.Printf("Client disconnected: %v", conn.RemoteAddr())
	}()

	// Handle message reading and broadcasting
	handleClient(conn, rdb)
}

func sendHistory(conn *websocket.Conn, rdb *redis.Client) {
    // Convert maxChatMessages to int64 for Redis operation
    messages, err := rdb.LRange(context.Background(), "chat_messages", 0, int64(maxChatMessages)-1).Result()
    if err != nil {
        log.Printf("Error retrieving history: %v", err)
        return
    }
    for _, msg := range messages {
        conn.WriteMessage(websocket.TextMessage, []byte(msg))
    }
}
// handleClient reads messages from the client and broadcasts them to all.
func handleClient(conn *websocket.Conn, rdb *redis.Client) {
	for {
		// Read the message from WebSocket connection
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			// If the connection is closed or an error occurs, break the loop
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected WebSocket closure: %v", err)
			}
			break
		}

		// Set the current timestamp to the message
		msg.Timestamp = time.Now()

		// Convert the message to JSON format
		msgData, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			return
		}

		// Push the message into Redis and broadcast to others
		if err := persistAndBroadcastMessage(msgData, rdb); err != nil {
			log.Printf("Error broadcasting message: %v", err)
			return
		}
	}
}


func persistAndBroadcastMessage(msgData []byte, rdb *redis.Client) error {
    err := rdb.LPush(context.Background(), "chat_messages", msgData).Err()
    if err != nil {
        return err
    }
    // Convert maxChatMessages to int64 for Redis operation
    rdb.LTrim(context.Background(), "chat_messages", 0, int64(maxChatMessages)-1)

    // Publish the message to the Redis Pub/Sub channel
    err = rdb.Publish(context.Background(), channel, string(msgData)).Err()
    if err != nil {
        return err
    }

    // Broadcast the message to all connected WebSocket clients
    Broadcast(msgData)
    return nil
}

// Broadcast sends a message to all connected clients.
func Broadcast(message []byte) {
	for conn := range clients {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error sending message to client: %v", err)
			conn.Close()        // Close the connection if sending fails
			delete(clients, conn) // Remove the client from the active list
		}
	}
}



