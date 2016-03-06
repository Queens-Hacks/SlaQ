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
		var wasInDatabase = false
		rows, err := db.Query("SELECT * FROM lobbies WHERE course_code = ?", courseCode)
		if err != nil {
			log.Println("Error searching for course in database")
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var course_code string
			var last_message_id int64

			err = rows.Scan(&id, &course_code, &last_message_id)
			if err != nil {
				log.Println("Scan error", err)
				break
			}

			someLobby = lobby{
				clients:    make(map[*wsClient]bool),
				broadcast:  make(chan *internalMessage),
				register:   make(chan *wsClient),
				deregister: make(chan *wsClient),
				channelId:  id,
				// In the database, we store the previous message id
				// so we need to increment it once on boot up
				nextMessageId:    last_message_id + 1,
				nextMessageMutex: &sync.Mutex{},
			}
			wasInDatabase = true
			allLobbies[courseCode] = someLobby
		}
		if !wasInDatabase {
			// Construct a new lobby struct, since it hasn't been created yet
			res, err := db.Exec("INSERT INTO lobbies(id, course_code, last_message_id) VALUES(?, ?, ?)", nil, courseCode, 1)
			if err != nil {
				fmt.Println("Error inserting lobby into database", err)
			}
			channelId, err := res.LastInsertId()
			if err != nil {
				fmt.Println("Error getting lobby id")
			}

			someLobby = lobby{
				clients:          make(map[*wsClient]bool),
				broadcast:        make(chan *internalMessage),
				register:         make(chan *wsClient),
				deregister:       make(chan *wsClient),
				channelId:        channelId,
				nextMessageId:    1,
				nextMessageMutex: &sync.Mutex{},
			}
			allLobbies[courseCode] = someLobby
		}
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

	// Next message id for this particular channel
	nextMessageId int64

	// Mutex for accessing the nextMessageId variable
	// We need this so we can store nextMessageId in a temporary variable,
	// then increment it, then return the old value
	// We could probably use atomic primitives, e.g. something like getAndIncrement(),
	// but this will solve our problem
	nextMessageMutex *sync.Mutex
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

	// Number of favorites/stars/likes, whatever, of a particular message
	Stars int64
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
				outgoingMessage := externalMessage{MessageText: string(msg.MessageText), MessageDisplayName: string(msg.MessageDisplayName), MessageId: msg.MessageId}
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

// This keeps a running counter of the messages in each room, and concurrency safely
// gives the next number when requested
func (theLobby *lobby) getNextMessageId() int64 {
	theLobby.nextMessageMutex.Lock()
	nextId := theLobby.nextMessageId
	theLobby.nextMessageId += 1
	_, err := db.Exec("UPDATE lobbies SET last_message_id = ? WHERE id = ?", nextId, theLobby.channelId)
	if err != nil {
		log.Println("Error updating DB message id", err)
	}
	theLobby.nextMessageMutex.Unlock()
	return nextId
}
