package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type termData struct {
	Count   int      `json:"totalCount"`
	Courses []course `json:"data"`
}

type course struct {
	ID              int               `json:"id"`
	Ref             string            `json:"courseReferenceNumber"`
	Number          string            `json:"courseNumber"`
	Subject         string            `json:"subject"`
	Type            string            `json:"scheduleTypeDescription"`
	Title           string            `json:"courseTitle"`
	DescriptionUrl  string            `json:""`
	Description     string            `json:""`
	Credits         float32           `json:"creditHours"`
	MaxEnrollment   int               `json:"maximumEnrollment"`
	Enrolled        int               `json:"enrollment"`
	Availability    int               `json:"seatsAvailable"`
	Faculty         []faculty         `json:"faculty"`
	MeetingsFaculty []meetingsFaculty `json:"meetingsFaculty"`
	Attributes      []attribute       `json:"sectionAttributes"`
}

type faculty struct {
	ID    string `json:"bannerId"`
	Ref   string `json:"courseReferenceNumber"`
	Name  string `json:"displayName"`
	Email string `json:"emailAddress"`
}

type meetingsFaculty struct {
	Section     string `json:"category"`
	Ref         string `json:"courseReferenceNumber"`
	MeetingTime meetingTime
}

type meetingTime struct {
	Begin         string  `json:"beginTime"`
	BuildingShort string  `json:"building"`
	BuildingLong  string  `json:"buildingDescription"`
	Room          string  `json:"room"`
	Section       string  `json:"category"`
	Ref           string  `json:"courseReferenceNumber"`
	EndDate       string  `json:"endDate"`
	EndTime       string  `json:"endTime"`
	StartDate     string  `json:"startDate"`
	Hours         float32 `json:"hoursWeek"`
	TypeShort     string  `json:"meetingType"`
	TypeLong      string  `json:"meetingTypeDescription"`
	Monday        bool    `json:"monday"`
	Tuesday       bool    `json:"tuesday"`
	Wednesday     bool    `json:"wednesday"`
	Thursday      bool    `json:"thursday"`
	Friday        bool    `json:"friday"`
	Saturday      bool    `json:"saturday"`
	Sunday        bool    `json:"sunday"`
}

type attribute struct {
	CodeShort string `json:"code"`
	CodeLong  string `json:"description"`
	Ref       string `json:"courseReferenceNumber"`
}

func timer(name string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", name, time.Since(start))
	}
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

func requestCourses(term string, offset string, max string, client http.Client) (*termData, error) {
	// Note: Endpoint is limited to 500 courses per request, we'll use some sort of pagination
	// Will probably not exceed 1000 courses so, for now, 2 requests will be enough

	var swarthmoreUrl strings.Builder

	swarthmoreUrl.WriteString("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/searchResults/searchResults?txt_term=")
	swarthmoreUrl.WriteString(term)
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

func getCourseDescriptionUrls(term string, data termData) {
	for i := range data.Courses {
		var formattedUrl strings.Builder

		formattedUrl.WriteString("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/searchResults/getCourseDescription?term=")
		formattedUrl.WriteString(term)
		formattedUrl.WriteString("&courseReferenceNumber=")
		formattedUrl.WriteString(data.Courses[i].Ref)

		url := formattedUrl.String()

		data.Courses[i].DescriptionUrl = url
	}
}

func requestCourseDescription(index int, data termData, client http.Client, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := client.Get(data.Courses[index].DescriptionUrl)

	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Println(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))

	if err != nil {
		fmt.Println(err)
	}

	section := doc.Find(`section[aria-labelledby="courseDescription"]`)
	_, formattedString, ok := strings.Cut(section.Text(), "Section information text:")

	if !ok {
		data.Courses[index].Description = "No course description provided. Contact Professor."
	} else {
		data.Courses[index].Description = strings.TrimSpace(formattedString)
	}
}

func main() {
	var semester, year string
	var wg sync.WaitGroup

	fmt.Print("Enter your semester (i.e. fall): ")
	fmt.Scan(&semester)

	fmt.Print("Enter your year (i.e. 2024): ")
	fmt.Scan(&year)

	term := setTerm(semester, year)

	defer timer("main")()

	jar, _ := cookiejar.New(nil)

	client := http.Client{
		Jar: jar,
	}

	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/registration")
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/term/termSelection?mode=search")
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/classSearch/getTerms?searchTerm=&offset=1&max=10&_=1717271345154")
	client.Get("https://studentregistration.swarthmore.edu/StudentRegistrationSsb/ssb/term/search?mode=search&term=202404&studyPath=&studyPathText=&startDatepicker=&endDatepicker=&uniqueSessionId=l47z91717271338036")

	fmt.Println("Hydrated client")

	processedCourses := 0

	var data termData

	fmt.Println("Requesting courses")

	for {
		if processedCourses == 0 {
			courses, err := requestCourses(term, "0", "500", client)

			if err != nil {
				fmt.Println(err)
				fmt.Println("retrying in 7 seconds")
				time.Sleep(7000 * time.Millisecond)
			}

			data.Courses = append(data.Courses, courses.Courses...)
			data.Count = courses.Count

		} else {
			courses, err := requestCourses(term, strconv.Itoa(processedCourses), "500", client)

			if err != nil {
				fmt.Println("retrying in 7 seconds")
				time.Sleep(7000 * time.Millisecond)
			}

			data.Courses = append(data.Courses, courses.Courses...)
		}

		processedCourses += 500

		if processedCourses >= data.Count {
			fmt.Println("Finished processing:", data.Count, "courses")
			break
		}
	}

	getCourseDescriptionUrls(term, data)

	for i := range data.Courses {
		wg.Add(1)
		go requestCourseDescription(i, data, client, &wg)
	}

	wg.Wait()

	output, err := json.MarshalIndent(data, "", "\t")

	if err != nil {
		fmt.Println(err)
	}

	err = os.WriteFile("courses.json", output, 0644)

	if err != nil {
		fmt.Println(err)
	}

}
