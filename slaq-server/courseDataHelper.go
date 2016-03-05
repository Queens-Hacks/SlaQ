package main

import (
	"encoding/json"
	"errors"
	"github.com/apognu/gocal"
	"log"
	"net/http"
	"strings"
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

		rows, err := db.Query("SELECT * from courses WHERE course_taker_id = ?", user.id)
		if err != nil {
			log.Println("error reading from database", err)
		}
		defer rows.Close()

		var sawACourse = false
		var courses []string
		for rows.Next() {

			var id int64
			var course_code string
			var course_taker_id int64
			err = rows.Scan(&id, &course_code, &course_taker_id)
			if err != nil {
				log.Println("Error scanning from database", err)
				break
			}
			courses = append(courses, course_code)
			sawACourse = true
		}
		if !sawACourse {
			courses, err = parseIcsFromUrl(user.ics_url)
			if err != nil {
				log.Println("Error parsing ics", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			for _, v := range courses {
				_, err := db.Exec("INSERT INTO courses(id, course_code, course_taker_id) VALUES (?, ?, ?);", nil, v, user.id)
				if err != nil {
					log.Println("error inserting course into db: ", err)
					break
				}
			}
		}

		json.NewEncoder(w).Encode(courses)

	} else {
		log.Println("error casting netid to string")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

}
