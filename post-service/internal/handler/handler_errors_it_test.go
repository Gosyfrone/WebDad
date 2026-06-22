package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Chemins d'erreur via le stack réel : couvre respondPostError (mapping des
// erreurs métier → codes HTTP) et les branches d'échec des handlers.

func TestIT_Errors_NotFound(t *testing.T) {
	r := newMongoRouter(t)
	tok := itToken(t, "alice", "user")
	missing := bson.NewObjectID().Hex()

	cases := []struct {
		name, method, path, body string
	}{
		{"GetPost", http.MethodGet, "/posts/" + missing, ""},
		{"LikePost", http.MethodPost, "/posts/" + missing + "/like", ""},
		{"RepostPost", http.MethodPost, "/posts/" + missing + "/repost", ""},
		{"VotePoll", http.MethodPost, "/posts/" + missing + "/poll/vote", `{"choice_id":"c1"}`},
		{"CommentOnMissing", http.MethodPost, "/posts/" + missing + "/comments", `{"content":"x"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := do(t, r, c.method, c.path, tok, c.body)
			if w.Code != http.StatusNotFound {
				t.Fatalf("%s = %d, attendu 404 (body=%s)", c.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestIT_Errors_InvalidID(t *testing.T) {
	r := newMongoRouter(t)
	tok := itToken(t, "alice", "user")
	if w := do(t, r, http.MethodGet, "/posts/not-a-hex", tok, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("GetPost(id invalide) = %d, attendu 400", w.Code)
	}
	if w := do(t, r, http.MethodPost, "/posts/not-a-hex/like", tok, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("LikePost(id invalide) = %d, attendu 400", w.Code)
	}
}

func TestIT_Errors_Forbidden(t *testing.T) {
	r := newMongoRouter(t)
	author := itToken(t, "alice", "user")
	intruder := itToken(t, "carol", "user")
	id := createPost(t, r, author, "post d'alice")

	// Un tiers (ni auteur ni mod) ne peut pas éditer/supprimer.
	if w := do(t, r, http.MethodPatch, "/posts/"+id, intruder, `{"content":"pirate #x"}`); w.Code != http.StatusForbidden {
		t.Fatalf("UpdatePost(tiers) = %d, attendu 403", w.Code)
	}
	// Restore/purge réservés aux mods → un user simple est rejeté par le middleware (403).
	if w := do(t, r, http.MethodPost, "/posts/"+id+"/restore", intruder, ""); w.Code != http.StatusForbidden {
		t.Fatalf("RestorePost(user) = %d, attendu 403", w.Code)
	}
}

func TestIT_Errors_ContentTooLong(t *testing.T) {
	r := newMongoRouter(t)
	tok := itToken(t, "alice", "user")
	long := strings.Repeat("a", 281)
	if w := do(t, r, http.MethodPost, "/posts", tok, fmt.Sprintf(`{"content":%q}`, long)); w.Code != http.StatusBadRequest {
		t.Fatalf("CreatePost(>280) = %d, attendu 400", w.Code)
	}
	// Un modérateur est exempté de la limite des 280 du handler (enforceContentLimit).
	// NB : le validateur Mongo plafonne content à 280 pour TOUS → l'insertion échoue
	// quand même en base (500). On vérifie seulement que le handler n'a pas opposé
	// son propre 400 (la branche d'exemption mod est bien empruntée).
	mod := itToken(t, "mod", "moderator")
	if w := do(t, r, http.MethodPost, "/posts", mod, fmt.Sprintf(`{"content":%q}`, long)); w.Code == http.StatusBadRequest {
		t.Fatalf("CreatePost(mod, >280) = 400 : la limite handler ne devrait pas s'appliquer aux mods")
	}
}

func TestIT_Errors_EmptyPost(t *testing.T) {
	r := newMongoRouter(t)
	tok := itToken(t, "alice", "user")
	if w := do(t, r, http.MethodPost, "/posts", tok, `{"content":"   "}`); w.Code != http.StatusBadRequest {
		t.Fatalf("CreatePost(vide) = %d, attendu 400", w.Code)
	}
	if w := do(t, r, http.MethodPost, "/posts", tok, `{bad json`); w.Code != http.StatusBadRequest {
		t.Fatalf("CreatePost(json invalide) = %d, attendu 400", w.Code)
	}
}

// ListCommentReplies (route oubliée du happy-path) : 200 sur un commentaire réel.
func TestIT_CommentReplies(t *testing.T) {
	r := newMongoRouter(t)
	author := itToken(t, "alice", "user")
	id := createPost(t, r, author, "post")
	w := do(t, r, http.MethodPost, "/posts/"+id+"/comments", author, `{"content":"racine"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateComment = %d", w.Code)
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	cid := resp.Data.ID
	if w := do(t, r, http.MethodGet, "/posts/"+id+"/comments/"+cid+"/replies", "", ""); !okCode(w.Code) {
		t.Fatalf("ListCommentReplies = %d", w.Code)
	}
}
