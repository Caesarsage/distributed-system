package chatroom

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

// handleClient manages a single TCP connection: prompt for username, register
// client, start writer goroutine and process incoming lines.
func handleClient(conn net.Conn, chatRoom *ChatRoom) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in handleClient: %v\n", r)
		}
		conn.Close()
	}()

	// Set initial read timeout for username
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	reader := bufio.NewReader(conn)

	// Ask for username or reconnect token
	conn.Write([]byte("Enter username (or 'reconnect:<username>:<token>' to reconnect): \n"))

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read username:", err)
		return
	}
	input = strings.TrimSpace(input)

	var username string
	var reconnectToken string
	var isReconnecting bool

	if strings.HasPrefix(input, "reconnect:") {
		parts := strings.Split(input, ":")
		if len(parts) == 3 {
			username = parts[1]
			reconnectToken = parts[2]
			isReconnecting = true
		} else {
			conn.Write([]byte("Invalid reconnect format. Use: reconnect:<username>:<token> \n"))
			return
		}
	} else {
		username = input
	}

	if username == "" {
		username = fmt.Sprintf("Guest%d", rand.Intn(1000))
	}

	if isReconnecting {
		if chatRoom.validateReconnectToken(username, reconnectToken) {
			fmt.Printf("%s reconnected successfully\n", username)
			conn.Write([]byte(fmt.Sprintf("Welcome back, %s!\n", username)))
		} else {
			conn.Write([]byte("Invalid reconnect token or session expired. \n"))
		}
	} else {
		// New connection - check it username is already connected
		if chatRoom.isUsernameConnected(username) {
			conn.Write([]byte("Username already connected, Use reconnect if you lost connection\n"))
			return
		}

		// check if session exists (was connected before)
		chatRoom.sessionsMu.Lock()
		existingSession := chatRoom.sessions[username]
		chatRoom.sessionsMu.Unlock()

		if existingSession != nil {
			// They were here before, give them their token
			token := existingSession.ReconnectToken
			msg := fmt.Sprintf("Tip: Save this reconnect token: %s\n", token)
			msg += fmt.Sprintf("   To reconnect later: reconnect:%s:%s\n", username, token)
			conn.Write([]byte(msg))
		} else {
			// Brand new user, create session
			session := chatRoom.createSession(username)
			token := session.ReconnectToken
			msg := fmt.Sprintf("Your reconnect token: %s\n", token)
			msg += fmt.Sprintf("   Save this to reconnect: reconnect:%s:%s\n", username, token)
			conn.Write([]byte(msg))
		}
	}

	// Create client
	client := &Client{
		conn:       conn,
		username:   username,
		outgoing:   make(chan string, 10),
		lastActive: time.Now(),
		reconnectToken: reconnectToken,
		// TEST MODE: Simulate slow client randomly
		isSlowClient: rand.Float64() < 0.1, // 10% chance
	}

	if client.isSlowClient {
		fmt.Printf("%s is a SLOW CLIENT (testing mode)\n", username)
	}

	// Clear read deadline for normal operation
	conn.SetReadDeadline(time.Time{})

	chatRoom.join <- client

	welcomeMsg := fmt.Sprintf("Welcome, %s!\n", username)
	welcomeMsg += "Commands:\n"
	welcomeMsg += "  /users - List all users\n"
	welcomeMsg += "  /history [N] - Show last N messages\n"
	welcomeMsg += "  /msg <user> <msg> - Private message\n"
	welcomeMsg += "  /token - Show your reconnect token\n"
	welcomeMsg += "  /stats - Show your stats\n"
	welcomeMsg += "  /simulate crash - Test crash handling\n"
	welcomeMsg += "  /quit - Leave\n"
	conn.Write([]byte(welcomeMsg))

	go readMessages(client, chatRoom)

	writeMessages(client)

	// Client disconnected - update session but don't delete
	chatRoom.updateSessionActivity(username)
	chatRoom.leave <- client
}

