package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

const TypeIdentityUpdated = "identity_updated"

type NotificationClient struct {
	url    string
	secret string
	client *http.Client
}

func NewNotificationClient(baseURL, secret string) *NotificationClient {
	if baseURL == "" {
		return nil
	}
	return &NotificationClient{
		url:    baseURL + "/internal/events",
		secret: secret,
		client: &http.Client{Timeout: 4 * time.Second},
	}
}

type IdentityEvent struct {
	Type          string `json:"type"`
	ActorID       string `json:"actor_id"`
	TargetUserID  string `json:"target_user_id"`
	Certification string `json:"certification,omitempty"`
	Role          string `json:"role,omitempty"`
}

func (c *NotificationClient) EmitIdentityUpdated(userID, certification, role string) {
	if c == nil || c.url == "" {
		return
	}
	c.emit(IdentityEvent{
		Type:          TypeIdentityUpdated,
		ActorID:       userID,
		TargetUserID:  userID,
		Certification: certification,
		Role:          role,
	})
}

func (c *NotificationClient) emit(ev IdentityEvent) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		body, err := json.Marshal(ev)
		if err != nil {
			return
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Secret", c.secret)
		resp, err := c.client.Do(req)
		if err != nil {
			log.Printf("[notification] identity event: %v", err)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			log.Printf("[notification] identity event: status=%d body=%s", resp.StatusCode, string(msg))
		}
	}()
}
