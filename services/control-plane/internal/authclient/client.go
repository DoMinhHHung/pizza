package authclient

import (
	"context"
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func (c *Client) Lookup(ctx context.Context, bearerToken, q string) (userID string, err error)
