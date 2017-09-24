package websocket

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
)

// Message is what client and endpoint exchange
type Message struct {
	Author    string    `json:"Author"`
	Content   string    `json:"Content"`
	When      time.Time `json:"Timestamp"`
	AvatarURL string    `json:"AvatarURL"`
}

// Serialize takes the a message and returns bytes
func (m *Message) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		return nil, fmt.Errorf("serialization failed: %v", err)
	}
	return buf.Bytes(), nil
}

// Deserialize takes bytes and returns a message
func (m *Message) Deserialize(enc []byte) (Message, error) {
	var buf bytes.Buffer
	buf.Write(enc)
	dec := gob.NewDecoder(&buf)
	err := dec.Decode(m)
	if err != nil {
		return *m, fmt.Errorf("deserialization failed: %v", err)
	}
	return *m, nil
}
