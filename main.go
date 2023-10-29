package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/websocket"
)

var (
	appRoot     = os.Getenv("APP_ROOT")
	templateDir = os.Getenv("TEMPLATE_DIR")
	port        = os.Getenv("PORT")
	clients     = make(map[*websocket.Conn]bool)
	broadcaster = make(chan []byte)
	upgrader    = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type ChatMessage struct {
	Username string `json:"username"`
	Message  string `json:"chat_message"`
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("%v", err)
		tmpl := template.Must(template.ParseFiles(fmt.Sprintf("%s/err_ws.html", templateDir)))
		tmpl.Execute(w, nil)
		return

	}
	defer ws.Close()
	clients[ws] = true

	for {
		var msg ChatMessage
		var msgHtml bytes.Buffer
		err := ws.ReadJSON(&msg)
		if err != nil {
			delete(clients, ws)
			break
		}
		tmpl := template.Must(template.ParseFiles(fmt.Sprintf("%s/chat_message.html", templateDir)))
		if err := tmpl.Execute(&msgHtml, msg); err != nil {
			log.Printf("error while render template: %v", err)
			return
		}
		broadcaster <- msgHtml.Bytes()
	}
}

func handleMessages() {
	for {
		msg := <-broadcaster
		for client := range clients {
			w, err := client.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write([]byte(msg))

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

func main() {
	http.Handle("/", http.FileServer(http.Dir(templateDir)))

	http.HandleFunc("/websocket", handleConnections)
	go handleMessages()

	if port == "" {
		port = "8000"
	}
	log.Println("Starting Server at port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
