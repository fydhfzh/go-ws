package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type M map[string]any

type WebSocketConnection struct {
	*websocket.Conn
	Username string
}

type SocketPayload struct {
	Message string
}

type SocketResponse struct {
	From    string
	Type    string
	Message string
}

const MESSAGE_NEW_USER = "New User"
const MESSAGE_CHAT = "Chat"
const MESSAGE_LEAVE = "Leave"

var connections = make([]*WebSocketConnection, 0)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		currentGorillaConn, err := upgrader.Upgrade(w, r, w.Header())
		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusInternalServerError)
			return
		}

		username := r.URL.Query().Get("username")
		currentConn := WebSocketConnection{Conn: currentGorillaConn, Username: username}
		connections = append(connections, &currentConn)

		go handleIO(&currentConn)
	})

	fmt.Println("Server starting at :8080")
	http.ListenAndServe(":8080", nil)
}

func handleIO(currentConn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	broadcastMessage(currentConn, MESSAGE_NEW_USER, "")

	for {
		payload := SocketPayload{}
		err := currentConn.ReadJSON(&payload)
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				broadcastMessage(currentConn, MESSAGE_LEAVE, "")
				ejectConnection(currentConn)
				return
			}

			log.Println("ERROR", err.Error())
			continue
		}

		broadcastMessage(currentConn, MESSAGE_CHAT, payload.Message)
	}
}

func broadcastMessage(currentConn *WebSocketConnection, messageType string, message string) {
	for _, conn := range connections {
		if conn == currentConn {
			continue
		}

		conn.WriteJSON(SocketResponse{
			From:    currentConn.Username,
			Type:    messageType,
			Message: message,
		})
	}
}

func ejectConnection(currentConn *WebSocketConnection) {
	for i := 0; i < len(connections); i++ {
		if connections[i].Username == currentConn.Username {
			connections[i] = connections[len(connections)-1]
			connections = connections[:len(connections)-1]
			break
		}
	}
}
