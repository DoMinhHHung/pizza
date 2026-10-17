package authclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type lookupResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
}

// Lookup resolves username or email to user ID via auth-service.
// GET {baseURL}/api/v1/users/lookup?q=...
func (c *Client) Lookup(ctx context.Context, bearerToken, q string) (userID string, err error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", fmt.Errorf("empty lookup query")
	}
	if c.baseURL == "" {
		return "", fmt.Errorf("auth service URL not configured")
	}

	u, err := url.Parse(c.baseURL + "/api/v1/users/lookup")
	if err != nil {
		return "", fmt.Errorf("parse auth url: %w", err)
	}
	qs := u.Query()
	qs.Set("q", q)
	u.RawQuery = qs.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	token := strings.TrimSpace(bearerToken)
	if token != "" {
		if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = "Bearer " + token
		}
		req.Header.Set("Authorization", token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth lookup request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("user not found")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("unauthorized calling auth service")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth lookup status %d", resp.StatusCode)
	}

	var body lookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decode lookup response: %w", err)
	}
	if body.ID == "" {
		return "", fmt.Errorf("auth lookup returned empty id")
	}
	return body.ID, nil
}
