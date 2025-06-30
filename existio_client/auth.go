package existio_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ExistOAuthEndpoint    = "https://exist.io/oauth2/"
	ExistOAuthRefreshDays = 30
)

// OAuth2 handles authentication with Exist.io
type OAuth2 struct {
	ReturnURL     string
	ClientID      string
	ClientSecret  string
	APIScope      string
	AccessToken   string
	RefreshToken  string
	LastRefresh   time.Time
	Client        *http.Client
	Server        *http.Server
	AuthCompleted chan bool
}

// NewOAuth2 creates a new OAuth2 instance
func NewOAuth2(returnURL, clientID, clientSecret, apiScope string, client *http.Client) *OAuth2 {
	if client == nil {
		client = StartSession()
	}
	return &OAuth2{
		ReturnURL:     returnURL,
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		APIScope:      apiScope,
		Client:        client,
		AuthCompleted: make(chan bool),
	}
}

// Authorize initiates the OAuth2 authorization flow
func (o *OAuth2) Authorize() error {
	queryParams := url.Values{
		"client_id":     {o.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {o.ReturnURL},
		"scope":         {o.APIScope},
	}

	authURL := fmt.Sprintf("%sauthorize?%s", ExistOAuthEndpoint, queryParams.Encode())
	fmt.Println("===Login to Exist===")
	fmt.Println("On this device, open the following address in your web browser:")
	fmt.Println(authURL)
	fmt.Println("")

	return o.AwaitExistOAuth2Tokens()
}

// AwaitExistOAuth2Tokens starts a local server to receive the OAuth2 callback
func (o *OAuth2) AwaitExistOAuth2Tokens() error {
	serverURL, err := url.Parse(o.ReturnURL)
	if err != nil {
		return fmt.Errorf("invalid return URL: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Code not found", http.StatusBadRequest)
			o.AuthCompleted <- false
			return
		}

		w.Write([]byte("OK!\n"))
		go func() {
			err := o.GetToken(code)
			o.AuthCompleted <- (err == nil)
		}()
	})

	o.Server = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", serverURL.Hostname(), serverURL.Port()),
		Handler: mux,
	}

	go func() {
		if err := o.Server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	result := <-o.AuthCompleted
	o.Server.Close()

	if !result {
		return fmt.Errorf("authorization failed")
	}
	return nil
}

// GetToken exchanges the authorization code for access and refresh tokens
func (o *OAuth2) GetToken(code string) error {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {o.ClientID},
		"client_secret": {o.ClientSecret},
		"redirect_uri":  {o.ReturnURL},
	}

	resp, err := o.Client.Post(
		fmt.Sprintf("%saccess_token", ExistOAuthEndpoint),
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	if tokenResp.Error != "" {
		return fmt.Errorf("Exist oAuth2: %s", tokenResp.Error)
	}

	o.AccessToken = tokenResp.AccessToken
	o.RefreshToken = tokenResp.RefreshToken
	o.LastRefresh = time.Now()
	return nil
}

// RefreshTokens refreshes the access and refresh tokens
func (o *OAuth2) RefreshTokens() error {
	if o.RefreshToken == "" {
		return o.Authorize()
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {o.RefreshToken},
		"client_id":     {o.ClientID},
		"client_secret": {o.ClientSecret},
	}

	resp, err := o.Client.Post(
		fmt.Sprintf("%saccess_token", ExistOAuthEndpoint),
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	if tokenResp.Error != "" {
		return fmt.Errorf("Exist oAuth2: Token Refresh Error: %s", tokenResp.Error)
	}

	o.AccessToken = tokenResp.AccessToken
	o.RefreshToken = tokenResp.RefreshToken
	o.LastRefresh = time.Now()
	return nil
}

// EvaluateTokens checks if tokens need to be refreshed
func (o *OAuth2) EvaluateTokens() error {
	if o.RefreshToken == "" {
		return o.Authorize()
	}

	aMonthAgo := time.Now().AddDate(0, 0, -ExistOAuthRefreshDays)
	if o.LastRefresh.IsZero() || o.LastRefresh.Before(aMonthAgo) {
		return o.RefreshTokens()
	}
	return nil
}
