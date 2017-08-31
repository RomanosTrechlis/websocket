package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/RomanosTrechlis/websocket"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/objx"
)

var endpoints map[string]*websocket.Endpoint

// set the active Avatar implementation
var avatars Avatar = TryAvatars{
	UseFileSystemAvatar,
	UseAuthAvatar,
	UseGravatar,
}

type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
	endpoint string
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	data := map[string]interface{}{
		"Host":     r.Host,
		"Endpoint": t.endpoint,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

type List struct {
	Rooms []Element
}
type Element struct {
	Name    string
	Pattern string
}

func createEndpoint(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("endpoint")
	n, err := websocket.NewEndpoint(s, "chat/"+s, "/socket/"+s,
		MustAuth(&templateHandler{filename: "chat.html", endpoint: "socket/" + s}), nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error crteating web socket endpoint: %v", err), http.StatusBadRequest)
		return
	}
	endpoints[s] = n
	go n.Serve()
	w.Header().Set("Location", "/chat/"+s)
	http.Redirect(w, r, "/chat/"+s, http.StatusTemporaryRedirect)
}

func listRooms(w http.ResponseWriter, r *http.Request) {
	templ := template.Must(template.ParseFiles(filepath.Join("templates", "list.html")))

	elements := make([]Element, 0)
	log.Println("endpoints:", 0)
	for name, endpoint := range endpoints {
		e := Element{
			Name:    name,
			Pattern: endpoint.GetApiPattern(),
		}
		elements = append(elements, e)
		log.Println("room:", e)
	}
	list := List{
		Rooms: elements,
	}
	fmt.Println(len(list.Rooms))
	templ.Execute(w, list)
}

func main() {
	gomniauth.SetSecurityKey(googleKey)
	gomniauth.WithProviders(
		google.New(googleAppId,
			googleSecret,
			"http://localhost:8080/auth/callback/google"),
	)

	endpoints = make(map[string]*websocket.Endpoint)

	e, _ := websocket.NewEndpoint("test", "chat/test", "/socket/test",
		MustAuth(&templateHandler{filename: "chat.html", endpoint: "socket/test"}), nil)
	endpoints["test"] = e
	go e.Serve()
	e2, _ := websocket.NewEndpoint("test2", "chat/test2", "/socket/test2",
		MustAuth(&templateHandler{filename: "chat.html", endpoint: "socket/test2"}), nil)
	endpoints["test2"] = e2
	go e2.Serve()

	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/avatars/",
		http.StripPrefix("/avatars/", http.FileServer(http.Dir("./avatars"))))

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		w.Header().Set("Location", "/chat")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/list", listRooms)

	http.HandleFunc("/endpoint", createEndpoint)

	// start the web server
	log.Println("Starting web server on", ":8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
