package main

import (
	"database/sql"
	// We need to import with the underscore because we don't directly use the library,
	// but we rely on it being available under sql.Open()
	"github.com/gorilla/context"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"net/url"
)

var db *sql.DB

func main() {
	initializeDatabase()
	defer db.Close()
	// Course page - this is the pretty page shown to the user - should write HTML
	http.HandleFunc("/course/", arbitraryChatPageHandler)
	// This is the websocket connection link - should upgrade to a websocket connection
	http.HandleFunc("/ws/course/", arbitraryWebsocketHandler)
	// Catch-all, including the home page
	http.HandleFunc("/", indexPageHandler)

	// This is an endpoint that returns one message over a GET (not websockets)
	// The URL style will be /singleMessage/courseCode/messageId
	// GET /singleMessage/cisc332/37
	http.HandleFunc("/singleMessage/", singleMessageHandler)

	// This endpoint will give the user a certain amount of messages, up to a limit
	// The URL style will be /getSomeMessages/:courseCode/:numMessages
	// GET /getSomeMessages/cisc332/50
	http.HandleFunc("/getSomeMessages/", getSomeMessagesHandler)

	// This does double duty:
	// The GET side just gives a basic login page with a form, which should be built from a template
	// The POST side actually takes the form and authenticates a user
	http.HandleFunc("/login", loginPageHandler)

	// This just deletes the user's session
	http.HandleFunc("/logout", logoutPageHandler)

	// ListenAndServe should never return, if it does, it's a fatal error
	// We are wrapping http.DefaultServeMux in context.ClearHandler because gorilla tells
	// us to... It is supposed used to prevent a resource leak.
	log.Fatal(http.ListenAndServe(":9999", context.ClearHandler(http.DefaultServeMux)))
}

func initializeDatabase() {
	// This line is important, because otherwise we shadow `db` in the global scope
	// it doesn't actually do what we want
	var err error
	db, err = sql.Open("sqlite3", "a.sqlite3")
	if err != nil {
		// We can't do anything without the database
		log.Fatal(err)
	}
	// Defer closing until the function exits

	// Forces the database to connect and writes to disk
	db.Ping()

	tableStmt := "CREATE TABLE IF NOT EXISTS messages (id INTEGER NOT NULL PRIMARY KEY, " +
		"channel_id INTEGER NOT NULL, " +
		"message_text TEXT NOT NULL, channel_msg_id INTEGER NOT NULL, author_display_name TEXT NOT NULL);" +
		"CREATE TABLE IF NOT EXISTS lobbies (id INTEGER NOT NULL PRIMARY KEY, course_code TEXT UNIQUE NOT NULL);" +
		"CREATE TABLE IF NOT EXISTS users (id INTEGER NOT NULL PRIMARY KEY, netid TEXT UNIQUE NOT NULL, ics_url TEXT UNIQUE NOT NULL);" +
		"CREATE TABLE IF NOT EXISTS stars (id INTEGER NOT NULL PRIMARY KEY, starrer_id INTEGER NOT NULL, staree_id INTEGER NOT NULL, message_id INTEGER NOT NULL);"
	_, err = db.Exec(tableStmt)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database successfully created")
}

type User struct {
	id      int64
	netid   string
	ics_url url.URL
}
