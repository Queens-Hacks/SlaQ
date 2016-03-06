package main

import (
	"encoding/json"
	"log"
)

func (theLobby *lobby) sendStar(messageToStar string, starrerId int64, starringMessageId int64) {
	rows, err := db.Query("SELECT * FROM messages WHERE channel_msg_id = ? AND channel_id = ?", messageToStar, theLobby.channelId)
	if err != nil {
		log.Println("Error finding messages for sendStar: ", err)
		return
	}

	var message_to_star_real_id int64
	var channel_id int64
	var message_text string
	var channel_msg_id int64
	var author_display_name string
	var author_id int64
	for rows.Next() {
		err = rows.Scan(&message_to_star_real_id, &channel_id, &message_text, &channel_msg_id, &author_display_name, &author_id)
		if err != nil {
			log.Println("error scanning row", err)
			return
		}
		break
	}
	rows.Close()

	_, err = db.Exec("INSERT INTO stars(id, starrer_id, starree_id, message_id) VALUES (?, ?, ?, ?);", nil, starrerId, author_id, message_to_star_real_id)
	if err != nil {
		log.Println("error writing star to database", err)
		return
	}

	stars, err := db.Query("SELECT * FROM stars WHERE message_id = ?;", message_to_star_real_id)
	if err != nil {
		log.Println("error loading stars", err)
		return
	}
	defer stars.Close()
	var starCounter = 0
	for stars.Next() {
		var star_id, starrer_id, starree_id, message_id int64
		stars.Scan(&star_id, &starrer_id, &starree_id, &message_id)
		starCounter += 1
	}

	type StarMessage struct {
		MessageId int64
		NumStars  int
	}

	outgoingMessage := StarMessage{
		MessageId: starringMessageId,
		NumStars:  starCounter,
	}

	marshalled, err := json.Marshal(outgoingMessage)
	if err != nil {
		log.Println("error marshalling star message", err)
		return
	}

	theLobby.broadcast <- &internalMessage{
		MessageText:        marshalled,
		MessageDisplayName: []byte("__ADMIN__"),
		MessageAuthorId:    0,
		MessageId:          starringMessageId,
	}

}
