package models

import "testing"

// ─── constantes ──────────────────────────────────────────────────────────────

func TestRoleConstants(t *testing.T) {
	if RoleUser != "user" {
		t.Fatalf("RoleUser = %q, attendu 'user'", RoleUser)
	}
	if RoleModerator != "moderator" {
		t.Fatalf("RoleModerator = %q, attendu 'moderator'", RoleModerator)
	}
	if RoleAdmin != "admin" {
		t.Fatalf("RoleAdmin = %q, attendu 'admin'", RoleAdmin)
	}
}

func TestAudienceConstants(t *testing.T) {
	if PollAudienceEveryone != "everyone" {
		t.Fatalf("PollAudienceEveryone = %q", PollAudienceEveryone)
	}
	if PollAudienceFollowers != "followers" {
		t.Fatalf("PollAudienceFollowers = %q", PollAudienceFollowers)
	}
	if ReplyAudienceEveryone != "everyone" {
		t.Fatalf("ReplyAudienceEveryone = %q", ReplyAudienceEveryone)
	}
	if ReplyAudienceFollowers != "followers" {
		t.Fatalf("ReplyAudienceFollowers = %q", ReplyAudienceFollowers)
	}
}

// ─── ReplyAudienceOf ─────────────────────────────────────────────────────────

func TestReplyAudienceOf_Nil(t *testing.T) {
	if got := ReplyAudienceOf(nil); got != ReplyAudienceEveryone {
		t.Fatalf("nil = %q, attendu 'everyone'", got)
	}
}

func TestReplyAudienceOf_Vide(t *testing.T) {
	p := &Post{ReplyAudience: ""}
	if got := ReplyAudienceOf(p); got != ReplyAudienceEveryone {
		t.Fatalf("vide = %q, attendu 'everyone'", got)
	}
}

func TestReplyAudienceOf_Everyone(t *testing.T) {
	p := &Post{ReplyAudience: ReplyAudienceEveryone}
	if got := ReplyAudienceOf(p); got != ReplyAudienceEveryone {
		t.Fatalf("everyone = %q, attendu 'everyone'", got)
	}
}

func TestReplyAudienceOf_Followers(t *testing.T) {
	p := &Post{ReplyAudience: ReplyAudienceFollowers}
	if got := ReplyAudienceOf(p); got != ReplyAudienceFollowers {
		t.Fatalf("followers = %q, attendu 'followers'", got)
	}
}
