package state

import (
	"encoding/gob"
	"github.com/ihoru/instapaper-to-exist/existio_client"
	store "github.com/ihoru/instapaper-to-exist/storage"
	"time"
)

func init() {
	// Register types for gob encoding/decoding
	gob.Register(time.Time{})
}

// Sessions storage
type Sessions struct {
	Exist existio_client.ExistAuth
}

// ReadingStats maps dates to article counts
type ReadingStats map[string]int

// Articles is a set of article URLs
type Articles map[string]bool

// LoadStates loads the state files (for backward compatibility)
func LoadStates(storage *store.Storage) (Sessions, Articles, ReadingStats) {
	sessions := Sessions{
		Exist: existio_client.ExistAuth{},
	}
	articles := make(Articles)
	readingStats := make(ReadingStats)

	storage.Load("sessions", &sessions)
	storage.Load("articles", &articles)
	storage.Load("stats", &readingStats)

	return sessions, articles, readingStats
}

// SaveStates saves the state files (for backward compatibility)
func SaveStates(storage *store.Storage, sessions *Sessions, articles *Articles, readingStats *ReadingStats) {
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
