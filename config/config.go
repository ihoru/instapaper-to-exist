package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
)

// Config holds all environment settings for the application
type Config struct {
	ExistClientID        string
	ExistClientSecret    string
	ExistOAuth2Return    string
	ExistAttributeName   string
	InstapaperArchiveRSS string
}

// LoadConfig loads configuration from environment variables or .env file
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file")
	}

	config := &Config{
		ExistClientID:        os.Getenv("EXIST_CLIENT_ID"),
		ExistClientSecret:    os.Getenv("EXIST_CLIENT_SECRET"),
		ExistOAuth2Return:    os.Getenv("EXIST_OAUTH2_RETURN"),
		ExistAttributeName:   os.Getenv("EXIST_ATTRIBUTE_NAME"),
		InstapaperArchiveRSS: os.Getenv("INSTAPAPER_ARCHIVE_RSS"),
	}

	// Set default values
	if config.ExistOAuth2Return == "" {
		config.ExistOAuth2Return = "http://localhost:9009/"
	}
	if config.ExistAttributeName == "" {
		config.ExistAttributeName = "Articles read"
	}

	// Validate required fields
	var missingVars []string
	if config.ExistClientID == "" {
		missingVars = append(missingVars, "EXIST_CLIENT_ID")
	}
	if config.ExistClientSecret == "" {
		missingVars = append(missingVars, "EXIST_CLIENT_SECRET")
	}
	if config.InstapaperArchiveRSS == "" {
		missingVars = append(missingVars, "INSTAPAPER_ARCHIVE_RSS")
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("required environment variables are missing: %v", missingVars)
	}

	return config, nil
}

// PrintMissingVarsHelp prints helpful information when required variables are missing
//
//goland:noinspection GoUnhandledErrorResult
func PrintMissingVarsHelp() {
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
}
