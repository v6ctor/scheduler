package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"
)

type termData struct {
	Count   int      `json:"totalCount"`
	Courses []course `json:"data"`
}

type course struct {
	ID         int    `json:"id"`
	CourseRef  string `json:"courseReferenceNumber"`
	CourseNum  string `json:"courseNumber"`
	Subject    string `json:"subject"`
	CourseType string `json:"scheduleTypeDescription"`
	Title      string `json:"courseTitle"`
}

func setTerm(semester string, year string) string {
	var term strings.Builder

	term.WriteString(year)

	if semester == "fall" {
		term.WriteString("04")
	} else {
		term.WriteString("02")
	}

	return term.String()
}

func requestCourses(offset string, max string, client http.Client) (*termData, error) {
	// Note: Endpoint is limited to 500 courses per request, we'll use some sort of pagination
	// Will probably not exceed 1000 courses so, for now, 2 requests will be enough

	var swarthmoreUrl strings.Builder

	swarthmoreUrl.WriteString("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/searchResults/searchResults?txt_term=")
	swarthmoreUrl.WriteString(setTerm("fall", "2024"))
	swarthmoreUrl.WriteString("&startDatepicker=&endDatepicker=&uniqueSessionId=cwtoq1717225731537&pageOffset=")
	swarthmoreUrl.WriteString(offset)
	swarthmoreUrl.WriteString("&pageMaxSize=")
	swarthmoreUrl.WriteString(max)
	swarthmoreUrl.WriteString("&sortColumn=subjectDescription&sortDirection=asc")

	resp, err := client.Get(swarthmoreUrl.String())

	if err != nil {
		return nil, fmt.Errorf("failed to fulfill GET request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	data := new(termData)

	if err := json.Unmarshal(body, data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return data, nil
}

func main() {
	jar, _ := cookiejar.New(nil)

	client := http.Client{
		Jar: jar,
	}

	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/registration")
	time.Sleep(2000 * time.Millisecond)
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/selfServiceMenu/data")
	time.Sleep(2000 * time.Millisecond)
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/term/termSelection?mode=search")
	time.Sleep(2000 * time.Millisecond)
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/selfServiceMenu/data")
	time.Sleep(2000 * time.Millisecond)
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/term/termSelection?mode=search")
	time.Sleep(2000 * time.Millisecond)
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/selfServiceMenu/data")
	time.Sleep(2000 * time.Millisecond)
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/classSearch/getTerms?searchTerm=&offset=1&max=10&_=1717271345154")
	time.Sleep(2000 * time.Millisecond)
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/term/search?mode=search&term=202404&studyPath=&studyPathText=&startDatepicker=&endDatepicker=&uniqueSessionId=l47z91717271338036")
	time.Sleep(2000 * time.Millisecond)
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/classSearch/classSearch")

	processedCourses := 0

	var data termData

	for {
		if processedCourses == 0 {
			courses, err := requestCourses("0", "500", client)

			if err != nil {
				fmt.Println("retrying in 7 seconds")
				time.Sleep(7000 * time.Millisecond)
			}

			data.Courses = append(data.Courses, courses.Courses...)
			data.Count = courses.Count

		} else {
			courses, err := requestCourses(strconv.Itoa(processedCourses), "500", client)

			if err != nil {
				fmt.Println("retrying in 7 seconds")
				time.Sleep(7000 * time.Millisecond)
			}

			data.Courses = append(data.Courses, courses.Courses...)
		}

		processedCourses += 500

		fmt.Println("Processed:", processedCourses)

		if processedCourses >= data.Count {
			fmt.Println("Finished!")
			fmt.Println(data)
			break
		}
	}

}
