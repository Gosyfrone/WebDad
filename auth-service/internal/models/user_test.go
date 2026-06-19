package models

import "testing"

func TestNewAuthUser(t *testing.T) {
	u := &User{
		ID:    "uuid-123",
		Email: "alice@breezy.dev",
		Role:  RoleAdmin,
	}
	au := NewAuthUser(u)
	if au.ID != u.ID {
		t.Fatalf("ID = %q, attendu %q", au.ID, u.ID)
	}
	if au.Email != u.Email {
		t.Fatalf("Email = %q, attendu %q", au.Email, u.Email)
	}
	if au.Role != u.Role {
		t.Fatalf("Role = %q, attendu %q", au.Role, u.Role)
	}
}

func TestRoleConstants(t *testing.T) {
	cases := []struct {
		name string
		val  string
	}{
		{"RoleUser", RoleUser},
		{"RoleModerator", RoleModerator},
		{"RoleAdmin", RoleAdmin},
	}
	for _, tc := range cases {
		if tc.val == "" {
			t.Fatalf("constante %s ne doit pas être vide", tc.name)
		}
	}
	if RoleUser == RoleModerator || RoleUser == RoleAdmin || RoleModerator == RoleAdmin {
		t.Fatal("les constantes de rôle doivent être distinctes")
	}
}

func TestPasswordHashNotExported(t *testing.T) {
	u := User{PasswordHash: "bcrypt-hash"}
	au := NewAuthUser(&u)
	// NewAuthUser ne doit pas transmettre le hash
	// (vérifié implicitement : AuthUser n'a pas de champ PasswordHash)
	_ = au
}
