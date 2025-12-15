package chatroom

// This file was split into multiple files (persistence.go, run.go, handlers.go,
// session.go, io.go). The old monolithic implementation was left here for a
// short transitional period. Keep a small wrapper for compatibility.

// StartServer starts the chat server (implemented in run.go).
func StartServer() {
	runServer()
}
