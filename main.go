package existio_instapaper

import (
	"bufio"
	"encoding/gob"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ihoru/existio_instapaper/existio_client"
)

// Configuration variables
var (
	ExistClientID        string
	ExistClientSecret    string
	ExistOAuth2Return    string
	ExistAttributeName   string
	InstapaperArchiveRSS string
)

// State directory and files
var (
	StateDir         string
	SessionsFile     string
	ArticlesFile     string
	ReadingStatsFile string
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

// Sessions storage
type Sessions struct {
	Exist map[string]interface{}
}

// ReadingStats maps dates to article counts
type ReadingStats map[string]int

// Articles is a set of article URLs
type Articles map[string]bool

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) > 1 && (value[0] == '"' && value[len(value)-1] == '"' ||
			value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}

		// Set environment variable if not already set
		if os.Getenv(key) == "" {
			err := os.Setenv(key, value)
			if err != nil {
				panic(err)
			}
		}
	}

	return scanner.Err()
}

func init() {
	// Load environment variables from .env file if it exists
	err := loadEnvFile(".env")
	if err != nil {
		panic(err)
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

	// Check required environment variables
	if ExistClientID == "" || ExistClientSecret == "" || InstapaperArchiveRSS == "" {
		fmt.Fprintln(os.Stderr, "Error: Required configuration variables missing. See the documentation.")
		os.Exit(1)
	}

	// Set up state directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}
	StateDir = filepath.Join(homeDir, ".local", "state", "exist-instapaper-go")
	if err := os.MkdirAll(StateDir, 0755); err != nil {
		log.Fatalf("Failed to create state directory: %v", err)
	}

	SessionsFile = filepath.Join(StateDir, "sessions")
	ArticlesFile = filepath.Join(StateDir, "articles")
	ReadingStatsFile = filepath.Join(StateDir, "stats")
}

// LoadStates loads the state files
func LoadStates() (Sessions, Articles, ReadingStats) {
	sessions := Sessions{
		Exist: make(map[string]interface{}),
	}
	articles := make(Articles)
	readingStats := make(ReadingStats)

	// Load sessions
	if _, err := os.Stat(SessionsFile); err == nil {
		file, err := os.Open(SessionsFile)
		if err != nil {
			log.Printf("Failed to open sessions file: %v", err)
		} else {
			defer file.Close()
			decoder := gob.NewDecoder(file)
			if err := decoder.Decode(&sessions); err != nil {
				log.Printf("Failed to decode sessions: %v", err)
				log.Printf("Removing corrupted sessions file and creating a new one.")
				file.Close()            // Close the file before removing it
				os.Remove(SessionsFile) // Remove the corrupted file
				// Continue with empty sessions
			}
		}
	}

	// Load articles
	if _, err := os.Stat(ArticlesFile); err == nil {
		file, err := os.Open(ArticlesFile)
		if err != nil {
			log.Printf("Failed to open articles file: %v", err)
		} else {
			defer file.Close()
			decoder := gob.NewDecoder(file)
			if err := decoder.Decode(&articles); err != nil {
				log.Printf("Failed to decode articles: %v", err)
				log.Printf("Removing corrupted articles file and creating a new one.")
				file.Close()            // Close the file before removing it
				os.Remove(ArticlesFile) // Remove the corrupted file
				// Continue with empty articles
			}
		}
	}

	// Load reading stats
	if _, err := os.Stat(ReadingStatsFile); err == nil {
		file, err := os.Open(ReadingStatsFile)
		if err != nil {
			log.Printf("Failed to open reading stats file: %v", err)
		} else {
			defer file.Close()
			decoder := gob.NewDecoder(file)
			if err := decoder.Decode(&readingStats); err != nil {
				log.Printf("Failed to decode reading stats: %v", err)
				log.Printf("Removing corrupted reading stats file and creating a new one.")
				file.Close()                // Close the file before removing it
				os.Remove(ReadingStatsFile) // Remove the corrupted file
				// Continue with empty reading stats
			}
		}
	}

	return sessions, articles, readingStats
}

// SaveStates saves the state files
func SaveStates(sessions *Sessions, articles *Articles, readingStats *ReadingStats) {
	// Save sessions
	if sessions != nil {
		file, err := os.Create(SessionsFile)
		if err != nil {
			log.Printf("Failed to create sessions file: %v", err)
		} else {
			defer file.Close()
			encoder := gob.NewEncoder(file)
			if err := encoder.Encode(sessions); err != nil {
				log.Printf("Failed to encode sessions: %v", err)
				// If encoding fails, remove the potentially corrupted file
				file.Close()
				os.Remove(SessionsFile)
			}
		}
	}

	// Save articles
	if articles != nil {
		file, err := os.Create(ArticlesFile)
		if err != nil {
			log.Printf("Failed to create articles file: %v", err)
		} else {
			defer file.Close()
			encoder := gob.NewEncoder(file)
			if err := encoder.Encode(articles); err != nil {
				log.Printf("Failed to encode articles: %v", err)
				// If encoding fails, remove the potentially corrupted file
				file.Close()
				os.Remove(ArticlesFile)
			}
		}
	}

	// Save reading stats
	if readingStats != nil {
		file, err := os.Create(ReadingStatsFile)
		if err != nil {
			log.Printf("Failed to create reading stats file: %v", err)
		} else {
			defer file.Close()
			encoder := gob.NewEncoder(file)
			if err := encoder.Encode(readingStats); err != nil {
				log.Printf("Failed to encode reading stats: %v", err)
				// If encoding fails, remove the potentially corrupted file
				file.Close()
				os.Remove(ReadingStatsFile)
			}
		}
	}
}

// GetExistSession initializes and authenticates with Exist.io
func GetExistSession(sessions *Sessions, client *http.Client) (*existio_client.OAuth2, error) {
	auth := existio_client.NewOAuth2(
		ExistOAuth2Return,
		ExistClientID,
		ExistClientSecret,
		"media_write",
		client,
	)

	if existData, ok := sessions.Exist["refresh_token"]; ok {
		auth.RefreshToken = existData.(string)
	}
	if existData, ok := sessions.Exist["access_token"]; ok {
		auth.AccessToken = existData.(string)
	}
	if existData, ok := sessions.Exist["refresh_lastdate"]; ok {
		auth.LastRefresh = existData.(time.Time)
	}

	if err := auth.EvaluateTokens(); err != nil {
		return nil, fmt.Errorf("failed to evaluate tokens: %v", err)
	}

	sessions.Exist["access_token"] = auth.AccessToken
	sessions.Exist["refresh_token"] = auth.RefreshToken
	sessions.Exist["refresh_lastdate"] = auth.LastRefresh

	SaveStates(sessions, nil, nil)
	return auth, nil
}

// GetExistAttrs initializes the Exist.io attributes client
func GetExistAttrs(sessions *Sessions, client *http.Client) (*existio_client.Attrs, error) {
	accessToken, ok := sessions.Exist["access_token"].(string)
	if !ok {
		return nil, fmt.Errorf("access token not found in sessions")
	}

	attrs := existio_client.NewAttrs(accessToken, 5*time.Second, client)
	if err := attrs.AcquireLabel("media", ExistAttributeName, existio_client.ValueTypeInteger, false); err != nil {
		return nil, fmt.Errorf("failed to acquire label: %v", err)
	}

	SaveStates(sessions, nil, nil)
	return attrs, nil
}

// Main function
func main() {
	// Parse command line arguments
	daysFlag := flag.Int("days", 3, "Number of days to consider for reading stats")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")
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
	sessions, articles, readingStats := LoadStates()

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

	log.Printf("Today's count = %d", readingStats[today])

	// Prepare data for submission
	var data []map[string]interface{}
	currentTime := time.Now()
	for i := 0; i < days; i++ {
		date := currentTime.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		count := readingStats[dateStr]
		data = append(data, attrs.FormatSubmission(date, ExistAttributeName, count))
	}

	// Submit data to Exist.io
	if err := attrs.UpdateBatch(data); err != nil {
		log.Fatalf("Failed to update batch: %v", err)
	}

	// Save states
	SaveStates(nil, &articles, &readingStats)
}
