package shared

import "time"

type Message struct {
	ID        string    `json:"id"`       // Unique message ID
	Text      string    `json:"text"`     // The message content
	Sender    string    `json:"sender"`   // Sender's user ID
	Receiver  string    `json:"receiver"` // Receiver's user ID or group ID
	Type      string    `json:"type"`     // Type of message (e.g., "chat", "system")
	Timestamp time.Time `json:"timestamp"`// Timestamp for the message
}
