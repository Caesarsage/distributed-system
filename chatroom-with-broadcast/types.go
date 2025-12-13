package main

import (
	"net"
	"sync"
)

// Client represents a connected chat user
type Client struct {
	conn     net.Conn
	username string
	outgoing chan string // Tube for messages to THIS client
}

// ChatRoom manages all connected clients
type ChatRoom struct {
	clients       map[*Client]bool // map of connected clients
	mu            sync.Mutex       // protects the clients map
	join          chan *Client
	leave         chan *Client
	broadcast     chan string  // Tube for messages to everyone
	listUsers     chan *Client // Tube for "show me users" requests
	directMessage chan DirectMessage
}

type DirectMessage struct {
	toClient *Client
	message  string
}
