package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type courseData struct {
	CourseTitle string
	HostAndPort string
	CourseCode  string
}

func arbitraryChatPageHandler(w http.ResponseWriter, r *http.Request) {
	// Get the segments out of the URL path
	// e.g. www.example.com/one/two --> we want "one", "two"
	// But I think Go gives us, as a URL, /one/two
	// which means that we actually get "", "one", "two": we get the empty string
	// So let's assert the length is three, not two
	// Because we want www.example.com/course/COURSECODE
	urlPathSegments := strings.Split(r.URL.Path, "/")
	if len(urlPathSegments) != 3 {
		// If they are at /course/coursecode/x, then we don't know what they want, so give a 404
		http.Error(w, http.StatusText(404), 404)
		return
	}
	courseCode := urlPathSegments[2]
	tmpl := template.Must(template.ParseFiles("templates/course_page.html"))

	cData := courseData{CourseTitle: "Databases", HostAndPort: r.Host, CourseCode: courseCode}
	tmpl.Execute(w, cData)

}

func indexPageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "We are on index page")
}
