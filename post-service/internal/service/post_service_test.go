package service

import (
	"errors"
	"testing"

	"github.com/webdad/post-service/internal/models"
)

// TestCanModify : un post n'est modifiable que par son auteur, un modérateur
// ou un administrateur. Fonction PURE.
func TestCanModify(t *testing.T) {
	post := &models.Post{AuthorID: "author-1"}

	cases := []struct {
		name      string
		actorID   string
		actorRole string
		want      bool
	}{
		{"auteur", "author-1", models.RoleUser, true},
		{"autre utilisateur", "author-2", models.RoleUser, false},
		{"modérateur tiers", "mod-9", models.RoleModerator, true},
		{"admin tiers", "admin-9", models.RoleAdmin, true},
		{"non-auteur sans rôle", "x", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canModify(post, tc.actorID, tc.actorRole); got != tc.want {
				t.Fatalf("canModify(%s,%s) = %v, attendu %v", tc.actorID, tc.actorRole, got, tc.want)
			}
		})
	}
}

// TestCanModifyNilPost : un post nil n'est jamais modifiable (pas de panic).
func TestCanModifyNilPost(t *testing.T) {
	if canModify(nil, "anyone", models.RoleAdmin) {
		t.Fatal("canModify(nil, …) doit être false")
	}
}

// TestParseID : un hex valide passe, le reste donne ErrInvalidID.
func TestParseID(t *testing.T) {
	if _, err := parseID("507f1f77bcf86cd799439011"); err != nil {
		t.Fatalf("ObjectID valide refusé : %v", err)
	}
	for _, bad := range []string{"", "xyz", "123"} {
		if _, err := parseID(bad); !errors.Is(err, ErrInvalidID) {
			t.Fatalf("parseID(%q) = %v, attendu ErrInvalidID", bad, err)
		}
	}
}

// TestClampLimit : défaut quand <=0, plafond à MaxLimit, valeur valide gardée.
func TestClampLimit(t *testing.T) {
	cases := []struct{ in, want int64 }{
		{0, DefaultLimit},
		{-5, DefaultLimit},
		{10, 10},
		{MaxLimit, MaxLimit},
		{MaxLimit + 1, MaxLimit},
	}
	for _, tc := range cases {
		if got := clampLimit(tc.in); got != tc.want {
			t.Fatalf("clampLimit(%d) = %d, attendu %d", tc.in, got, tc.want)
		}
	}
}

// TestClampOffset : un offset négatif est ramené à 0.
func TestClampOffset(t *testing.T) {
	if got := clampOffset(-1); got != 0 {
		t.Fatalf("clampOffset(-1) = %d, attendu 0", got)
	}
	if got := clampOffset(5); got != 5 {
		t.Fatalf("clampOffset(5) = %d, attendu 5", got)
	}
}
