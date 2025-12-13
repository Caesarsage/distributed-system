package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"

)

// NewChatRoom creates a new ChatRoom
func NewChatRoom() *ChatRoom {
	return &ChatRoom{
		clients:       make(map[*Client]bool),
		join:          make(chan *Client),
		leave:         make(chan *Client),
		broadcast:     make(chan string),
		listUsers:     make(chan *Client),
		directMessage: make(chan DirectMessage),
	}
}

// SEVER
func runServer() {
	chatRoom := NewChatRoom()

	go chatRoom.Run() // Start the chat room

	listener, err := net.Listen("tcp", ":9000")

	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Chat server started on :9000")
	fmt.Println("Waiting for client connections...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		// New connection
		go handleNewClient(conn, chatRoom)
	}
}

// CHATROOM OPERATIONS
// This loop runs FOREVER, waiting for messages
func (ch *ChatRoom) Run() {
	fmt.Println("Chat room is now running...")

	for {
		select {
		//	receiving client join
		case client := <-ch.join:
			ch.joinChat(client)

			// receiving client leave
		case client := <-ch.leave:
			ch.leaveChat(client)

		case message := <-ch.broadcast:
			ch.broadcastMessage(message)

		case client := <-ch.listUsers:
			ch.sendUserList(client)

		case dm := <-ch.directMessage:
			select {
			case dm.toClient.outgoing <- dm.message:
				// sent successfully
			default:
				fmt.Printf("Count not send DM to %s (channel full)\n", dm.toClient.username)
			}
		}
	}
}

func (ch *ChatRoom) sendUserList(requestingClient *Client) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	fmt.Printf("Sending user list to %s client", requestingClient.username)

	list := "Users online:\n"

	for client := range ch.clients {
		list += " - " + client.username + "\n"
	}

	// requestingClient.outgoing <- list
	// using select to avoid blocking
	select {
	case requestingClient.outgoing <- list:
	default:
		fmt.Printf("Couldn't send user list to %s\n", requestingClient.username)
	}
}

func (ch *ChatRoom) broadcastMessage(message string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	fmt.Printf("Broadcasting message to %d clients: %s", len(ch.clients), message)

	for client := range ch.clients {
		select {
		case client.outgoing <- message:
		default:
			fmt.Printf("Could not send to %s (channel full)\n", client.username)
		}
	}
}

func (ch *ChatRoom) leaveChat(client *Client) {
	ch.mu.Lock()

	if ch.clients[client] {
		delete(ch.clients, client) // Remove client from the map
		close(client.outgoing)     // Close the outgoing channel to stop the writer goroutine
	}
	ch.mu.Unlock()

	fmt.Printf("%s left the chat (Total: %d)\n", client.username, len(ch.clients))

	announcement := fmt.Sprintf("%s has left the chat\n", client.username)
	ch.broadcastMessage(announcement)

}

func (ch *ChatRoom) joinChat(client *Client) {
	ch.mu.Lock()
	ch.clients[client] = true // Add client to the map to mark as connected
	ch.mu.Unlock()

	fmt.Printf("%s joined the chat (Total: %d)\n", client.username, len(ch.clients))

	announcement := fmt.Sprintf("%s has joined the chat\n", client.username)
	ch.broadcastMessage(announcement)
}

func (ch *ChatRoom) findClientByUsername(username string) *Client {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	for client := range ch.clients {
		if client.username == username {
			return client
		}
	}

	return nil
}

func handleNewClient(conn net.Conn, chatRoom *ChatRoom) {
	//1. Get the username
	reader := bufio.NewReader(conn)
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading username:", err)
		conn.Close()
		return
	}
	username = strings.TrimSpace(username)

	if username == "" {
		username = "Anonymous"
	}

	// 2. Create client object
	client := &Client{
		conn:     conn,
		username: username,
		outgoing: make(chan string, 10), // Buffered channel to hold outgoing messages up to 10
	}

	// 3. Join chatroom
	chatRoom.join <- client // Notify chat room of new client

	// send welcome message to client
	welcomeMsg := fmt.Sprintf("Welcome, %s!\n", username)
	welcomeMsg += "Commands:\n"
	welcomeMsg += "  /users - List all users\n"
	welcomeMsg += "  /msg <username> <message> - Send private message\n"
	welcomeMsg += "  /quit - Leave the chat\n"
	conn.Write([]byte(welcomeMsg))

	//4. Start reading messages from client input
	go readMessages(client, chatRoom)

	// Start receiving messages and write back to client
	writeMessages(client)

	// When writeMessages returns, it means the client has disconnected
	chatRoom.leave <- client // Notify chat room that client has left
	conn.Close()
}

// listening for input
func readMessages(client *Client, chatRoom *ChatRoom) {
	reader := bufio.NewReader(client.conn)

	for {
		message, err := reader.ReadString('\n')  // Waits for Enter key
		if err != nil {
			fmt.Printf("Client %s disconnected.\n", client.username)
			return
		}

		message = strings.TrimSpace(message)

		if message == "" {
			continue
		}

		// Check if it is a command
		if strings.HasPrefix(message, "/") {
			handleCommand(client, chatRoom, message)
			continue
		}

		// Regular message - broadcast to everyone
		formattedMessage := fmt.Sprintf("[%s]: %s\n", client.username, message)
		chatRoom.broadcast <- formattedMessage

	}
}

func handleCommand(client *Client, chatRoom *ChatRoom, command string) {
	parts := strings.Fields(command)

	if len(parts) == 0 {
		return
	}

	fmt.Println(parts)

	switch parts[0] {
	case "/users":
		chatRoom.listUsers <- client

	case "/msg":
		if len(parts) < 3 {
			select {
			case client.outgoing <- "Usage: /msg <username> <message>\n":
			default:
			}
			return
		}

		targetUsername := parts[1]
		messageText := strings.Join(parts[2:], " ")

		targetClient := chatRoom.findClientByUsername(targetUsername)

		if targetClient == nil {
			client.outgoing <- fmt.Sprintf("User '%s' is not connected\n", targetUsername)
			return
		}

		if targetClient == client {
			client.outgoing <- "You can'nt send a message to yourself!\n"
		}

		// Send to target user
		privateMsg := fmt.Sprintf("[Private from %s]: %s\n", client.username, messageText)
		chatRoom.directMessage <- DirectMessage{
			toClient: targetClient,
			message:  privateMsg,
		}

		// Send confirmation to user
		confirmation := fmt.Sprintf("Private message sent to %s\n", targetUsername)
		chatRoom.directMessage <- DirectMessage{
			toClient: client,
			message:  confirmation,
		}

	case "/quit":
		client.outgoing <- "Goodbye!\n"
		client.conn.Close()
		return

	default:
		client.outgoing <- fmt.Sprintf("Unknown command: %s", parts[0])
	}

}

// listening for output
func writeMessages(client *Client) {
	for message := range client.outgoing {
		_, err := client.conn.Write([]byte(message))
		if err != nil {
			// Can't write to client, likely disconnected
			return
		}
	}
}
