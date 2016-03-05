package main

import (
	"database/sql"
	// We need to import with the underscore because we don't directly use the library,
	// but we rely on it being available under sql.Open()
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
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

	// ListenAndServe should never return, if it does, it's a fatal error
	log.Fatal(http.ListenAndServe(":9999", nil))
}

func initializeDatabase() {
	// This line is important, because otherwise we shadow `db` in the global scope
	// it doesn't actually do what we want
	var err error
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		// We can't do anything without the database
		log.Fatal(err)
	}
	// Defer closing until the function exits

	// Forces the database to connect and writes to disk
	db.Ping()

	tableStmt := "CREATE TABLE IF NOT EXISTS messages (id INTEGER NOT NULL PRIMARY KEY, " +
		"channel_id INTEGER NOT NULL, " +
		"message_text TEXT NOT NULL); CREATE TABLE IF NOT EXISTS lobbies (id INTEGER NOT NULL PRIMARY KEY, course_code TEXT NOT NULL);"
	_, err = db.Exec(tableStmt)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database successfully created")
}
