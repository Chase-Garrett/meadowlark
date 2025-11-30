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

		var msg protocol.Message
		if err := json.Unmarshal(messageBytes, &msg); err == nil {
			msg.Sender = c.username // ensure correctly identified sender
			c.hub.forward <- &msg
		}
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
			if err == nil {
				c.conn.WriteMessage(websocket.TextMessage, messageBytes)
			}
		}
	}
}
