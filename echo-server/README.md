# Echo Server

Get a server and client talking to each other


## The server



## The client


## How to use

1. Start the server (terminal 1)

```go
  go run main.go server
```

You should see:
```
Server started on port 8081
Waiting for connections...
```

2. Start the client(s) (Accept multiple concurrent clients)

```go
  go run main.go client
```

You should see:
```
Connected to server!
Type messages and press Enter (type 'exit' to exit)
---

You:
```

3. Send a message!

```
You: hello world

Server: [<timestamp>]: ECHO: hello world - <counter>
```

## Let Me Explain the Key Parts

### The Server Part
```go
go listener, err := net.Listen("tcp", ":8080")
```

**Translation:** "Hey computer, I want to listen for connections on port 8081"

```go
go conn, err := listener.Accept()
```

**Translation:** "Wait here until someone connects" (this blocks/waits)

```go

go handleClient(conn)
```

**Translation:** "Handle this client in the background, so I can accept more clients"

The `go` keyword is CRUCIAL - it's what makes it concurrent!

### The Client Part

```go
go conn, err := net.Dial("tcp", "localhost:8080")
```

**Translation:** "Connect me to the server at localhost:8080"
