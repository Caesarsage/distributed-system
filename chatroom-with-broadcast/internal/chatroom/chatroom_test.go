package chatroom

import (
	"strings"
	"testing"
	"time"
)

func TestBroadcast(t *testing.T) {
	cr, _ := NewChatRoom("./testdata")
	defer cr.shutdown()

	go cr.Run()

	// Create mock clients
	client1 := &Client{
		username: "Alice",
		outgoing: make(chan string, 10),
	}
	client2 := &Client{
		username: "Bob",
		outgoing: make(chan string, 10),
	}

	// Join clients
	cr.join <- client1
	cr.join <- client2
	time.Sleep(100 * time.Millisecond)

	// Broadcast message
	cr.broadcast <- "[Alice]: Hello!"

	// Verify both receive it (ignore join/history noise)
	expectMessageContains(t, client1.outgoing, "Hello!", "Client1")
	expectMessageContains(t, client2.outgoing, "Hello!", "Client2")
}

func expectMessageContains(t *testing.T, ch <-chan string, substr, label string) {
	t.Helper()

	timeout := time.After(1 * time.Second)
	last := ""

	for {
		select {
		case msg := <-ch:
			last = msg
			if strings.Contains(msg, substr) {
				return
			}
		case <-timeout:
			if last == "" {
				t.Fatalf("%s didn't receive any message containing %q", label, substr)
			}
			t.Fatalf("%s didn't receive message containing %q; last message: %q", label, substr, last)
		}
	}
}
