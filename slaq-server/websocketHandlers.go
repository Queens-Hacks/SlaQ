package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
	"encoding/json"
)

// Set up our upgrader to websockets
// We only need to change one default (for now)
var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func arbitraryWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	urlPathSegments := strings.Split(r.URL.Path, "/")
	if len(urlPathSegments) != 4 {
		// If they are at /ws/course/coursecode/x, then we don't know what they want, so give a 404
		http.Error(w, http.StatusText(404), 404)
		return
	}
	courseCode := urlPathSegments[3]

	// Helper function that guarantees we get a non-nil lobby
	someLobby := getALobby(courseCode)

	// The third argument (nil) is to add HTTP headers... We don't need to do that
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading websocket connection:", err)
		http.Error(w, "Could not negotiate upgrade to websocket", http.StatusInternalServerError)
		return
	}

	client := &wsClient{wsConnection: conn, messagesForClient: make(chan []byte)}

	welcomeMessage := externalMessage{MessageText:"Welcome to the chat for " + courseCode, MessageDisplayName:"The Admins"}
	welcomeMessageJson, err := json.Marshal(welcomeMessage)

	if err != nil {
		log.Println("Error marshalling welcome message:", err)
	} else {
		client.wsConnection.WriteMessage(websocket.TextMessage, welcomeMessageJson)
	}

	someLobby.register <- client

	go client.writeMessageLoop(&someLobby)

	client.readMessageLoop(&someLobby)
}
