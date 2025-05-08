package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Message struct {
	ID    int
	Topic string
	Text  string
}

type Client struct {
	Topic     string
	Stream    chan Message
	Timeout   <-chan time.Time
	Connected time.Time
}

type InfoCenter struct {
	mu      sync.Mutex
	clients map[string][]*Client
}

var DEBUG bool
var MSG_ID int = -1
var TIMEOUT time.Duration = 60

func main() {
	infoCenter := &InfoCenter{
		clients: make(map[string][]*Client),
	}

	// Setting up launch flags
	flag.BoolVar(&DEBUG, "debug", false, "Enable Debugging")
	url := flag.String("url", "localhost:8080", "Custom URL")
	flag.Parse()

	router := gin.Default()

	router.POST("/infocenter/:topic", func(c *gin.Context) {
		Hint("POST - start.")

		topic := c.Param("topic")
		body, err := io.ReadAll(c.Request.Body)

		if err != nil {
			Hint("! POST - error reading body.")
			c.String(http.StatusInternalServerError, "Error reading request body")
			return
		}

		message := string(body)
		infoCenter.SendMessage(topic, message)

		c.Status(http.StatusNoContent)

		Hint("POST - end.")
	})

	router.GET("/infocenter/:topic", func(c *gin.Context) {
		Hint("GET - start.")

		topic := c.Param("topic")

		// Setting headers
		c.Writer.Header().Set("Content-Type", "text/event-stream") // Dictates that SSE will be used
		c.Writer.Header().Set("Cache-Control", "no-cache")         // Turn off caching, ensuring each response will be fresh
		c.Writer.Header().Set("Connection", "keep-alive")          // Keep connection after the initial request is done

		client := infoCenter.AddClient(topic)
		defer infoCenter.RemoveClient(client)

		c.Status(http.StatusOK) // Set http status to 200 OK

		err := infoCenter.StreamMessages(c.Request.Context(), client, c.Writer)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		Hint("GET - end.")
	})

	Hint("MAIN - Running on " + *url)
	router.Run(*url)
}

// Method for sending a message to a topic
func (ic *InfoCenter) SendMessage(topic, message string) {
	Hint("SendMessage - start.")

	ic.mu.Lock()
	defer ic.mu.Unlock()

	// Incrementing message id and creating a new message
	MSG_ID++
	msg := Message{ID: MSG_ID, Topic: topic, Text: message}

	for i, client := range ic.clients[topic] {
		Hint("SendMessage - going through clients in " + topic)
		Hint("SendMessage - client " + strconv.Itoa(i))

		select {
		case client.Stream <- msg:
			Hint("SendMessage - message recieved: " + client.Topic)
		default:
			// This is hit if:
			// 1. There is no reciever
			// 2. Client channel is closed

			Hint("SendMessage - defaulting.")
		}
	}

	Hint("SendMessage - end.")
}

// Method for adding a client to a topic
func (ic *InfoCenter) AddClient(topic string) *Client {
	Hint("AddClient - start.")

	ic.mu.Lock()
	defer ic.mu.Unlock()

	// Creating a new client
	client := &Client{
		Topic:     topic,
		Stream:    make(chan Message),
		Timeout:   time.After(TIMEOUT * time.Second),
		Connected: time.Now(),
	}

	// Adding client to the map
	ic.clients[topic] = append(ic.clients[topic], client)

	Hint("AddClient - new client: topic=" + client.Topic)
	Hint("AddClient - end.")

	return client
}

// Method for removing a client from a topic
func (ic *InfoCenter) RemoveClient(client *Client) {
	Hint("RemoveClient - start.")

	ic.mu.Lock()
	defer ic.mu.Unlock()

	// Looping through all clients in target topic until the target client is found
	for i := 0; i < len(ic.clients[client.Topic]); i++ {
		if client == ic.clients[client.Topic][i] {
			slice := ic.clients[client.Topic]

			copy(slice[i:], slice[i+1:]) // shift slice[i+1:] left
			slice[len(slice)-1] = nil    // erase last element
			slice = slice[:len(slice)-1] // truncate slice

			ic.clients[client.Topic] = slice

			close(client.Stream)

			Hint("RemoveClient - client count in " + client.Topic + " = " + strconv.Itoa(len(ic.clients[client.Topic])))

			// Check if topic has no more clients
			if len(ic.clients[client.Topic]) == 0 {
				Hint("RemoveClient - removing topic " + client.Topic)
				delete(ic.clients, client.Topic)
			}

			Hint("RemoveClient - client removed.")
			Hint("RemoveClient - end.")

			return
		}
	}
}

// Method for streaming messages to clients
func (ic *InfoCenter) StreamMessages(ctx context.Context, client *Client, w io.Writer) error {
	Hint("StreamMessages - start.")

	// Initialize flusher
	f, ok := w.(http.Flusher)

	if !ok {
		Hint("StreamMessages - flusher cast failed.")

		return errors.New("flusher cast failed")
	}

	fmt.Fprint(w, "-- Client initialized --\n")
	f.Flush()

	for {
		select {
		case <-client.Timeout:
			Hint("StreamMessages - timeout (timestamp: " + time.Now().Format("15:04:05") + ")")
			fmt.Fprintf(w, "event: timeout\ndata: %s\n", time.Since(client.Connected).Round(time.Second))
			f.Flush()
			return nil

		case msg := <-client.Stream:
			Hint("StreamMessages - msg sent.")
			fmt.Fprintf(w, "id: %d\nevent: msg\ndata: %s\n", msg.ID, msg.Text)
			f.Flush()

		case <-ctx.Done():
			Hint("StreamMessages - session ended at " + time.Now().Format("15:04:05"))
			return nil
		}
	}
}

// Function for debugging in log
func Hint(hint string) {
	if DEBUG {
		fmt.Fprintf(os.Stdout, "\033[0;31m%s%s\033[0m", hint, "\n")
	}
}
