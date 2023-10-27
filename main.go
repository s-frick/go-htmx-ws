package main

import (
	"bytes"
	"flag"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcaster = make(chan []byte)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatMessage struct {
	Username string `json:"username"`
	Message  string `json:"chat_message"`
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
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
		tmpl := template.Must(template.ParseFiles("public/chat_message.html"))
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
	port := flag.String("port", "8000", "Port of webserver")
	flag.StringVar(port, "p", "8000", "Port of webserver")
	flag.Parse()

	http.Handle("/", http.FileServer(http.Dir("./public")))

	http.HandleFunc("/websocket", handleConnections)
	go handleMessages()

	log.Println("Starting Server at port " + *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
