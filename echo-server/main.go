package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// THE SERVER

func runServer() {
	//Step 1: Start listening on a port
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}

	defer listener.Close()

	fmt.Println("Server listening on port 8080")
	fmt.Println("Waiting for client connection...")

	// Step 2: Open a loop to accept client connections
	for {
		// This blocks waiting for client connections
		conn, err := listener.Accept()

		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		fmt.Println("Client connected!!")

		go handleClient(conn) // Handle client in a separate goroutine
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	// Step 3: Create a reader to read messages from the client
	reader := bufio.NewReader(conn)

	fmt.Println("Ready to receive messages...")
	messageCouner := 0

	for {
		// Step 4: Read message from client
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected from client.")
			return
		}

		message = strings.TrimSpace(message)
		messageCouner++

		// Step 5: Send echo response with conn.Write
		response := fmt.Sprintf("[%s]: ECHO: %s - %d\n", time.Now().Format("15:04:05"), message, messageCouner)

		conn.Write([]byte(response))

		fmt.Printf("Sent back \"%s\"\n", strings.TrimSpace(response))
	}
}

// THE CLIENT
func runClient() {
	// Step 1: Connect to the server
	serverCconn, err := net.Dial("tcp", "localhost:8081")

	if err != nil {
		fmt.Println("Unable to connet to server", err)
		fmt.Println("Make sure the server is running before starting the client")
		return
	}

	defer serverCconn.Close()

	fmt.Println("Connected to server")

	// Create a reader to read input from the user
	inputReader := bufio.NewReader(os.Stdin)

	// Create a reader to read responses from the server
	serverReader := bufio.NewReader(serverCconn)

	for {
		// Step 2: Get user input

		fmt.Print("You: ")
		message, _ := inputReader.ReadString('\n')
		message = strings.TrimSpace(message)

		if message == "exit" {
			fmt.Println("Exiting...")
			return
		}

		serverCconn.Write([]byte(message + "\n"))

		response, err := serverReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading response from server:", err)
			return
		}

		response = strings.TrimSpace(response)

		fmt.Printf("Server: %s\n", response)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("   go run main.go server   # to run as server")
		fmt.Println("   go run main.go client   # to run as client")
		return
	}

	mode := os.Args[1]

	switch mode {
	case "server":
		runServer()
	case "client":
		runClient()
	default:
		fmt.Println("Unknown mode:", mode)
	}
}
