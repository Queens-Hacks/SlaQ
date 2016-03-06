package main

import (
	"encoding/json"
	"github.com/mvdan/xurls"
	"github.com/paddycarey/gophy"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
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

func (theLobby *lobby) sendIsTimsOpen() {
	currentTime := time.Now()

	var locations []string

	/* *** JDUC *** */

	switch currentTime.Weekday() {
	case time.Monday:
		fallthrough
	case time.Tuesday:
		fallthrough
	case time.Wednesday:
		fallthrough
	case time.Thursday:
		fallthrough
	case time.Friday:
		if currentTime.Hour() == 7 && currentTime.Minute() >= 30 {
			locations = append(locations, "JDUC (Until 3pm)")
		} else if currentTime.Hour() == 7 && currentTime.Minute() < 30 {

		} else if currentTime.Hour() > 7 && currentTime.Hour() < 15 {
			locations = append(locations, "JDUC (Until 3pm)")
		} else if currentTime.Hour() == 15 && currentTime.Minute() == 0 {
			locations = append(locations, "JDUC (Until 3pm)")
		}
	case time.Saturday:
		// Do nothing
	case time.Sunday:
		// Do nothing
	}

	/* *** Queen's Centre *** */

	switch currentTime.Weekday() {
	case time.Monday:
		fallthrough
	case time.Tuesday:
		fallthrough
	case time.Wednesday:
		fallthrough
	case time.Thursday:
		fallthrough
	case time.Friday:
		if currentTime.Hour() >= 8 && currentTime.Hour() < 23 {
			locations = append(locations, "Queen's Centre (until 11pm)")
		} else if currentTime.Hour() == 23 && currentTime.Minute() == 0 {
			locations = append(locations, "Queen's Centre (until 11pm)")
		}
	case time.Saturday:
		fallthrough
	case time.Sunday:
		if currentTime.Hour() >= 8 && currentTime.Hour() < 19 {
			locations = append(locations, "Queen's Centre (until 7pm)")
		} else if currentTime.Hour() == 19 && currentTime.Minute() == 0 {
			locations = append(locations, "Queen's Centre (until 7pm)")
		}
	}

	/* *** Self-Serve BioSci *** */

	switch currentTime.Weekday() {
	case time.Monday:
		fallthrough
	case time.Tuesday:
		fallthrough
	case time.Wednesday:
		fallthrough
	case time.Thursday:
		if currentTime.Hour() == 8 && currentTime.Minute() >= 30 {
			locations = append(locations, "Self-Serve @ BioSci (until 3:30pm)")
		} else if currentTime.Hour() == 8 && currentTime.Minute() < 30 {

		} else if currentTime.Hour() > 8 && currentTime.Hour() < 15 {
			locations = append(locations, "Self-Serve @ BioSci (until 3:30pm)")
		} else if currentTime.Hour() == 15 && currentTime.Minute() <= 30 {
			locations = append(locations, "Self-Serve @ BioSci (until 3:30pm)")
		}
	case time.Friday:
		if currentTime.Hour() == 8 && currentTime.Minute() >= 30 {
			locations = append(locations, "Self-Serve @ BioSci (until 1:30pm)")
		} else if currentTime.Hour() == 8 && currentTime.Minute() < 30 {

		} else if currentTime.Hour() > 8 && currentTime.Hour() < 13 {
			locations = append(locations, "Self-Serve @ BioSci (until 1:30pm)")
		} else if currentTime.Hour() == 13 && currentTime.Minute() <= 30 {
			locations = append(locations, "Self-Serve @ BioSci (until 1:30pm)")
		}
	}

	/* *** BioSci *** */
	switch currentTime.Weekday() {
	case time.Monday:
		fallthrough
	case time.Tuesday:
		fallthrough
	case time.Wednesday:
		fallthrough
	case time.Thursday:
		if currentTime.Hour() >= 7 && currentTime.Hour() < 18 {
			locations = append(locations, "BioSci (until 6pm)")
		} else if currentTime.Hour() == 18 && currentTime.Minute() == 0 {
			locations = append(locations, "BioSci (until 6pm)")
		}
	case time.Friday:
		if currentTime.Hour() >= 7 && currentTime.Hour() < 15 {
			locations = append(locations, "BioSci (until 3:30pm)")
		} else if currentTime.Hour() == 15 && currentTime.Minute() <= 30 {
			locations = append(locations, "BioSci (until 3:30pm)")
		}
	}

	messageId := theLobby.getNextMessageId()

	var outgoingMessage *internalMessage

	friendlyTimeString := currentTime.Format("3:04PM")

	if len(locations) == 0 {
		// Construct an internal struct, this case including our internal user id
		outgoingMessage = &internalMessage{MessageText: []byte("It is " + friendlyTimeString + " and there are no Tims open! :( "), MessageAuthorId: 0, MessageDisplayName: []byte("The Admins"), MessageId: messageId}
	} else {

		outString := "It is " + friendlyTimeString + " and the following Tims are open: "
		for i, v := range locations {
			if i != len(locations)-1 {
				outString += v + ", "
			} else {
				outString += v
			}
		}

		outgoingMessage = &internalMessage{MessageText: []byte(outString), MessageAuthorId: 0, MessageDisplayName: []byte("System"), MessageId: messageId}
	}

	// Send the message out for broadcast
	theLobby.broadcast <- outgoingMessage
}

func (theLobby *lobby) sendQuote(messageDisplayName string, messageAuthorId int64, messageId int64) {
	res, err := http.Get("http://api.icndb.com/jokes/random?firstName=John&amp;lastName=Doe")
	if err != nil {
		log.Println("error getting quote", err)
		return
	}
	var f interface{}
	err = json.NewDecoder(res.Body).Decode(&f)
	if err != nil {
		log.Println("error decoding json", err)
		return
	}

	m := f.(map[string]interface{})

	successType := m["type"].(string)
	if successType != "success" {
		log.Println("quotes api issue")
		return
	}

	valueInterface := m["value"].(map[string]interface{})
	theJoke := valueInterface["joke"].(string)

	outgoingMessage := &internalMessage{MessageText: []byte(theJoke), MessageAuthorId: messageAuthorId, MessageDisplayName: []byte(messageDisplayName), MessageId: messageId}

	theLobby.broadcast <- outgoingMessage
}
