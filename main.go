package main

import (
	"flag"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcaster = make(chan ChatMessage)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatMessage struct {
	Username string `json:"username"`
	Text     string `json:"text"`
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
		err := ws.ReadJSON(&msg)
		if err != nil {
			delete(clients, ws)
			break
		}
		broadcaster <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcaster
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil && unsafeError(err) {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func unsafeError(err error) bool {
	return !websocket.IsCloseError(err, websocket.CloseGoingAway) && err != io.EOF
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
