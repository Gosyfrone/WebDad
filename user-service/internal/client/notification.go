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

const (
	TypeFollow                     = "follow"
	TypeFollowRequest              = "follow_request"
	TypeFollowRequestAccepted      = "follow_request_accepted"
	TypeFollowRequestAcceptConfirm = "follow_request_accept_confirm"
	TypeFollowRequestRejected      = "follow_request_rejected"
)

type NotificationClient struct {
	url    string
	secret string
	client *http.Client
}

func NewNotificationClient(baseURL, secret string) *NotificationClient {
	return &NotificationClient{
		url:    baseURL + "/internal/events",
		secret: secret,
		client: &http.Client{Timeout: 4 * time.Second},
	}
}

type Event struct {
	Type        string `json:"type"`
	ActorID     string `json:"actor_id"`
	RecipientID string `json:"recipient_id"`
	Retract     bool   `json:"retract,omitempty"`
}

func (c *NotificationClient) Emit(ev Event) {
	if c == nil || c.url == "" {
		return
	}
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
			log.Printf("[notification] follow request event: %v", err)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			log.Printf("[notification] follow request event: status=%d body=%s", resp.StatusCode, string(msg))
		}
	}()
}
