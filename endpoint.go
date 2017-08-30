package websocket

import (
	"fmt"
	"github.com/stretchr/objx"
	"os"
	"net/http"
	"github.com/RomanosTrechlis/GoProgBlueprints/trace"
	"github.com/gorilla/websocket"
	"log"
)

type Endpoint struct {
	id      string
	pattern string
	handler http.Handler

	tracer trace.Tracer

	clients   map[*Client]bool
	broadcast chan *Message

	register   chan *Client
	unregister chan *Client
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewEndpoint(name, pattern string) *Endpoint {
	return &Endpoint{
		id:         name,
		pattern:    pattern,
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		tracer:     trace.New(os.Stdout),
	}
}

func NewEndpoint2(name, pattern string, handler http.Handler) *Endpoint {
	return &Endpoint{
		id:         name,
		pattern:    pattern,
		handler:    handler,
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		tracer:     trace.New(os.Stdout),
	}
}

func (e *Endpoint) Run() {
	e.tracer.Trace(fmt.Sprintf("/%s", e.id))
	http.Handle(fmt.Sprintf("/%s", e.id), e.handler)
	http.Handle(e.pattern, e)
	for {
		select {
		case client := <-e.register:
			e.clients[client] = true
			e.tracer.Trace("New client registered")
		case client := <-e.unregister:
			delete(e.clients, client)
			close(client.send)
			e.tracer.Trace("Client unregistered")
		case msg := <-e.broadcast:

			e.tracer.Trace("Message received: ", msg)
			for client := range e.clients {
				client.send <- msg
				e.tracer.Trace("-- sent to client")
			}
		}
	}
}

func (e *Endpoint) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}
	authCookie, err := req.Cookie("auth")
	if err != nil {
		log.Fatal("Failed to get auth cookie:", err)
		return
	}
	client := &Client{
		socket:   socket,
		send:     make(chan *Message, 1024),
		endpoint: e,
		userData: objx.MustFromBase64(authCookie.Value),
	}
	e.register <- client
	defer func() { e.unregister <- client }()
	go client.write()
	client.read()
}

