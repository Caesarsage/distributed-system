package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func runClient() {
	conn, err := net.Dial("tcp", ":9000")

	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to chat server")

	// Start reading from server (in background)
	go func() {
		reader := bufio.NewReader(conn)
		for {
			message, err := reader.ReadString('\n')  // Waits for Enter key
			if err != nil {
				fmt.Println("Disconnected from server.")
				os.Exit(0)
			}
			fmt.Print("\r" + message)
			fmt.Print(">> ")
		}
	}()

	// Read input from user and send to server
	inputReader := bufio.NewReader(os.Stdin)

	fmt.Println("Welcome to the chat server! Please enter your username: ")

	for {
		fmt.Print(">> ")
		message, _ := inputReader.ReadString('\n') // Waits for Enter key
		message = strings.TrimSpace(message)

		if message == "" {
			continue
		}

		// Write/Send message to the server
		conn.Write([]byte(message + "\n"))

	}
}

