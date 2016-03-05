package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// Maps are not concurrency safe, so we need a mutex to manage access to the map
var allLobbiesMutex = &sync.Mutex{}

// Map from every course code to a lobby
var allLobbies = make(map[string]lobby)

func getALobby(courseCode string) lobby {
	allLobbiesMutex.Lock()
	// See if the allLobbies map contains our desired lobby
	someLobby, ok := allLobbies[courseCode]
	if !ok {
		// Construct a new lobby struct, since it hasn't been created yet
		res, err := db.Exec("INSERT INTO lobbies(id, course_code) VALUES(?, ?)", nil, courseCode)
		if err != nil {
			fmt.Println("Error inserting lobby into database", err)
		}
		channelId, err := res.LastInsertId()
		if err != nil {
			fmt.Println("Error getting lobby id")
		}

		someLobby = lobby{
			clients:    make(map[*wsClient]bool),
			broadcast:  make(chan *internalMessage),
			register:   make(chan *wsClient),
			deregister: make(chan *wsClient),
			channelId:  channelId,
		}
		allLobbies[courseCode] = someLobby
	}
	go someLobby.serveLobby()
	allLobbiesMutex.Unlock()
	return someLobby
}

type lobby struct {
	// A list of our clients, to which we broadcast messages
	// Boolean because we need something, but we don't actually care
	clients map[*wsClient]bool

	// The channel on which we receive broadcast messages (from a particular client)
	broadcast chan *internalMessage

	// The channel on which we receive register requests (from a new websocket connection)
	register chan *wsClient

	// The channel on which we receive deregister requests (from a websocket CLOSE message)
	deregister chan *wsClient

	// Number that is uniquely assigned to this channel
	channelId int64
}

type internalMessage struct {
	// Actual contents of the message
	MessageText []byte

	// The display name set by the user
	MessageDisplayName []byte

	// The internal author id for our use
	MessageAuthorId int64

	// Message id assigned by the database, but this is also sent to the frontend
	MessageId int64
}

type externalMessage struct {
	// Actual contents of the message
	MessageText string

	// The display name set by the user
	MessageDisplayName string

	// Messaged id assigned by the database, used by the front-end to intelligently re-order messages
	// and potentially re-request missing ones
	MessageId int64
}

func (theLobby *lobby) serveLobby() {
	for {
		// Use this magic select statement syntax, where instead of blocking
		// on a channel, it only chooses the one which is ready!
		select {
		case someClient := <-theLobby.register:
			theLobby.clients[someClient] = true

		case someClient := <-theLobby.deregister:
			// Remove the client from the map
			delete(theLobby.clients, someClient)
			// Close the messages channel to prevent a resource leak
			close(someClient.messagesForClient)

		case msg := <-theLobby.broadcast:
			for someClient := range theLobby.clients {
				// TODO: Check the author and do magic (replace name with "You")
				outgoingMessage := externalMessage{MessageText: string(msg.MessageText), MessageDisplayName: string(msg.MessageDisplayName)}
				stringifiedJson, err := json.Marshal(outgoingMessage)
				if err != nil {
					log.Println("Error marshalling outgoing message", err)
					// Break because we don't want to try the same bogus message with all the users - just exit now
					break
				}
				select {
				case someClient.messagesForClient <- stringifiedJson:
					// The above command sent the message, so we're done!
				default:
					// This is an error condition, let's remove this client
					theLobby.deregister <- someClient
				}

			}
		}

	}
}
