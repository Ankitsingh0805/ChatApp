package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	serverAddr := "ws://localhost:8080/ws"
	if len(os.Args) > 1 {
		serverAddr = os.Args[1]
	}

	conn, _, err := websocket.DefaultDialer.Dial(serverAddr, nil)
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	defer conn.Close()

	go func() {
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message:", err)
				return
			}
			var msg Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				log.Println("Error unmarshaling message:", err)
				continue
			}
			fmt.Printf("[%s] %s: %s\n", msg.Timestamp.Format(time.RFC3339), msg.Sender, msg.Text)
		}
	}()

	for {
		var input string
		fmt.Print("Enter message (or 'exit' to quit): ")
		fmt.Scanln(&input)
		if input == "exit" {
			break
		}

		msg := Message{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			Text:      input,
			Sender:    "User1", // Replace with actual user ID
			Receiver:  "User2", // Replace with actual receiver ID
			Type:      "chat",
			Timestamp: time.Now(),
		}

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Println("Error marshaling message:", err)
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			log.Println("Error sending message:", err)
		}
	}
}

