package models

import (
	"testing"
	"time"
)

// TestIsAdultAt vérifie le calcul calendaire de la majorité (seuil 18 ans),
// notamment le cas de l'anniversaire pas encore atteint dans l'année.
func TestIsAdultAt(t *testing.T) {
	now := time.Date(2026, time.June, 19, 12, 0, 0, 0, time.UTC)
	bd := func(y int, m time.Month, d int) *time.Time {
		t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		return &t
	}

	cases := []struct {
		name  string
		birth *time.Time
		want  bool
	}{
		{"nil = adulte par défaut", nil, true},
		{"exactement 18 ans aujourd'hui", bd(2008, time.June, 19), true},
		{"18 ans demain (encore mineur)", bd(2008, time.June, 20), false},
		{"18 ans hier", bd(2008, time.June, 18), true},
		{"largement majeur", bd(1990, time.January, 1), true},
		{"17 ans", bd(2009, time.March, 3), false},
		{"anniversaire plus tard dans l'année", bd(2008, time.December, 31), false},
	}
	for _, c := range cases {
		if got := IsAdultAt(c.birth, now); got != c.want {
			t.Errorf("%s : IsAdultAt = %v, attendu %v", c.name, got, c.want)
		}
	}
}

// TestNsfwEnabledOf : absent (nil) ⇒ true (défaut ON, prod inchangée).
func TestNsfwEnabledOf(t *testing.T) {
	if !NsfwEnabledOf(&Profil{}) {
		t.Error("préférence absente devrait valoir true (défaut ON)")
	}
	off := false
	if NsfwEnabledOf(&Profil{NsfwEnabled: &off}) {
		t.Error("préférence false devrait valoir false")
	}
}

// TestHydrateViewerPolicy : nsfw_visible = is_adult && nsfw_enabled. Un mineur
// reste filtré même si la préférence stockée vaut true.
func TestHydrateViewerPolicy(t *testing.T) {
	now := time.Date(2026, time.June, 19, 0, 0, 0, 0, time.UTC)
	minorBirth := time.Date(2015, time.June, 19, 0, 0, 0, 0, time.UTC)
	on := true

	minor := &Profil{BirthDate: &minorBirth, NsfwEnabled: &on}
	HydrateViewerPolicy(minor, now)
	if minor.IsAdult || minor.NsfwVisible {
		t.Errorf("mineur : is_adult=%v nsfw_visible=%v, attendu false/false", minor.IsAdult, minor.NsfwVisible)
	}

	adultBirth := time.Date(2000, time.June, 19, 0, 0, 0, 0, time.UTC)
	adult := &Profil{BirthDate: &adultBirth, NsfwEnabled: &on}
	HydrateViewerPolicy(adult, now)
	if !adult.IsAdult || !adult.NsfwVisible {
		t.Errorf("majeur+ON : is_adult=%v nsfw_visible=%v, attendu true/true", adult.IsAdult, adult.NsfwVisible)
	}
}