func readMessages(client *Client, chatRoom *ChatRoom) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in readMessages for %s: %v\n", client.username, r)
		}
	}()

	reader := bufio.NewReader(client.conn)

	for {
		// Set read timeout
		client.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		message, err := reader.ReadString('\n')
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Printf("%s timed out\n", client.username)
			} else {
				fmt.Printf("%s disconnected: %v\n", client.username, err)
			}
			return
		}

		client.markActive()

		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}

		client.mu.Lock()
		client.messagesRecv++
		client.mu.Unlock()

		// Process command
		if strings.HasPrefix(message, "/") {
			handleCommand(client, chatRoom, message)
			continue
		}

		// Broadcast message
		formatted := fmt.Sprintf("[%s]: %s\n", client.username, message)
		chatRoom.broadcast <- formatted
	}
}

// handleCommand parses and executes a simple client command. Returns true if
// the input was a command and was handled, false if it should be treated as
// a normal message.
func handleCommand(client *Client, chatRoom *ChatRoom, command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	// TODO: Breakdown and Move to hamdlers.go
	switch parts[0] {
	case "/users":
		chatRoom.listUsers <- client

	case "/stats":
		client.mu.Lock()
		stats := "Your Stats:\n"
		stats += fmt.Sprintf("  Messages sent: %d\n", client.messagesSent)
		stats += fmt.Sprintf("  Messages received: %d\n", client.messagesRecv)
		stats += fmt.Sprintf("  Last active: %s ago\n", time.Since(client.lastActive).Round(time.Second))
		if client.isSlowClient {
			stats += "  You are a SLOW CLIENT (test mode)\n"
		}
		client.mu.Unlock()

		select {
		case client.outgoing <- stats:
		default:
		}

	case "/simulate":
		if len(parts) > 1 && parts[1] == "crash" {
			client.outgoing <- "Simulating crash...\n"
			time.Sleep(100 * time.Millisecond)
			client.conn.Close() // Abrupt disconnect!
			return
		}

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
			select {
			case client.outgoing <- fmt.Sprintf("User '%s' not found\n", targetUsername):
			default:
			}
			return
		}

		if targetClient == client {
			select {
			case client.outgoing <- "Can't message yourself!\n":
			default:
			}
			return
		}

		privateMsg := fmt.Sprintf("[From %s]: %s\n", client.username, messageText)
		select {
		case targetClient.outgoing <- privateMsg:
		default:
			select {
			case client.outgoing <- fmt.Sprintf("%s's inbox is full\n", targetUsername):
			default:
			}
			return
		}

		select {
		case client.outgoing <- fmt.Sprintf("Message sent to %s\n", targetUsername):
		default:
		}

	case "/history":
		chatRoom.handleHistoryCommand(client, parts)

	case "/token":
		chatRoom.sessionsMu.Lock()
		session := chatRoom.sessions[client.username]
		chatRoom.sessionsMu.Unlock()

		if session != nil {
			msg := "Your reconnect token:\n"
			msg += fmt.Sprintf("   reconnect:%s:%s\n", client.username, session.ReconnectToken)
			msg += "   Use this to reconnect if you disconnect.\n"
			select {
			case client.outgoing <- msg:
			default:
			}
		} else {
			select {
			case client.outgoing <- " No session found\n":
			default:
			}
		}

	case "/quit":
		announcement := fmt.Sprintf("%s left the chat\n", client.username)
		chatRoom.broadcast <- announcement

		select {
		case client.outgoing <- "Goodbye!\n":
		default:
		}

		time.Sleep(100 * time.Millisecond)
		client.conn.Close()

	default:
		select {
		case client.outgoing <- fmt.Sprintf("Unknown: %s\n", parts[0]):
		default:
		}
	}
}

// clientWriteLoop forwards messages from client.outgoing into the TCP connection.
func writeMessages(client *Client) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in writeMessages for %s: %v\n", client.username, r)
		}
	}()

	writer := bufio.NewWriter(client.conn)

	for message := range client.outgoing {
		// Simulate slow client
		if client.isSlowClient {
			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
		}

		_, err := writer.WriteString(message)
		if err != nil {
			fmt.Printf("⚠️  Write error for %s: %v\n", client.username, err)
			return
		}

		err = writer.Flush()
		if err != nil {
			fmt.Printf("⚠️  Flush error for %s: %v\n", client.username, err)
			return
		}
	}
}

