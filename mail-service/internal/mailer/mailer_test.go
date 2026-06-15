package mailer

import (
	"errors"
	"testing"
)

func TestConsoleMailerReportsDeliveryUnavailable(t *testing.T) {
	m := &consoleMailer{}
	err := m.Send(Message{
		To:      "test@example.com",
		Subject: "Test",
		Text:    "Message local",
	})
	if !errors.Is(err, ErrDeliveryUnavailable) {
		t.Fatalf("Send() error = %v, want ErrDeliveryUnavailable", err)
	}
}
