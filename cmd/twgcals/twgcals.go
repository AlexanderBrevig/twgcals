package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	user, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	credentialPath := filepath.Join(user, "credentials.json")
	if mp := os.Getenv("TWGCALS_CREDENTIALS"); mp != "" {
		credentialPath = mp
	}
	b, err := ioutil.ReadFile(credentialPath)
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
				loc, err := time.LoadLocation("Local")
				if err != nil {
					log.Fatal(err)
				}
				parsed, err := time.Parse(time.RFC3339, item.Start.DateTime)
				if err != nil {
					log.Fatal(err)
				}
				parsed = parsed.In(loc)
				date := parsed.Format("2006-01-02T15:04:05.999999-07:00")
				parsed, err = time.Parse(time.RFC3339, item.End.DateTime)
				if err != nil {
					log.Fatal(err)
				}
				parsed = parsed.In(loc)
				enddate := parsed.Format("2006-01-02T15:04:05.999999-07:00")
				const COUNTSEL = 0
				const DESCSEL = 1
				const ADDCMD = 2
				const PROJSEL = 3
				const DUESEL = 4
				taskparts := []string{
					"count",
					fmt.Sprintf("desc:'%s'", item.Summary),
					"add",
					fmt.Sprintf("project:%s", project),
					fmt.Sprintf("due:%v", date),
					fmt.Sprintf("until:%v", enddate),
					"rc.dateformat:Y-M-DTH:N:SZ",
					item.Summary,
					"+twgcals",
				}
				// Add tags from calendar name
				// Please name your calendar project.subproj
				// F.ex work.client
				tags := []string{}
				for _, tag := range strings.Split(project, ".") {
					tags = append(tags, "+"+tag)
				}
				taskparts = append(taskparts, tags...)

				cmd := exec.Command("task", taskparts[PROJSEL], taskparts[DUESEL], taskparts[DESCSEL], taskparts[COUNTSEL])
				stdout, err := cmd.Output()
				if err != nil {
					log.Fatal(err)
				}
				count, err := strconv.Atoi(string(stdout[0]))
				if err != nil {
					log.Fatal(err)
				}
				if count == 0 {
					taskadd := exec.Command("task", taskparts[ADDCMD:]...)
					err = taskadd.Run()
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}
