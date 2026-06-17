package service

import (
	"errors"
	"testing"
	"time"

	"github.com/webdad/profil-service/internal/models"
)

func ptr[T any](v T) *T { return &v }

// TestPlanUpdate_DisplayNameChange : un changement effectif de display_name
// pose la valeur ET display_name_changed_at ; une valeur identique = no-op.
func TestPlanUpdate_DisplayNameChange(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	current := &models.Profil{DisplayName: "bob"}

	set, err := planUpdate(current, models.UpdateProfilRequest{DisplayName: ptr("bobby")}, now, 0)
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if set["display_name"] != "bobby" {
		t.Fatalf("display_name = %v, attendu bobby", set["display_name"])
	}
	if set["display_name_changed_at"] != now {
		t.Fatalf("display_name_changed_at = %v, attendu %v", set["display_name_changed_at"], now)
	}

	// Valeur identique → ni display_name ni timestamp dans le set.
	set, err = planUpdate(current, models.UpdateProfilRequest{DisplayName: ptr("bob")}, now, 0)
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if _, ok := set["display_name"]; ok {
		t.Fatal("display_name ne devrait pas être dans le set (valeur identique)")
	}
	if _, ok := set["display_name_changed_at"]; ok {
		t.Fatal("display_name_changed_at ne devrait pas être posé (no-op)")
	}
}

func TestValidDisplayName(t *testing.T) {
	valid := []string{"Jean Dupont", "Élodie_75", "Anne-Marie", "山田 太郎", "Мария-2", "Jean.Dupont", "J.R.R. Tolkien", "Jr."}
	for _, name := range valid {
		if !validDisplayName(name) {
			t.Errorf("nom valide refusé : %q", name)
		}
	}

	invalid := []string{"", "Jean@Dupont", "#Jean", "O'Connor", "Jean 😊", "Jean\tDupont"}
	for _, name := range invalid {
		if validDisplayName(name) {
			t.Errorf("nom invalide accepté : %q", name)
		}
	}
}

func TestPlanUpdate_DisplayNameValidationPreservesLegacyNames(t *testing.T) {
	now := time.Now().UTC()
	current := &models.Profil{DisplayName: "O'Connor"}

	set, err := planUpdate(current, models.UpdateProfilRequest{
		DisplayName: ptr("O'Connor"),
		Bio:         ptr("Nouvelle bio"),
	}, now, 0)
	if err != nil {
		t.Fatalf("nom legacy inchangé refusé : %v", err)
	}
	if _, ok := set["display_name"]; ok {
		t.Fatal("le nom legacy inchangé ne doit pas être réécrit")
	}
	if set["bio"] != "Nouvelle bio" {
		t.Fatalf("la bio doit rester modifiable, obtenu %v", set["bio"])
	}

	if _, err := planUpdate(current, models.UpdateProfilRequest{DisplayName: ptr("Jean@Dupont")}, now, 0); !errors.Is(err, ErrInvalidDisplayName) {
		t.Fatalf("attendu ErrInvalidDisplayName, obtenu %v", err)
	}
}

// TestPlanUpdate_Cooldown : avec un cooldown actif, un changement trop proche
// du précédent est refusé ; passé le délai il est autorisé ; cooldown=0 ne
// refuse jamais.
func TestPlanUpdate_Cooldown(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	cooldown := 168 * time.Hour // 7 jours
	changed := now.Add(-24 * time.Hour)
	current := &models.Profil{DisplayName: "bob", DisplayNameChangedAt: &changed}
	req := models.UpdateProfilRequest{DisplayName: ptr("bobby")}

	if _, err := planUpdate(current, req, now, cooldown); !errors.Is(err, ErrDisplayNameCooldown) {
		t.Fatalf("attendu ErrDisplayNameCooldown, obtenu %v", err)
	}

	// Hors fenêtre de cooldown → autorisé.
	old := now.Add(-8 * 24 * time.Hour)
	current.DisplayNameChangedAt = &old
	if _, err := planUpdate(current, req, now, cooldown); err != nil {
		t.Fatalf("changement hors cooldown refusé : %v", err)
	}

	// Cooldown désactivé (0) → jamais refusé même juste après un changement.
	current.DisplayNameChangedAt = &changed
	if _, err := planUpdate(current, req, now, 0); err != nil {
		t.Fatalf("cooldown=0 ne doit jamais refuser : %v", err)
	}
}

