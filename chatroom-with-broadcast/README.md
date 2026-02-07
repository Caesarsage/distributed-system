
# Distributed Chatroom with Broadcast

## Overview

This project implements a distributed chatroom server and client in Go, supporting multiple users, message broadcasting, and basic persistence. It is designed as a learning tool for distributed systems, network programming, and concurrency in Go.

---

## Architecture

The system consists of two main components:

- **Server**: Listens for TCP connections, manages chatroom state, handles user join/leave, message broadcasting, and persistence.
- **Client**: Connects to the server, relays user input, and displays messages from the chatroom.

### Directory Structure

- `cmd/server/`: Entry point for the server
- `cmd/client/`: Entry point for the client
- `internal/chatroom/`: Core chatroom logic (server, client, handlers, persistence, types)
- `chatdata/`: Directory for persisted chat data (snapshots, logs)

---

## How It Works

### Server

The server listens on TCP port `9000` and manages all connected clients. It uses Go channels and goroutines to handle:

- **Client join/leave**: Tracks active users, announces arrivals/departures
- **Broadcasting**: Forwards messages from any client to all others
- **Persistence**: Periodically saves chat history and supports recovery
- **Direct messages & user listing**: (Extensible, see code)

The main server loop is in `internal/chatroom/run.go` and uses a `select` statement to multiplex events (joins, leaves, broadcasts, etc.).

### Client

The client connects to the server, starts a goroutine to listen for incoming messages, and reads user input from stdin. Messages are sent to the server, which then broadcasts them to all connected clients.

---

## Key Features

- **Multi-user chat**: Any number of clients can join and chat in real time
- **Broadcast messaging**: All messages are sent to all connected users
- **Persistence**: Chat history is periodically snapshotted and can be restored
- **Graceful join/leave**: Users are announced as they join or leave
- **Concurrency**: Uses goroutines and channels for safe, concurrent operation

---

## Code Highlights

### Server Startup

```go
// cmd/server/main.go
func main() {
	chatroom.StartServer()
}
```

### Client Startup

```go
// cmd/client/main.go
func main() {
	chatroom.StartClient()
}
```

### Chatroom Event Loop

```go
// internal/chatroom/run.go
func (cr *ChatRoom) Run() {
	for {
		select {
		case client := <-cr.join:
			cr.handleJoin(client)
		case client := <-cr.leave:
			cr.handleLeave(client)
		case message := <-cr.broadcast:
			cr.handleBroadcast(message)
		// ...
		}
	}
}
```

### Client Message Handling

```go
// internal/chatroom/client.go
go func() {
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		// Print incoming messages
	}
}()
```

---

## Running the Project

1. **Start the server:**
   ```sh
   go run ./cmd/server
   ```
   Output:
   ```
   Starting server from cmd/server...
   Server started on :9000
   ```

2. **Start one or more clients (in separate terminals):**
   ```sh
   go run ./cmd/client
   ```
   Output:
   ```
   Starting client from cmd/client...
   Connected to chat server
   Welcome to the chat server!
   >>
   ```

3. **Chat!**
   - Type messages in any client window. All connected clients will see the messages broadcast in real time.
   - When a user joins or leaves, a system message is broadcast.

---

## Example Session

**Client 1:**
```
Welcome to the chat server!
>> Hello everyone!
>>
*** user2 joined the chat ***
>>
```

**Client 2:**
```
Welcome to the chat server!
>>
*** user1 joined the chat ***
[user1]: Hello everyone!
>>
```

---

## Extending the Project

- Add authentication or usernames
- Implement private messaging
- Add chat channels/rooms
- Improve persistence (e.g., database)
- Add a web or GUI client

---

## Learning Outcomes

- Go network programming (TCP, net.Conn)
- Concurrency with goroutines and channels
- State management in distributed systems
- Basic persistence and recovery

---

## License

MIT
