package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"html"
	"log"
	"strings"
)

type wsClient struct {
	// The underlying gorilla websocket connection
	wsConnection *websocket.Conn

	// Messages intended for this client
	messagesForClient chan []byte

	// The user's id - from the database somehow
	userId int64
}

func (client *wsClient) readMessageLoop(someLobby *lobby) {
	// When our connection exits, we want to close the connection and deregister with the lobby
	// A connection exit may be e.g. a timeout
	var deregisterFunction = func() {
		someLobby.deregister <- client
		client.wsConnection.Close()
	}
	defer deregisterFunction()

	// Infinite loop
	for {
		_, msg, err := client.wsConnection.ReadMessage()
		if err != nil {
			log.Println("readMessage error:", err)
			return
		}
		incomingMessage := externalMessage{}
		err = json.Unmarshal(msg, &incomingMessage)
		if err != nil {
			log.Println("Error unmarshalling user data:", err)
			continue
		}

		// We don't allow blank messages, or messages that are just whitespace
		if strings.TrimSpace(string(incomingMessage.MessageText)) == "" {
			log.Println("Rejecting message from " + string(client.userId) + " because it contains only whitespace. ")
			continue
		}

		// Escape HTML in case the users are being nasty
		incomingMessage.MessageText = html.EscapeString(string(incomingMessage.MessageText))

		// This replaces the internal database ID with our custom ID, that is incremented
		// per room, instead of globally

		// It is unclear if there is a use of the unique global ID.
		messageId := someLobby.getNextMessageId()

		_, err = db.Exec("INSERT INTO messages(id, channel_id, message_text, channel_msg_id, author_display_name, author_id) VALUES(?, ?, ?, ?, ?, ?)", nil, someLobby.channelId, incomingMessage.MessageText, messageId, incomingMessage.MessageDisplayName, client.userId)
		if err != nil {
			log.Println("Error recording message to database", err)
		}

		// TODO: Do magic with Slack commands

		// Construct an internal struct, this case including our internal user id
		outgoingMessage := &internalMessage{MessageText: []byte(incomingMessage.MessageText), MessageAuthorId: client.userId, MessageDisplayName: []byte(incomingMessage.MessageDisplayName), MessageId: messageId}

		// Send the message out for broadcast
		someLobby.broadcast <- outgoingMessage
	}

}

func (client *wsClient) writeMessageLoop(someLobby *lobby) {
	defer client.wsConnection.Close()
	// Infinite loop
	for {
		msg := <-client.messagesForClient
		err := client.wsConnection.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Error sending message:", err)
			// Connection gets closed automatically
			return
		}
	}
}
