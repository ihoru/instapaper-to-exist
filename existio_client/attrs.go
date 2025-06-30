package existio_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	ExistAPIEndpoint = "https://exist.io/api/2/attributes/"
	ValueTypeInteger = 0
)

// Attrs handles attribute operations with the Exist.io API
type Attrs struct {
	AccessToken string
	Timeout     time.Duration
	Client      *http.Client
}

// NewAttrs creates a new Attrs instance
func NewAttrs(accessToken string, timeout time.Duration, client *http.Client) *Attrs {
	if client == nil {
		client = TimeoutClient(timeout)
	}
	return &Attrs{
		AccessToken: accessToken,
		Timeout:     timeout,
		Client:      client,
	}
}

// LabelToAttr converts a label to an attribute name
func (a *Attrs) LabelToAttr(label string) string {
	return strings.ToLower(strings.ReplaceAll(label, " ", "_"))
}

// CreateLabel creates a new attribute label
func (a *Attrs) CreateLabel(group, label string, valueType int, manual bool) error {
	type createRequest struct {
		Group     string `json:"group"`
		Label     string `json:"label"`
		ValueType int    `json:"value_type"`
		Manual    bool   `json:"manual"`
	}

	reqData := []createRequest{
		{
			Group:     group,
			Label:     label,
			ValueType: valueType,
			Manual:    manual,
		},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%screate/", ExistAPIEndpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	q.Add("success_objects", "1")
	req.URL.RawQuery = q.Encode()

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("failed to decode error response: %v", err)
		}
		return fmt.Errorf("Exist API: Create Attribute: %v", errResp)
	}

	return nil
}

// AcquireLabel acquires an attribute label
func (a *Attrs) AcquireLabel(group, label string, valueType int, manual bool) error {
	type acquireRequest struct {
		Name   string `json:"name"`
		Manual bool   `json:"manual"`
	}

	reqData := []acquireRequest{
		{
			Name:   a.LabelToAttr(label),
			Manual: manual,
		},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%sacquire/", ExistAPIEndpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var errResp struct {
		Failed []struct {
			ErrorCode string `json:"error_code"`
		} `json:"failed"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("failed to decode error response: %v", err)
	}

	if len(errResp.Failed) > 0 && errResp.Failed[0].ErrorCode == "not_found" {
		return a.CreateLabel(group, label, valueType, manual)
	}

	return fmt.Errorf("Exist API: Attribute Acquisition: %v", errResp)
}

// AcquireTemplate acquires a template
func (a *Attrs) AcquireTemplate(template string, manual bool) error {
	type acquireRequest struct {
		Template string `json:"template"`
		Manual   bool   `json:"manual"`
	}

	reqData := []acquireRequest{
		{
			Template: template,
			Manual:   manual,
		},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%sacquire/", ExistAPIEndpoint), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var errResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("failed to decode error response: %v", err)
	}

	return fmt.Errorf("Exist API: Template Acquisition: %v", errResp)
}

// ChunkSubmissions splits an array into chunks of the specified size
func (a *Attrs) ChunkSubmissions(arr []map[string]interface{}, size int) [][]map[string]interface{} {
	var chunks [][]map[string]interface{}
	for i := 0; i < len(arr); i += size {
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		chunks = append(chunks, arr[i:end])
	}
	return chunks
}

// UpdateBatch updates a batch of attributes
func (a *Attrs) UpdateBatch(data []map[string]interface{}) error {
	for _, chunk := range a.ChunkSubmissions(data, 20) {
		jsonData, err := json.Marshal(chunk)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("POST", fmt.Sprintf("%supdate/", ExistAPIEndpoint), bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.AccessToken))
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			continue
		} else if resp.StatusCode == http.StatusAccepted {
			var errResp map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
				return fmt.Errorf("failed to decode error response: %v", err)
			}
			return fmt.Errorf("Exist API: Submission: Some failed to update. %v", errResp)
		}

		var errResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("failed to decode error response: %v", err)
		}
		return fmt.Errorf("Exist API: Submission: %v", errResp)
	}

	return nil
}

// FormatSubmission formats a submission for the Exist.io API
func (a *Attrs) FormatSubmission(date time.Time, name string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"date":  date.Format("2006-01-02"),
		"name":  a.LabelToAttr(name),
		"value": value,
	}
}

// UpdateLabel updates a single attribute
func (a *Attrs) UpdateLabel(date time.Time, name string, value interface{}) error {
	return a.UpdateBatch([]map[string]interface{}{a.FormatSubmission(date, name, value)})
}
