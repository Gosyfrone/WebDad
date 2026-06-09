package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type FollowClient struct {
	baseURL        string
	httpClient     *http.Client
	internalSecret string
}

func NewFollowClient(baseURL, secret string) *FollowClient {
	return &FollowClient{
		baseURL:        baseURL,
		httpClient:     &http.Client{Timeout: 3 * time.Second},
		internalSecret: secret,
	}
}

type followResponse struct {
	IsFollowing bool `json:"isFollowing"`
}

func (c *FollowClient) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	endpoint := fmt.Sprintf("%s/internal/%s/is-following/%s",
		c.baseURL, url.PathEscape(followerID), url.PathEscape(followingID),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("X-Internal-Secret", c.internalSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return false, fmt.Errorf("user-service follow status %d", resp.StatusCode)
	}

	var result followResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.IsFollowing, nil
}
