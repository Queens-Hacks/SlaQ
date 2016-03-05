package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	//"github.com/gorilla/mux"
	//"github.com/gorilla/sessions"
)

var db *sql.DB

func main() {
	initializeDatabase()

}

func initializeDatabase() {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		// We can't do anything without the database
		log.Fatal(err)
	}
	// Defer closing until the function exits
	defer db.Close()
	// Forces the database to connect and writes to disk
	db.Ping()

	tableStmt := "CREATE TABLE IF NOT EXISTS messages (id INTEGER NOT NULL PRIMARY KEY, " +
		"channel_id INTEGER NOT NULL, " +
		"message_text TEXT NOT NULL);"
	_, err = db.Exec(tableStmt)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database successfully created")

	// Course page - this is the pretty page shown to the user - should write HTML
	http.HandleFunc("/course/", arbitraryChatPageHandler)
	// This is the websocket connection link - should upgrade to a websocket connection
	http.HandleFunc("/ws/course/", arbitraryWebsocketHandler)
	// Catch-all, including the home page
	http.HandleFunc("/", indexPageHandler)

	// ListenAndServe should never return, if it does, it's a fatal error
	log.Fatal(http.ListenAndServe(":9999", nil))
}
