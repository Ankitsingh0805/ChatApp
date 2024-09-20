package main 

import(
	"log"
	"net/http"
	"net/rpc"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/ygodefraga/real-time-chat/shared"
)

var (
	// The Upgrader is used to upgrade a regular HTTP connection to a WebSocket connection.
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool { return true }, // A function that determines whether the client is authorized to upgrade the HTTP connection to WebSocket
	}
	chat_broadcast = make(chan Message) // This is a channel that will be used to broadcast messages to all clients connected to the WebSocket server.
	persist_broadcast = make(chan Message) // This is a channel that will be used to broadcast messages to all clients connected to the WebSocket server.
	historical_broadcast = make(chan Message) // This is the channel used if some user who joined the chat late will be able to see the chat happened before in that chat room
	clients   = make(map[string]*websocket.Conn) // This is a map that associates client IDs with websocket.Conn objects (pointers). It is used to track all active WebSocket connections on the server.
)


func handleConnections(w http.ResponseWriter, r *http.Request) {
	/*
		The Upgrade function of the upgrader does exactly that: it upgrades a normal HTTP connection to a WebSocket connection. This function takes three arguments:

		w: The http.ResponseWriter object used to write the HTTP response.
		r: The http.Request object that represents the received HTTP request.
		nil: Optionally, an http.Header object that may contain additional headers for the upgrade.
	*/
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	// This is the part of the code responsible for reading messages from the newly connected client
	// (conn.ReadJSON(&msg)) and sending them to the chat_broadcast channel so they can be distributed to other connected clients.
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			return
		}

		if msg.Type == "new_client" {  // When a new client joins
			if addNewUser(msg, conn) {  // Add the new user to the system
				go readAllMessagesFromRPC(msg.Sender)  // Fetch all messages for the client from a remote server (likely for chat history)
			}
		} else if msg.Type == "chat" {  // If the message is a chat message
			chat_broadcast <- msg  // Broadcast the chat message to all connected clients
			persist_broadcast <- msg  // Persist the chat message (likely for storage in a database)
		} else if msg.Type == "session_end" {  // When the session ends
			go removeClient(msg.Sender)  // Remove the client from the system
		} else {
			log.Println("Unknown message:", msg.Type)  // Log any unknown message types for debugging
		}
	}
}


func addNewUser(msg Message, conn *websocket.Conn) bool {
	// Add the client to the map of clients
	clientID := msg.Text

	// Check if the client ID is already in use
	if _, exists := clients[clientID]; exists {
		// Client ID already in use, return an error
		log.Printf("User already exists: %s\n", clientID)
		err := conn.WriteJSON(Message{
			Text:     "User already exists",
			Sender:   "server",
			Receiver: clientID,
			Type:     "error",
			Timestamp: time.Now(),
		})
		if err != nil {
			log.Println("Error sending error message:", err)
		}
		return false
	} else {
		clients[clientID] = conn
		log.Printf("New user: %s\n", clientID)
		return true
	}
}


func handleChatMessages() {
    for {
        msg := <- chat_broadcast

		log.Printf("Sending message from client %s to client %s: %s\n", msg.Sender, msg.Receiver, msg.Text)
		// Check if the recipient is online and send the message only to them
		if conn, ok := clients[msg.Receiver]; ok {
			err := conn.WriteJSON(msg)
			if err != nil {
				log.Println(err)
				conn.Close()
				delete(clients, msg.Receiver)
			}
		}
    }
}


func handleHistoricalMessages() {
    for {
        msg := <- historical_broadcast
		// Check if the recipient is online and send the message only to them
		if conn, ok := clients[msg.Receiver]; ok {
			err := conn.WriteJSON(msg)
			if err != nil {
				log.Println(err)
				conn.Close()
				delete(clients, msg.Receiver)
			}
		}
    }
}

func forwardMessagesToRPC() {
    for {
		msg := <-persist_broadcast

        client, err := rpc.DialHTTP("tcp", "localhost:1123")
        if err != nil {
            log.Println("Failed to connect to RPC server:", err)
			continue
        }
        defer client.Close()

		var reply string
		err = client.Call("MessageRPCServer.PersistMessage", msg, &reply)
		if err != nil {
			log.Println("Error calling RPC service:", err)
			continue // Exit the internal loop and attempt to reconnect to the RPC server
		}
		log.Println("RPC service response:", reply)
	}
}

func readAllMessagesFromRPC(receiver string) {
	client, err := rpc.DialHTTP("tcp", "localhost:1122") // Assuming RPC server is running on localhost:1122
	if err != nil {
		log.Println("Error connecting to Query server:", err)
		return
	}
	defer client.Close()

	var reply []Message
	args := struct{ Receiver string }{Receiver: receiver} // Create and initialize the struct
	err = client.Call("MessageRPCServer.ReadAllMessages", args, &reply)
	if err != nil {
		log.Fatal("Error calling RPC service:", err)
	}

	// Process the reply, which contains all messages
	for _, msg := range reply {
		historical_broadcast <- msg
	}
}

func removeClient(clientID string) {
	delete(clients, clientID)
	log.Printf("Client '%s' removed\n", clientID)
}

func main() {
	// Route configuration
	http.HandleFunc("/ws", handleConnections)

	// Start the server
	log.Println("Server started on port 8080")

	go handleHistoricalMessages()
	go handleChatMessages()
	go forwardMessagesToRPC()
	
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error starting the server: ", err)
	}
}




