package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func singleMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Must only GET this endpoint", http.StatusBadRequest)
		return
	}
	paths := strings.Split(r.URL.Path, "/")

	// Make sure the URL is in the format we expect
	if len(paths) != 4 {
		http.Error(w, "Your URL is the wrong format, try /singleMessage/:courseCode/:messageId", http.StatusBadRequest)
		return
	}

	// As mentioned above, pull out the human readable course code (CISC332) and the messageId (2)
	chanMessageId := paths[3]
	courseCode := paths[2]

	channelId, err := getChannelIdFromCourseCode(courseCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Now that we have the channel_id, we combine it with the user-visible message id to get us an actual message
	rows, err := db.Query("SELECT * FROM messages WHERE channel_id = ? AND channel_msg_id = ?", channelId, chanMessageId)
	if err != nil {
		log.Println("Error loading message from database", err)
		http.Error(w, "Database error on our end!", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {

		messageToSend, err := readOneMessageFromRows(rows)
		if err != nil {
			log.Println("Error loading one message from rows", err)
			http.Error(w, "Database error on our end!", http.StatusInternalServerError)
			return
		}
		// Make sure to set our headers correctly, may as well
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		// Encode and send the message
		json.NewEncoder(w).Encode(messageToSend)

	}
}

func getSomeMessagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Must only GET this endpoint", http.StatusBadRequest)
		return
	}

	paths := strings.Split(r.URL.Path, "/")

	// Make sure the URL is in the format we expect
	if len(paths) != 4 {
		http.Error(w, "Your URL is the wrong format, try /getSomeMessages/:courseCode/:numMessages", http.StatusBadRequest)
		return
	}

	// As mentioned above, pull out the human readable course code (CISC332) and the messageId (2)
	numMessages := paths[3]
	courseCode := paths[2]

	channelId, err := getChannelIdFromCourseCode(courseCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Now that we have the channel_id, we get the messages for it
	rows, err := db.Query("SELECT * FROM messages WHERE channel_id = ? ORDER BY channel_msg_id DESC LIMIT ?", channelId, numMessages)
	if err != nil {
		log.Println("Error loading message from database", err)
		http.Error(w, "Database error on our end!", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	// Outgoing slice, so our array of external messages
	var messagesToSend []externalMessage

	for rows.Next() {
		oneMessage, err := readOneMessageFromRows(rows)
		if err != nil {
			log.Println("Error loading one message from rows", err)
			http.Error(w, "Database error on our end!", http.StatusInternalServerError)
			return
		}
		messagesToSend = append(messagesToSend, oneMessage)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(messagesToSend)

}

func getMostStarredMessages(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(r.URL.Path, "/")
	if len(paths) != 3 {
		http.Error(w, "Your URL is the wrong format, try /getMostStarred/n", http.StatusBadRequest)
		return
	}
	numWanted := paths[2]
	res, err := db.Query("select count(*) as 'numstars', messages.author_display_name, messages.message_text from stars join messages on messages.id = stars.message_id group by messages.id order by numstars desc limit ?;", numWanted)
	if err != nil {
		log.Println("error getting top stars from db", err)
		http.Error(w, "error loading stars from db", http.StatusInternalServerError)
		return
	}
	defer res.Close()

	type topStarredMsg struct {
		NumStars           int64
		MessageDisplayName string
		MessageText        string
	}

	var topStarredSlice []topStarredMsg

	for res.Next() {
		var NumStars int64
		var MessageDisplayName string
		var MessageText string
		err = res.Scan(&NumStars, &MessageDisplayName, &MessageText)
		if err != nil {
			log.Println("error scanning in message", err)
			http.Error(w, "error scanning stars from db", http.StatusInternalServerError)
			return
		}

		oneMsg := topStarredMsg{
			NumStars:           NumStars,
			MessageDisplayName: MessageDisplayName,
			MessageText:        MessageText,
		}
		topStarredSlice = append(topStarredSlice, oneMsg)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(topStarredSlice)
}