// TestPlanUpdate_BirthDateSetOnce : settable tant que vide ; une fois posée,
// la rejouer à l'identique est tolérée mais la changer est refusée.
func TestPlanUpdate_BirthDateSetOnce(t *testing.T) {
	now := time.Now().UTC()
	bd := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	// Vide → premier renseignement autorisé.
	set, err := planUpdate(&models.Profil{}, models.UpdateProfilRequest{BirthDate: &bd}, now, 0)
	if err != nil {
		t.Fatalf("1er renseignement refusé : %v", err)
	}
	if set["birth_date"] != bd {
		t.Fatalf("birth_date = %v, attendu %v", set["birth_date"], bd)
	}

	current := &models.Profil{BirthDate: &bd}

	// Même valeur → toléré (no-op, pas dans le set).
	set, err = planUpdate(current, models.UpdateProfilRequest{BirthDate: &bd}, now, 0)
	if err != nil {
		t.Fatalf("rejeu identique refusé : %v", err)
	}
	if _, ok := set["birth_date"]; ok {
		t.Fatal("birth_date ne devrait pas être réécrit (valeur identique)")
	}

	// Valeur différente → refus (set-once).
	other := time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC)
	if _, err := planUpdate(current, models.UpdateProfilRequest{BirthDate: &other}, now, 0); !errors.Is(err, ErrBirthDateLocked) {
		t.Fatalf("attendu ErrBirthDateLocked, obtenu %v", err)
	}
}

func TestPlanUpdate_Visibility(t *testing.T) {
	now := time.Now().UTC()
	current := &models.Profil{Visibility: models.VisibilityPublic, ActivityVisibility: models.VisibilityPublic}

	set, err := planUpdate(current, models.UpdateProfilRequest{Visibility: ptr(models.VisibilityPrivate)}, now, 0)
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if set["visibility"] != models.VisibilityPrivate {
		t.Fatalf("visibility = %v, attendu %s", set["visibility"], models.VisibilityPrivate)
	}
	if _, ok := set["activity_visibility"]; ok {
		t.Fatal("passer le profil en privé ne doit pas modifier la préférence d'activité")
	}

	set, err = planUpdate(current, models.UpdateProfilRequest{Visibility: ptr(models.VisibilityPublic)}, now, 0)
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if _, ok := set["visibility"]; ok {
		t.Fatal("visibility ne devrait pas être réécrite (valeur identique)")
	}
}

func TestPlanUpdate_ActivityVisibility(t *testing.T) {
	now := time.Now().UTC()
	current := &models.Profil{Visibility: models.VisibilityPublic, ActivityVisibility: models.VisibilityPublic}

	set, err := planUpdate(current, models.UpdateProfilRequest{ActivityVisibility: ptr(models.VisibilityPrivate)}, now, 0)
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if set["activity_visibility"] != models.VisibilityPrivate {
		t.Fatalf("activity_visibility = %v, attendu %s", set["activity_visibility"], models.VisibilityPrivate)
	}

	current = &models.Profil{Visibility: models.VisibilityPrivate, ActivityVisibility: models.VisibilityPrivate}
	set, err = planUpdate(current, models.UpdateProfilRequest{ActivityVisibility: ptr(models.VisibilityPublic)}, now, 0)
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if set["activity_visibility"] != models.VisibilityPublic {
		t.Fatalf("activity_visibility = %v, attendu %s", set["activity_visibility"], models.VisibilityPublic)
	}
}

func TestPlanUpdate_Nationality(t *testing.T) {
	now := time.Now().UTC()
	set, err := planUpdate(&models.Profil{}, models.UpdateProfilRequest{Nationality: ptr("FR")}, now, 0)
	if err != nil {
		t.Fatalf("err inattendue : %v", err)
	}
	if set["nationality"] != "FR" {
		t.Fatalf("nationality = %v, attendu FR", set["nationality"])
	}
}
