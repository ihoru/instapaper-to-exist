package existio_client

import (
	"net/http"
	"time"
)

// TimeoutClient creates an HTTP client with a specified timeout
func TimeoutClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// StartSession creates an HTTP client with a default timeout of 10 seconds
func StartSession() *http.Client {
	return TimeoutClient(10 * time.Second)
}
