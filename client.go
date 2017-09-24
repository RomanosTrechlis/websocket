package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

// Client is an listener of websocket
type Client struct {
	name     string
	socket   *websocket.Conn
	sendChan chan *Message
	endpoint *Endpoint
	userData map[string]interface{}
}

func (c *Client) read() {
	defer c.socket.Close()
	for {
		var msg Message
		err := c.socket.ReadJSON(&msg)
		if err != nil {
			return
		}
		msg.When = time.Now()
		msg.Author = c.userData["name"].(string)
		if avatarURL, ok := c.userData["avatar_url"]; ok {
			msg.AvatarURL = avatarURL.(string)
		}
		c.endpoint.broadcast <- &msg
	}
}

func (c *Client) write() {
	defer c.socket.Close()
	for msg := range c.sendChan {
		err := c.socket.WriteJSON(msg)
		if err != nil {
			break
		}
	}
}
