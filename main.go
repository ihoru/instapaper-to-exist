package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/ihoru/existio_instapaper/state"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ihoru/existio_instapaper/existio_client"
	"github.com/ihoru/existio_instapaper/storage"
)

// Configuration variables
var (
	ExistClientID        string
	ExistClientSecret    string
	ExistOAuth2Return    string
	ExistAttributeName   string
	InstapaperArchiveRSS string
	storageInstance      *storage.Storage
)

// RSS feed structures
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Items []Item `xml:"item"`
}

type Item struct {
	GUID string `xml:"guid"`
}

//goland:noinspection GoUnhandledErrorResult
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ExistClientID = os.Getenv("EXIST_CLIENT_ID")
	ExistClientSecret = os.Getenv("EXIST_CLIENT_SECRET")
	ExistOAuth2Return = os.Getenv("EXIST_OAUTH2_RETURN")
	if ExistOAuth2Return == "" {
		ExistOAuth2Return = "http://localhost:9009/"
	}
	ExistAttributeName = os.Getenv("EXIST_ATTRIBUTE_NAME")
	if ExistAttributeName == "" {
		ExistAttributeName = "Articles read"
	}
	InstapaperArchiveRSS = os.Getenv("INSTAPAPER_ARCHIVE_RSS")

	var missingVars []string
	if ExistClientID == "" {
		missingVars = append(missingVars, "EXIST_CLIENT_ID")
	}
	if ExistClientSecret == "" {
		missingVars = append(missingVars, "EXIST_CLIENT_SECRET")
	}
	if InstapaperArchiveRSS == "" {
		missingVars = append(missingVars, "INSTAPAPER_ARCHIVE_RSS")
	}

	if len(missingVars) > 0 {
		fmt.Fprintf(os.Stderr, "Error: Required environment variables are missing: %v\n", missingVars)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Please ensure these variables are set either:")
		fmt.Fprintln(os.Stderr, "1. As environment variables in your shell")
		fmt.Fprintln(os.Stderr, "2. In a .env file in the same directory as this executable")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example .env file content:")
		fmt.Fprintln(os.Stderr, "EXIST_CLIENT_ID=your_client_id_here")
		fmt.Fprintln(os.Stderr, "EXIST_CLIENT_SECRET=your_client_secret_here")
		fmt.Fprintln(os.Stderr, "INSTAPAPER_ARCHIVE_RSS=https://instapaper.com/archive/rss/123/XXX")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "See the README.md file for detailed setup instructions.")
		os.Exit(1)
	}

	// Initialize storage
	storageInstance = storage.NewStorage("exist-instapaper-go")
}

// GetExistSession initializes and authenticates with Exist.io
func GetExistSession(sessions *state.Sessions, client *http.Client) (*existio_client.OAuth2, error) {
	auth := existio_client.NewOAuth2(
		ExistOAuth2Return,
		ExistClientID,
		ExistClientSecret,
		"media_write",
		client,
	)

	if sessions.Exist.RefreshToken != "" {
		auth.RefreshToken = sessions.Exist.RefreshToken
	}
	if sessions.Exist.AccessToken != "" {
		auth.AccessToken = sessions.Exist.AccessToken
	}
	if !sessions.Exist.LastRefresh.IsZero() {
		auth.LastRefresh = sessions.Exist.LastRefresh
	}

	if err := auth.EvaluateTokens(); err != nil {
		return nil, fmt.Errorf("failed to evaluate tokens: %v", err)
	}

	sessions.Exist.AccessToken = auth.AccessToken
	sessions.Exist.RefreshToken = auth.RefreshToken
	sessions.Exist.LastRefresh = auth.LastRefresh

	state.SaveStates(storageInstance, sessions, nil, nil)
	return auth, nil
}

// GetExistAttrs initializes the Exist.io attributes client
func GetExistAttrs(sessions *state.Sessions, client *http.Client) (*existio_client.Attrs, error) {
	accessToken := sessions.Exist.AccessToken
	if accessToken == "" {
		return nil, fmt.Errorf("access token not found in sessions")
	}

	attrs := existio_client.NewAttrs(accessToken, 5*time.Second, client)
	if err := attrs.AcquireLabel("media", ExistAttributeName, existio_client.ValueTypeInteger, false); err != nil {
		return nil, fmt.Errorf("failed to acquire label: %v", err)
	}

	state.SaveStates(storageInstance, sessions, nil, nil)
	return attrs, nil
}

// Main function
func main() {
	// Parse command line arguments
	daysFlag := flag.Int("days", 3, "Number of days to consider for reading stats")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")
	todayValueFlag := flag.Int("today", -1, "Value to set for today's stats")
	yesterdayValueFlag := flag.Int("yesterday", -1, "Value to set for yesterdays's stats")
	flag.Parse()

	// Set up logging
	if *verboseFlag {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	} else {
		log.SetFlags(log.Ldate | log.Ltime)
	}

	days := *daysFlag
	if days <= 0 {
		log.Fatal("Days must be a positive integer")
	}

	// Load states
	sessions, articles, readingStats := state.LoadStates(storageInstance)

	// Initialize HTTP client
	client := existio_client.StartSession()

	// Get Exist.io session
	_, err := GetExistSession(&sessions, client)
	if err != nil {
		log.Fatalf("Failed to get Exist session: %v", err)
	}

	// Get Exist.io attributes client
	attrs, err := GetExistAttrs(&sessions, client)
	if err != nil {
		log.Fatalf("Failed to get Exist attributes: %v", err)
	}

	// Fetch Instapaper RSS feed
	resp, err := client.Get(InstapaperArchiveRSS)
	if err != nil {
		log.Fatalf("Failed to fetch Instapaper feed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Instapaper: Failed to fetch feed! Status code: %d", resp.StatusCode)
	}

	// Parse RSS feed
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	var rss RSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		log.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Process articles
	today := time.Now().Format("2006-01-02")
	for _, item := range rss.Channel.Items {
		url := item.GUID
		if !articles[url] {
			articles[url] = true
			readingStats[today]++
		}
	}
	if *todayValueFlag >= 0 {
		readingStats[today] = *todayValueFlag
	}

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if *yesterdayValueFlag >= 0 {
		readingStats[yesterday] = *yesterdayValueFlag
	}

	// Prepare data for submission
	var data []map[string]interface{}
	currentTime := time.Now()
	for i := 0; i < days; i++ {
		date := currentTime.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		count := readingStats[dateStr]
		log.Printf("%s = %d", dateStr, count)
		data = append(data, attrs.FormatSubmission(date, ExistAttributeName, count))
	}

	// Submit data to Exist.io
	if err := attrs.UpdateBatch(data); err != nil {
		log.Fatalf("Failed to update batch: %v", err)
	}

	// Save states
	state.SaveStates(storageInstance, &sessions, &articles, &readingStats)
}
