package remote_repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/avantifellows/nex-gen-cms/config"
)

// APIRepository interacts with a remote API
type APIRepository struct{}

// NewAPIRepository creates a new api repository
func NewAPIRepository() *APIRepository {
	return &APIRepository{}
}

func (r *APIRepository) CallAPI(urlEndPoint string, method string, body any) ([]byte, error) {
	// Create an HTTP client
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	var reqBody io.Reader
	if body != nil {
		// Check if body is already in byte[] form
		jsonBodyBytes, ok := body.([]byte)
		// if not, then convert it to byte[]
		if !ok {
			var err error
			jsonBodyBytes, err = json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("error marshalling request body: %v", err)
			}
		}

		reqBody = bytes.NewBuffer(jsonBodyBytes)
	}

	// Build a request url
	apiUrl := config.GetEnv("DB_SERVICE_ENDPOINT", "") + urlEndPoint

	req, err := http.NewRequest(method, apiUrl, reqBody)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add headers if needed (e.g., Authorization, Content-Type)
	bearerToken := config.GetEnv("DB_SERVICE_TOKEN", "")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	return bodyBytes, nil
}
