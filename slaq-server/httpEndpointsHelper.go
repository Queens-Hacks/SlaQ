package main

import (
	"database/sql"
	"errors"
	"log"
)

func getChannelIdFromCourseCode(courseCode string) (int64, error) {
	// This will escape courseCode for us against SQL injection attacks
	rows, err := db.Query("SELECT * FROM lobbies WHERE course_code = ?", courseCode)
	if err != nil {
		log.Println("Error loading lobby from database", err)
		return -1, errors.New("Error loading lobby from database")
	}

	// We will pull the first one of these we see, since we have a UNIQUE constraint in the
	// database, it isn't possible to get more than one result. We want these variables
	// in the outer scope, versus within the rows.Next() scope
	var channelId int64
	var course_code string
	var last_message_id int64
	var foundCourse bool
	for rows.Next() {
		err = rows.Scan(&channelId, &course_code, &last_message_id)
		// We can close here, because we know there will only ever be one - also if we look
		// at the logic of the if condition, we either break or return
		// We never continue into another iteration
		rows.Close()
		if err != nil {
			log.Println("Error scanning lobbies table row into variables", err)
			return -1, errors.New("Error scanning lobbies table row into variables")
		} else {
			foundCourse = true
			break
		}
	}

	// If we got zero rows, we didn't flip the boolean, which means we didn't find the course,
	// which means the user arguably sent a bad request
	if !foundCourse {
		log.Println("No course by that name found", courseCode)
		return -1, errors.New("Could not find a course by that name")
	}

	return channelId, nil
}

func readOneMessageFromRows(rows *sql.Rows) (externalMessage, error) {
	// There should only be one message, if there isn't just one, we have a problem
	// with our database schema and constraints
	var dbMessageId int64
	var dbChannelId int64
	var dbMessageText string
	var dbChannelMessageId int64
	var dbAuthorName string
	var dbAuthorId int64
	err := rows.Scan(&dbMessageId, &dbChannelId, &dbMessageText, &dbChannelMessageId, &dbAuthorName, &dbAuthorId)

	if err != nil {
		log.Println("Error scanning messagestable row into variables: ", err)
		return externalMessage{}, errors.New("Error scanning messagestable row into variables")
	}

	messageToSend := externalMessage{
		MessageText:        dbMessageText,
		MessageDisplayName: dbAuthorName,
		MessageId:          dbMessageId,
		Stars:              getNumStarsForMessageId(dbMessageId),
	}

	return messageToSend, nil
}

func getNumStarsForMessageId(messageId int64) int64 {
	rows, err := db.Query("select id from stars where stars.message_id = ?", messageId)
	if err != nil {
		log.Println("error reading stars for message: ", err)
	}
	defer rows.Close()
	var counter int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		counter += 1
	}
	return counter
}
