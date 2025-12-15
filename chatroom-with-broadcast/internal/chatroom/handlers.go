package chatroom

import (
	"fmt"
	"strings"
	"time"
)

// handleJoin adds a client to the active client set and broadcasts a join message.
func (cr *ChatRoom) handleJoin(client *Client) {
	cr.mu.Lock()
	cr.clients[client] = true
	cr.mu.Unlock()

	client.markActive()

	fmt.Printf("%s joined (total: %d)\n", client.username, len(cr.clients))

	cr.sendHistory(client, 10) // Lat 10 messages

	announcement := fmt.Sprintf("*** %s joined the chat ***\n", client.username)
	cr.handleBroadcast(announcement)
}

func (cr *ChatRoom) handleLeave(client *Client) {
	cr.mu.Lock()
	if !cr.clients[client] {
		cr.mu.Unlock()
		return // Already removed
	}
	delete(cr.clients, client)
	cr.mu.Unlock()

	fmt.Printf(" %s left (total: %d)\n", client.username, len(cr.clients))

	// Close channel safely
	select {
	case <-client.outgoing:
		// Already closed
	default:
		close(client.outgoing)
	}

	announcement := fmt.Sprintf("*** %s left the chat ***\n", client.username)
	cr.handleBroadcast(announcement)
}

// handleBroadcast records a chat message to persistence and forwards to all clients.
func (cr *ChatRoom) handleBroadcast(message string) {
	// Extract username from message if it's a chat message
	parts := strings.SplitN(message, ": ", 2)
	from := "system"
	actualContent := message

	if len(parts) == 2 {
		from = strings.Trim(parts[0], "[]")
		actualContent = parts[1]
	}

	// Create message
	cr.messageMu.Lock()
	msg := Message{
		ID:        cr.nextMessageID,
		From:      from,
		Content:   actualContent,
		Timestamp: time.Now(),
		Channel:   "global",
	}
	cr.nextMessageID++
	cr.messages = append(cr.messages, msg)
	cr.messageMu.Unlock()

	if err := cr.persistMessage(msg); err != nil {
		fmt.Printf("Failed to persist message: %v\n", err)
		// Still sent it (better than losing it completely)
	}

	// Now broadcast
	cr.mu.Lock()
	clients := make([]*Client, 0, len(cr.clients))
	for client := range cr.clients {
		clients = append(clients, client)
	}
	cr.totalMessages++
	cr.mu.Unlock()

	fmt.Printf(" Broadcasting to %d clients: %s", len(clients), message)

	// Send to each client (non-blocking)
	for _, client := range clients {
		select {
		case client.outgoing <- message:
			client.mu.Lock()
			client.messagesSent++
			client.mu.Unlock()
		default:
			fmt.Printf(" Skipped %s (channel full - slow client)\n", client.username)
		}
	}
}

func (cr *ChatRoom) sendHistory(client *Client, count int) {
	cr.messageMu.Lock()
	defer cr.messageMu.Unlock()

	start := len(cr.messages) - count
	if start < 0 {
		start = 0
	}

	historyMsg := "Recent messages: \n"
	for i := start; i < len(cr.messages); i++ {
		msg := cr.messages[i]
		historyMsg += fmt.Sprintf(" [%s]: %s\n", msg.From, msg.Content)
	}

	select {
	case client.outgoing <- historyMsg:
	default:
	}
}

func (cr *ChatRoom) sendUserList(client *Client) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	list := "Users online:\n"
	for c := range cr.clients {
		status := ""
		if c.isInactive(1 * time.Minute) {
			status = " (idle)"
		}
		list += fmt.Sprintf("  - %s%s\n", c.username, status)
	}

	list += fmt.Sprintf("\nTotal messages: %d\n", cr.totalMessages)
	list += fmt.Sprintf("Uptime: %s\n", time.Since(cr.startTime).Round(time.Second))

	select {
	case client.outgoing <- list:
	default:
		fmt.Printf(" Couldn't send user list to %s\n", client.username)
	}
}

func (cr *ChatRoom) handleDirectMessage(dm DirectMessage) {
	select {
	case dm.toClient.outgoing <- dm.message:
		dm.toClient.mu.Lock()
		dm.toClient.messagesSent++
		dm.toClient.mu.Unlock()
	default:
		fmt.Printf(" Couldn't deliver DM to %s\n", dm.toClient.username)
	}
}

// findClientByUsername returns the first connected client with the given username or nil.
func (cr *ChatRoom) handleHistoryCommand(client *Client, args []string) {
	count := 20 // Default
	if len(args) > 1 {
		fmt.Sscanf(args[1], "%d", &count)
	}

	if count > 100 {
		count = 100 // Limit
	}

	cr.sendHistory(client, count)
}

func (cr *ChatRoom) findClientByUsername(username string) *Client {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	for client := range cr.clients {
		if client.username == username {
			return client
		}
	}
	return nil
}

// ==

func (c *Client) markActive() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastActive = time.Now()
}

func (c *Client) isInactive(timeout time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Since(c.lastActive) > timeout
}

