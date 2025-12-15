package chatroom

import (
	"net"
	"os"
	"sync"
	"time"
)

type Message struct {
	ID        int       `json:"id"`
	From      string    `json:"from"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Channel   string    `json:"channel"` // global or private:username
}

type Client struct {
	conn         net.Conn
	username     string
	outgoing     chan string
	lastActive   time.Time
	messagesSent int
	messagesRecv int
	isSlowClient bool // For testing

	// sessionID      string
	reconnectToken string
	mu             sync.Mutex
}

type ChatRoom struct {
	clients       map[*Client]bool
	mu            sync.Mutex
	join          chan *Client
	leave         chan *Client
	broadcast     chan string
	listUsers     chan *Client
	directMessage chan DirectMessage

	totalMessages int
	startTime     time.Time

	// Persistence fields...
	messages      []Message
	messageMu     sync.Mutex
	nextMessageID int
	walFile       *os.File
	walMu         sync.Mutex
	dataDir       string

	sessions   map[string]*SessionInfo
	sessionsMu sync.Mutex
}

type SessionInfo struct {
	Username       string
	ReconnectToken string
	LastSeen       time.Time
	CreatedAat     time.Time
}

type DirectMessage struct {
	toClient *Client
	message  string
}
