package models

import "testing"

func TestTypeConstants_Follow(t *testing.T) {
	if TypeFollow != "follow" {
		t.Fatalf("TypeFollow = %q", TypeFollow)
	}
	if TypeFollowRequest != "follow_request" {
		t.Fatalf("TypeFollowRequest = %q", TypeFollowRequest)
	}
	if TypeFollowRequestAccepted != "follow_request_accepted" {
		t.Fatalf("TypeFollowRequestAccepted = %q", TypeFollowRequestAccepted)
	}
}

func TestTypeConstants_Post(t *testing.T) {
	if TypeLike != "like" {
		t.Fatalf("TypeLike = %q", TypeLike)
	}
	if TypeComment != "comment" {
		t.Fatalf("TypeComment = %q", TypeComment)
	}
	if TypeRepost != "repost" {
		t.Fatalf("TypeRepost = %q", TypeRepost)
	}
	if TypeMention != "mention" {
		t.Fatalf("TypeMention = %q", TypeMention)
	}
}

func TestTypeConstants_Message(t *testing.T) {
	if TypeMessage != "message" {
		t.Fatalf("TypeMessage = %q", TypeMessage)
	}
	if TypeMessageMention != "message_mention" {
		t.Fatalf("TypeMessageMention = %q", TypeMessageMention)
	}
}
