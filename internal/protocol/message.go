package protocol

// message structure for all E2EE websocket messages
// server can see sender and recipient but the message content itself is encrypted
type Message struct {
	Recipient string `json:"recipient"` // not encrypted
	Sender    string `json:"sender"`    // not encrypted
	Content   []byte `json:"content"`   // encrypted
}
