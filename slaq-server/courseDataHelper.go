package main
import (
	"github.com/apognu/gocal"
	"log"
	"net/http"
	"strings"
	"errors"
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
