package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("   go run main.go server   # Run as server")
		fmt.Println("   go run main.go client   # Run as client")
		return
	}

	mode := os.Args[1]

	switch mode {
	case "server":
		runServer()
	case "client":
		runClient()
	default:
		fmt.Println("Invalid mode. Use 'servrer' or 'client'")
	}

}
