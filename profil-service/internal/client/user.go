package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// UserClient lit le graphe social depuis user-service via l'API gateway.
type UserClient struct {
	baseURL string
	http    *http.Client
}

func NewUserClient(baseURL string) *UserClient {
	return &UserClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *UserClient) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	url := fmt.Sprintf("%s/internal/follows/%s/is-following/%s", c.baseURL, followerID, followingID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("user-service is-following status %d", resp.StatusCode)
	}
	var payload struct {
		IsFollowing bool `json:"isFollowing"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}
	return payload.IsFollowing, nil
}
