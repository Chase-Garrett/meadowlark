package server

import "meadowlark/internal/protocol"

// hub maintains the active clients and forwards messages
type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	forward    chan *protocol.Message
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		forward:    make(chan *protocol.Message),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.username] = client
		case client := <-h.unregister:
			if _, ok := h.clients[client.username]; ok {
				delete(h.clients, client.username)
				close(cleint.send)
			}
		case message := <-h.forward:
			// find recipient client and send the message
			if recipient, ok := h.clients[message.Recipient]; ok {
				select {
				case recipient.send <- message:
				default:
					close(recipient.send)
					delete(h.clients, recipient.username)
				}
			}
		}
	}
}
