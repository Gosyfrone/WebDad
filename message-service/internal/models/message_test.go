package models

import "testing"

func TestRoleConstants(t *testing.T) {
	if RoleUser != "user" {
		t.Fatalf("RoleUser = %q", RoleUser)
	}
	if RoleModerator != "moderator" {
		t.Fatalf("RoleModerator = %q", RoleModerator)
	}
	if RoleAdmin != "admin" {
		t.Fatalf("RoleAdmin = %q", RoleAdmin)
	}
}

func TestConversationTypeConstants(t *testing.T) {
	if TypeDM != "dm" {
		t.Fatalf("TypeDM = %q", TypeDM)
	}
	if TypeGroup != "group" {
		t.Fatalf("TypeGroup = %q", TypeGroup)
	}
	if TypeCommunity != "community" {
		t.Fatalf("TypeCommunity = %q", TypeCommunity)
	}
}

func TestMemberRoleConstants(t *testing.T) {
	if MemberOwner != "owner" {
		t.Fatalf("MemberOwner = %q", MemberOwner)
	}
	if MemberAdmin != "admin" {
		t.Fatalf("MemberAdmin = %q", MemberAdmin)
	}
	if MemberTalker != "talker" {
		t.Fatalf("MemberTalker = %q", MemberTalker)
	}
	if MemberViewer != "viewer" {
		t.Fatalf("MemberViewer = %q", MemberViewer)
	}
}
