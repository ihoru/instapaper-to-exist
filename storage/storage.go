package storage

import (
	"encoding/gob"
	"log"
	"os"
	"path/filepath"
	"time"
)

func init() {
	// Register types for gob encoding/decoding
	gob.Register(time.Time{})
}

// Sessions storage
type Sessions struct {
	Exist map[string]interface{}
}

// ReadingStats maps dates to article counts
type ReadingStats map[string]int

// Articles is a set of article URLs
type Articles map[string]bool

// Storage handles persistent storage operations
type Storage struct {
	stateDir string
}

// NewStorage creates a new Storage instance
func NewStorage(appName string) *Storage {
	// Set up state directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}
	stateDir := filepath.Join(homeDir, ".local", "state", appName)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		log.Fatalf("Failed to create state directory: %v", err)
	}

	return &Storage{
		stateDir: stateDir,
	}
}

// Load loads data from a file using gob decoder
func (s *Storage) Load(fileName string, data interface{}) error {
	filePath := filepath.Join(s.stateDir, fileName)
	if _, err := os.Stat(filePath); err == nil {
		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("Failed to open file %s: %v", filePath, err)
			return err
		}
		defer file.Close()

		decoder := gob.NewDecoder(file)
		if err := decoder.Decode(data); err != nil {
			log.Printf("Failed to decode %s: %v", filePath, err)
			log.Printf("Removing corrupted file and creating a new one.")
			file.Close()        // Close the file before removing it
			os.Remove(filePath) // Remove the corrupted file
			return err
		}
	}
	return nil
}

// Save saves data to a file using gob encoder
func (s *Storage) Save(fileName string, data interface{}) error {
	filePath := filepath.Join(s.stateDir, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file %s: %v", filePath, err)
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		log.Printf("Failed to encode %s: %v", filePath, err)
		// If encoding fails, remove the potentially corrupted file
		file.Close()
		os.Remove(filePath)
		return err
	}
	return nil
}

// LoadStates loads the state files (for backward compatibility)
func LoadStates(storage *Storage) (Sessions, Articles, ReadingStats) {
	sessions := Sessions{
		Exist: make(map[string]interface{}),
	}
	articles := make(Articles)
	readingStats := make(ReadingStats)

	storage.Load("sessions", &sessions)
	storage.Load("articles", &articles)
	storage.Load("stats", &readingStats)

	return sessions, articles, readingStats
}

// SaveStates saves the state files (for backward compatibility)
func SaveStates(storage *Storage, sessions *Sessions, articles *Articles, readingStats *ReadingStats) {
	// Save sessions
	if sessions != nil {
		storage.Save("sessions", sessions)
	}

	// Save articles
	if articles != nil {
		storage.Save("articles", articles)
	}

	// Save reading stats
	if readingStats != nil {
		storage.Save("stats", readingStats)
	}
}
