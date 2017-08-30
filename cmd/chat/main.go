package main

import (
	"github.com/stretchr/objx"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/RomanosTrechlis/websocket"
)

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
	log.Println("What the hell")
	data := map[string]interface{}{
		"Host":     r.Host,
		"Endpoint": t.endpoint,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

func createEndpoint(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("endpoint")
	n := websocket.NewEndpoint2("chat/" + s, "/socket/" + s, MustAuth(&templateHandler{filename: "chat.html", endpoint: "socket/" + s}))
	go n.Run()
	w.Header().Set("Location", "/chat/" + s)
	http.Redirect(w, r, "/chat/" + s, http.StatusAccepted)
}

func main() {
	gomniauth.SetSecurityKey(googleKey)
	gomniauth.WithProviders(
		google.New(googleAppId,
			googleSecret,
			"http://localhost:8080/auth/callback/google"),
	)

	e := websocket.NewEndpoint2("chat/test", "/socket/test", MustAuth(&templateHandler{filename: "chat.html", endpoint: "socket/test"}))
	e2 := websocket.NewEndpoint2("chat/test2", "/socket/test2", MustAuth(&templateHandler{filename: "chat.html", endpoint: "socket/test2"}))

	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html", endpoint: "test"}))
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

	http.HandleFunc("/endpoint", createEndpoint)

	go e.Run()
	go e2.Run()
	// start the web server
	log.Println("Starting web server on", ":8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
