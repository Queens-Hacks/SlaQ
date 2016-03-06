package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/apognu/gocal"
	"log"
	"net/http"
	"strings"
)

func parseIcsFromUrl(icsUrl string) ([]string, error) {
	// Download the .ics from the website
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(icsUrl)

	// This will probably fire if the internet is now
	if err != nil {
		log.Println("Cannot get calendar:", err)
		return make([]string, 0), errors.New("Couldn't get calendar url")
	}

	defer resp.Body.Close()

	// Parse the io.Reader
	c := gocal.NewParser(resp.Body)
	c.Parse()

	// Our empty list of courses
	var courseList []string

	// Users won't have many courses - probably max of eight
	// And those will probably have max of three sections - lecture, lab, tutorial
	// So our realistic upper bound on the size of this map is 24, so that's fine
	// we shouldn't run into any speed/scaling issues
	var coursesSeen = make(map[string]bool)

	// For every event
	for _, v := range c.Events {
		// The summary contains the course code and section number
		// so we have to parse it out
		summary := v.Summary
		splits := strings.Split(summary, " ")
		// The length can vary - they all start with [COURSE_TYPE] [COURSE_NUM] e.g. CMPE 332
		// but there can be extra stuff added on.

		// All we should guarantee is at least two, so our slice operations don't fail
		if len(splits) < 2 {
			log.Println("Can't understand course", summary)
			continue
		}
		courseType := splits[0]
		courseNumber := splits[1]

		// This is just being done we can join the string, maybe there's a better way, but this works
		courseTypeNumSlice := []string{courseType, courseNumber}
		oneCourse := strings.Join(courseTypeNumSlice, "")

		// We are using the map because course can have the same name & code, but different sections
		// We just want each unique course
		if !coursesSeen[oneCourse] {
			courseList = append(courseList, oneCourse)
			coursesSeen[oneCourse] = true
		}

	}

	// This could be an empty list, which is valid
	return courseList, nil
}

// Returns a JSON array of strings indicating the courses the user is enrolled in
func getMyCoursesHandler(w http.ResponseWriter, r *http.Request) {
	theSession, err := sessionStore.Get(r, SESSION_NAME)
	if err != nil {
		log.Println("Error getting a session")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// User didn't log in properly
	if theSession.Values["username"] == nil || theSession.Values["username"] == "" {
		log.Println("unauth user trying to get courses")
		http.Error(w, "not allowed", http.StatusUnauthorized)
		return
	}
	// Cast username to a string
	netid, ok := theSession.Values["username"].(string)

	if ok {
		user, err := getUserFromNetid(netid)
		if err != nil {
			log.Println("error getting user id")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Get all the courses that the user is enrolled in
		rows, err := db.Query("SELECT * from courses WHERE course_taker_id = ?", user.id)
		if err != nil {
			log.Println("error reading from database", err)
		}
		defer rows.Close()

		// If the user has zero courses, we want to update our database with the latest
		// otherwise pull from the cache
		var sawACourse = false

		// The slice to return to the user as a JSON array
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
			// If we see a single course, set this true
			sawACourse = true
		}
		if !sawACourse {
			// We didn't see a course, so either the user isn't enrolled in any, or
			// we just don't have any

			// We will optimize for the common case, that the users of the system are current
			// Queen's students, and we will assume that we just don't have the entries

			// Possible TODO: Add a flag per user indicating whether or not we have read their
			// courses into the database
			courses, err = parseIcsFromUrl(user.ics_url)
			if err != nil {
				log.Println("Error parsing ics", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			for _, v := range courses {
				// Insert each course into the database
				// This is really bad because there will be multiple course names by different takers
				// The canonical way would probably be to have a junction table
				// e.g. courses has one course per row
				// then CourseUserJunction has a course_id and user_id entry, and joins the two
				// TODO: Make this canonical (proper) relational style
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
