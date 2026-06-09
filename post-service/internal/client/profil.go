package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

type ProfilClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewProfilClient(baseURL string) *ProfilClient {
	return &ProfilClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

type visibilityResponse struct {
	Visibility string `json:"visibility"`
}

func (c *ProfilClient) Visibility(ctx context.Context, userID string) (string, error) {
	endpoint := fmt.Sprintf("%s/profils/%s/visibility", c.baseURL, url.PathEscape(userID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return VisibilityPublic, nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("profil-service visibility status %d", resp.StatusCode)
	}

	var result visibilityResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Visibility == "" {
		return VisibilityPublic, nil
	}
	return result.Visibility, nil
}
