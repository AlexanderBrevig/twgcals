package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	t := time.Now().Format(time.RFC3339)
	tmax := time.Now().Add(time.Hour * 25).Format(time.RFC3339)
	calendars, err := srv.CalendarList.List().Do()
	if err != nil {
	}
	for _, calendar := range *&calendars.Items {
		var project string
		if len(calendar.SummaryOverride) > 0 {
			project = calendar.SummaryOverride
		} else {
			project = calendar.Summary
		}
		events, err := srv.Events.List(calendar.Id).ShowDeleted(false).
			SingleEvents(true).TimeMin(t).TimeMax(tmax).MaxResults(10).OrderBy("startTime").Do()
		if err != nil {
			log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
		}
		if len(events.Items) > 0 {
			for _, item := range events.Items {
				date := item.Start.DateTime
				if date == "" {
					date = item.Start.Date
				}
				enddate := item.End.DateTime
				task := fmt.Sprintf("project:%s %v due:%v until:%v rc.dateformat:Y-M-DTH:N:SZ\n", project, item.Summary, date, enddate)

				cmd := exec.Command("task", fmt.Sprintf("project:%s", project), fmt.Sprintf("desc:'%s'", item.Summary), fmt.Sprintf("due:%v", date), "count")
				stdout, err := cmd.Output()
				if err != nil {
					log.Fatal(err)
				}
				count, err := strconv.Atoi(string(stdout[0]))
				if err != nil {
					log.Fatal(err)
				}
				if count == 0 {
					taslparts := strings.Split("add "+task, " ")
					taskadd := exec.Command("task", taslparts...)
					err = taskadd.Run()
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}