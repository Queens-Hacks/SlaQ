package main

import (
	"github.com/apognu/gocal"
	"log"
	"net/http"
	"strings"
	"errors"
	"encoding/json"
)

func parseIcsFromUrl(icsUrl string) ([]string, error) {
	resp, err := http.Get(icsUrl)
	if err != nil {
		log.Println("Cannot get calendar:", err)
		return make([]string, 0), errors.New("Couldn't get calendar url")
	}
	defer resp.Body.Close()
	c := gocal.NewParser(resp.Body)
	c.Parse()
	var courseList []string
	var coursesSeen = make(map[string]bool)
	for _, v := range c.Events {
		summary := v.Summary
		splits := strings.Split(summary, " ")
		if len(splits) < 2 {
			log.Println("Can't understand course", summary)
			continue
		}
		courseType := splits[0]
		courseNumber := splits[1]
		courseTypeNumSlice := []string{courseType, courseNumber}
		oneCourse := strings.Join(courseTypeNumSlice, "")
		if !coursesSeen[oneCourse] {
			courseList = append(courseList, oneCourse)
			coursesSeen[oneCourse] = true
		}

	}

	return courseList, nil
}

func getMyCoursesHandler(w http.ResponseWriter, r *http.Request) {
	theSession, err := sessionStore.Get(r, SESSION_NAME)
	if err != nil {
		log.Println("Error getting a session")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if theSession.Values["username"] == nil || theSession.Values["username"] == "" {
		log.Println("unauth user trying to get courses")
		http.Error(w, "not allowed", http.StatusUnauthorized)
		return
	}
	netid, ok := theSession.Values["username"].(string)
	if ok {
		user, err := getUserFromNetid(netid)
		if err != nil {
			log.Println("error getting user id")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		courses, err := parseIcsFromUrl(user.ics_url)
		if err != nil {
			log.Println("Error parsing ics", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(courses)

	} else {
		log.Println("error casting netid to string")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

}
