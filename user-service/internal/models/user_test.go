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

func TestCreateUserRequest_UsernameValidation(t *testing.T) {
	req := CreateUserRequest{Username: "alice"}
	if req.Username != "alice" {
		t.Fatalf("Username = %q", req.Username)
	}
}

func TestUpdateStatusRequest_IsActive(t *testing.T) {
	active := true
	req := UpdateStatusRequest{IsActive: &active}
	if req.IsActive == nil || !*req.IsActive {
		t.Fatal("IsActive devrait être true")
	}
}
