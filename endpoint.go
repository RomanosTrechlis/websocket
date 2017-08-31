package websocket

import (
	"fmt"
	"log"
	"net/http"

	"github.com/RomanosTrechlis/golog"
	"github.com/gorilla/websocket"
	"github.com/stretchr/objx"
	"os"
)

// Endpoint
type Endpoint struct {
	// name is the identifier of the Endpoint
	name string
	// pattern is the http(s) route
	pattern string
	// handler is the http.Handler that handles the
	handler http.Handler
	// wsPattern is the route on which websocket of Endpoint listens
	wsPattern string

	// logger for logging actions
	logger *golog.LogWrapper

	// clients map holds a list of all connected clients
	clients map[*Client]bool
	// broadcast is the channel that sends messages to clients
	broadcast chan *Message

	// register is a channel that allows clients to join Endpoint
	register chan *Client
	// unregister is a channel that allows clients to leave Endpoint
	unregister chan *Client
}

func init() {
	endpointsByName = make(map[string]bool)
	endpointsByWsPattern = make(map[string]bool)
	endpointsByPattern = make(map[string]bool)
}

func NewEndpoint(name, pattern, wsPattern string, handler http.Handler, logger *golog.LogWrapper) (*Endpoint, error) {
	if checkEndpointExists(name) {
		return nil, fmt.Errorf("endpoint with name '%s' already exists", name)
	}
	if checkEndpointWsPatternExists(wsPattern) {
		return nil, fmt.Errorf("endpoint with web socket pattern '%s' already exists", wsPattern)
	}
	if pattern != "" && checkEndpointPatternExists(pattern) {
		return nil, fmt.Errorf("endpoint with api pattern '%s' already exists", pattern)
	}
	// build logger in case there is none
	if logger == nil {
		logger = golog.New()
		logger.New(os.Stdout, golog.TRACE, 0)
	}
	endpoint := &Endpoint{
		name:       name,
		wsPattern:  wsPattern,
		pattern:    pattern,
		handler:    handler,
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		logger:     logger,
	}
	endpointsByName[name] = true
	endpointsByWsPattern[wsPattern] = true
	endpointsByPattern[pattern] = true
	return endpoint, nil
}

func (e *Endpoint) GetApiPattern() string {
	return e.pattern
}

func (e *Endpoint) Serve() {
	// register the endpoints
	if e.pattern != "" && e.handler != nil {
		e.logger.Trace("creating api route '/%s'", e.pattern)
		http.Handle(fmt.Sprintf("/%s", e.pattern), e.handler)
	}

	e.logger.Trace("creating web socket endpoint '%s'", e.wsPattern)
	http.Handle(e.wsPattern, e)
	for {
		select {
		case client := <-e.register:
			e.clients[client] = true
			e.logger.Info("New client registered")
		case client := <-e.unregister:
			delete(e.clients, client)
			close(client.sendChan)
			e.logger.Info("Client unregistered")
		case msg := <-e.broadcast:
			e.logger.Trace("Message received: ", msg)
			for client := range e.clients {
				client.sendChan <- msg
				e.logger.Trace("-- sent to client")
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
		sendChan: make(chan *Message, 1024),
		endpoint: e,
		userData: objx.MustFromBase64(authCookie.Value),
	}
	e.register <- client
	defer func() { e.unregister <- client }()
	go client.write()
	client.read()
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var endpointsByName map[string]bool
var endpointsByWsPattern map[string]bool
var endpointsByPattern map[string]bool

func checkEndpointExists(name string) bool {
	_, ok := endpointsByName[name]
	return ok
}

func checkEndpointWsPatternExists(wsPattern string) bool {
	_, ok := endpointsByWsPattern[wsPattern]
	return ok
}

func checkEndpointPatternExists(pattern string) bool {
	_, ok := endpointsByPattern[pattern]
	return ok
}
