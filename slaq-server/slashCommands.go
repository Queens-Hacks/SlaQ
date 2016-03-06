package main

import (
	"encoding/json"
	"github.com/mvdan/xurls"
	"github.com/paddycarey/gophy"
	"log"
	"net/url"
	"strconv"
)

func (theLobby *lobby) sendGiphy(searchTerm string, authorName string, userId int64, messageId int64) {
	searchTerm = url.QueryEscape(searchTerm)
	gophyOptions := &gophy.ClientOptions{}
	gophyClient := gophy.NewClient(gophyOptions)

	// Search for the particular gif with the parameters
	// "" -> rating (.e.g PG, PG-13, etc)... We want it all
	// 1 is the number of entries, we just want one
	// 0 is the offet, how many pages
	gifs, num, err := gophyClient.SearchGifs(searchTerm, "", 1, 0)
	if err != nil {
		log.Println("error searching giphy: ", err)
		return
	}
	if num > 0 {
		imageUrl := gifs[0].Images.FixedWidth.URL
		giphyMessage := `<img src="` + imageUrl + `" alt="` + searchTerm + `" title="` + searchTerm + `" class="gif" >`
		theLobby.broadcast <- &internalMessage{
			MessageText:        []byte(giphyMessage),
			MessageDisplayName: []byte(authorName),
			MessageAuthorId:    userId,
			MessageId:          messageId,
		}
	}
}

func (theLobby *lobby) sendStar(messageToStar string, starrerId int64, starringMessageId int64) {
	// Find the real message id, versus the channel message id... It gets confusing
	rows, err := db.Query("SELECT * FROM messages WHERE channel_msg_id = ? AND channel_id = ?", messageToStar, theLobby.channelId)
	if err != nil {
		log.Println("Error finding messages for sendStar: ", err)
		return
	}

	// Read in the singular message in this scope
	var message_to_star_real_id int64
	var channel_id int64
	var message_text string
	var channel_msg_id int64
	var author_display_name string
	var author_id int64

	foundOne := false
	for rows.Next() {
		foundOne = true
		err = rows.Scan(&message_to_star_real_id, &channel_id, &message_text, &channel_msg_id, &author_display_name, &author_id)
		if err != nil {
			log.Println("error scanning row", err)
			return
		}
		break
	}
	rows.Close()

	if !foundOne {
		// We don't actually have a message... Let's back out
		return
	}

	// now that we have the real id, let's make sure this is this particular user's first star
	rows, err = db.Query("select * from stars where starrer_id = ? and message_id = ?;", starrerId, message_to_star_real_id)
	if err != nil {
		log.Println("error searching for existing star")
		return
	}

	for rows.Next() {
		log.Println("user prevented from double starring: ", starrerId)
		rows.Close()
		return
	}
	rows.Close()

	// Now that we have the true message id, insert our star
	_, err = db.Exec("INSERT INTO stars(id, starrer_id, starree_id, message_id) VALUES (?, ?, ?, ?);", nil, starrerId, author_id, message_to_star_real_id)
	if err != nil {
		log.Println("error writing star to database", err)
		return
	}

	// Now, get all the stars
	stars, err := db.Query("SELECT * FROM stars WHERE message_id = ?;", message_to_star_real_id)
	if err != nil {
		log.Println("error loading stars", err)
		return
	}
	defer stars.Close()
	var starCounter = 0

	// Count the stars
	for stars.Next() {
		var star_id, starrer_id, starree_id, message_id int64
		stars.Scan(&star_id, &starrer_id, &starree_id, &message_id)
		starCounter += 1
	}

	// Special message to send to the front end ...
	type StarMessage struct {
		MessageId int64
		NumStars  int
	}

	i, err := strconv.ParseInt(messageToStar, 10, 64)
	if err != nil {
		log.Println("couldnt parse string into an integer", err)
		return
	}

	outgoingMessage := StarMessage{
		MessageId: i,
		NumStars:  starCounter,
	}

	marshalled, err := json.Marshal(outgoingMessage)
	if err != nil {
		log.Println("error marshalling star message", err)
		return
	}

	// ... from a magic sender
	theLobby.broadcast <- &internalMessage{
		MessageText:        marshalled,
		MessageDisplayName: []byte("__ADMIN__"),
		MessageAuthorId:    0,
		MessageId:          starringMessageId,
	}

}

func (theLobby *lobby) linkifyMessage(messageString string, messageAuthorId int64, messageDisplayName string, messageId int64) {

	linkifiedMessage := xurls.Relaxed.ReplaceAllStringFunc(messageString, func(inString string) string {
		url, err := url.Parse(inString)
		var scheme string
		if err == nil {
			scheme = url.Scheme
		} else {
			log.Println("error parsing url: ", err)
		}
		if scheme == "" {
			url.Scheme = "http"
		}
		return `<a href="` + url.String() + `">` + inString + `</a>`
	})

	// Construct an internal struct, this case including our internal user id
	outgoingMessage := &internalMessage{MessageText: []byte(linkifiedMessage), MessageAuthorId: messageAuthorId, MessageDisplayName: []byte(messageDisplayName), MessageId: messageId}

	// Send the message out for broadcast
	theLobby.broadcast <- outgoingMessage

}

func (theLobby *lobby) sendCoolFace(messageDisplayName string, messageAuthorId int64, messageId int64) {
	coolFaceMessage := `<span class="coolface">` + coolFaces[rand.Intn(len(coolFaces))] + `</span>`

	// Construct an internal struct, this case including our internal user id
	outgoingMessage := &internalMessage{MessageText: []byte(coolFaceMessage), MessageAuthorId: messageAuthorId, MessageDisplayName: []byte(messageDisplayName), MessageId: messageId}

	// Send the message out for broadcast
	theLobby.broadcast <- outgoingMessage
}
