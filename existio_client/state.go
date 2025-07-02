package existio_client

import (
	"time"
)

// ExistAuth stores authentication data for Exist.io
type ExistAuth struct {
	AccessToken  string
	RefreshToken string
	LastRefresh  time.Time
}
