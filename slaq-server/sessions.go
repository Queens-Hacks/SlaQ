package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	"net/http"
)

const SESSION_NAME = "slaq-server"
const SECRET_KEY = "my-secret"

var sessionStore = sessions.NewCookieStore([]byte(SECRET_KEY))

type User struct {
	id      int64
	netid   string
	ics_url string
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// It's a GET - let's give them the login page
		tmpl := template.Must(template.ParseFiles("templates/login_page.html"))
		tmpl.Execute(w, nil)
		return
	case "POST":
		// Get the username and password from the incoming post
		r.ParseForm()
		username := r.Form.Get("username")
		password := r.Form.Get("password")
		// If they don't log in, we won't redirect them to an authenticated page

		// Sanity check
		if username != "" && password != "" {
			icsUrl, err := getIcsUrlFromUsernameAndPassword(username, password)
			if err != nil {
				log.Println("sessions error", err)
				http.Error(w, "Bad login", http.StatusUnauthorized)
				return
			}
			// Now the netid and password are both valid
			user, err := getUserFromNetid(username)
			if err != nil {
				// If there is an error, then we weren't able to construct a user struct
				// This is probably because the user isn't in our database yet!

				// So let's fix that - create an entry for the user in our database
				res, err := db.Exec("INSERT INTO users(id, netid, ics_url) VALUES (?, ?, ?)", nil, username, icsUrl)
				if err != nil {
					log.Println("Error getting new user id: ", err)
					return
				}
				// Get the user's id, so we can add it to the struct
				userId, err := res.LastInsertId()
				user.ics_url = icsUrl
				user.id = userId
				user.netid = username
			}
			// Now give the user a session - they just successfully logged in
			theSession, err := sessionStore.Get(r, SESSION_NAME)
			if err != nil {
				log.Println("Error loading session", err)
				http.Error(w, "Couldn't get you a session!", http.StatusInternalServerError)
				return
			}
			// We must save it as a string, not as a URL
			theSession.Values["icsUrl"] = user.ics_url
			theSession.Values["username"] = user.netid
			theSession.Save(r, w)
			http.Redirect(w, r, "/", 302)
		} else {
			http.Redirect(w, r, "/", 302)
		}

	}
}

func logoutPageHandler(w http.ResponseWriter, r *http.Request) {
	theSession, err := sessionStore.Get(r, SESSION_NAME)
	if err != nil {
		log.Println("Error loading session", err)
		http.Error(w, "Couldn't get you a session!", http.StatusInternalServerError)
	}
	// Magic delete session value is -1
	theSession.Options.MaxAge = -1
	theSession.Save(r, w)
	http.Redirect(w, r, "/", 302)
}

func getUserFromNetid(netid string) (User, error) {
	// Try to find the user in the database
	rows, err := db.Query("SELECT * FROM users WHERE netid = ?", netid)

	if err != nil {
		log.Println("Error looking up user in database")
		return User{}, errors.New("Internal error")
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var netid string
		var ics_url string

		err = rows.Scan(&id, &netid, &ics_url)
		if err != nil {
			log.Println("Error scanning from database")
			return User{}, errors.New("Internal error")
		}

		return User{
			id:      id,
			netid:   netid,
			ics_url: ics_url,
		}, nil
	}

	// In this case, everything worked, we just got back a table with zero rows
	return User{}, errors.New("No user found")
}
