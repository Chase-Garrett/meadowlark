package server

import (
	"encoding/json"
	"log"

	"github.com/Chase-Garrett/meadowlark/internal/protocol"

	"github.com/gorilla/websocket"
)

// middleware between websocket connection and hub
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan *protocol.Message
	username string
}

// IncomingMessage represents a message received from the client
type IncomingMessage struct {
	Recipient string      `json:"recipient"`
	Sender    string      `json:"sender"`
	Content   interface{} `json:"content"` // Can be string or base64 string
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var incoming IncomingMessage
		if err := json.Unmarshal(messageBytes, &incoming); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Convert content to []byte
		// Frontend sends content as a string, we convert to []byte
		var contentBytes []byte
		if contentStr, ok := incoming.Content.(string); ok {
			contentBytes = []byte(contentStr)
		} else {
			// Fallback: try to unmarshal as protocol.Message for base64 []byte support
			var msg protocol.Message
			if json.Unmarshal(messageBytes, &msg) == nil {
				contentBytes = msg.Content
			} else {
				log.Printf("Could not parse content, expected string, got: %T", incoming.Content)
				continue
			}
		}

		msg := &protocol.Message{
			Recipient: incoming.Recipient,
			Sender:    c.username, // ensure correctly identified sender
			Content:   contentBytes,
		}

		c.hub.forward <- msg
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshaling message: %v", err)
				continue
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}
		}
	}
}
