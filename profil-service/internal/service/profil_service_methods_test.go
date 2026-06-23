package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/profil-service/internal/models"
)

type fakeProfilRepo struct {
	profil       *models.Profil
	search       []models.Profil
	err          error
	insertErr    error
	updateErr    error
	deleteErr    error
	inserted     *models.Profil
	searchTerm   string
	searchLimit  int64
	updatedUser  string
	updateSet    bson.M
	updateCalled bool
}

func (f *fakeProfilRepo) GetByUserID(_ context.Context, _ string) (*models.Profil, error) {
	if f.err != nil {
		return nil, f.err
	}
	return cloneProfil(f.profil), nil
}

func (f *fakeProfilRepo) Insert(_ context.Context, p *models.Profil) error {
	f.inserted = cloneProfil(p)
	return f.insertErr
}

func (f *fakeProfilRepo) SearchByDisplayName(_ context.Context, pattern string, limit int64) ([]models.Profil, error) {
	f.searchTerm = pattern
	f.searchLimit = limit
	if f.err != nil {
		return nil, f.err
	}
	return f.search, nil
}

func (f *fakeProfilRepo) Update(_ context.Context, userID string, set bson.M) (*models.Profil, error) {
	f.updateCalled = true
	f.updatedUser = userID
	f.updateSet = set
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	next := cloneProfil(f.profil)
	if next == nil {
		next = &models.Profil{UserID: userID}
	}
	if v, ok := set["display_name"].(string); ok {
		next.DisplayName = v
	}
	if v, ok := set["visibility"].(string); ok {
		next.Visibility = v
	}
	if v, ok := set["certification"].(string); ok {
		next.Certification = v
	}
	if v, ok := set["is_online"].(bool); ok {
		next.IsOnline = v
	}
	if v, ok := set["last_login_at"].(time.Time); ok {
		next.LastLoginAt = &v
	}
	return next, nil
}

func (f *fakeProfilRepo) Delete(_ context.Context, _ string) error {
	return f.deleteErr
}

func cloneProfil(p *models.Profil) *models.Profil {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

type spyFollowChecker struct {
	acceptCalled bool
	acceptErr    error
}

func (s *spyFollowChecker) IsFollowing(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (s *spyFollowChecker) AcceptAllFollowRequests(_ context.Context, _ string) error {
	s.acceptCalled = true
	return s.acceptErr
}

func TestCreate_ValidProfilSetsDefaults(t *testing.T) {
	repo := &fakeProfilRepo{}
	svc := New(repo, 0)
	birthDate := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	gender := "female"

	got, err := svc.Create(context.Background(), "u1", models.CreateProfilRequest{
		DisplayName: " Alice ",
		BirthDate:   &birthDate,
		Gender:      &gender,
	})
	if err != nil {
		t.Fatalf("Create erreur inattendue: %v", err)
	}
	if got.DisplayName != "Alice" || repo.inserted.DisplayName != "Alice" {
		t.Fatalf("display_name doit etre trimme, got=%q inserted=%q", got.DisplayName, repo.inserted.DisplayName)
	}
	if got.Visibility != models.VisibilityPublic || got.LikesVisibility != models.VisibilityPublic || got.ActivityVisibility != models.VisibilityPublic {
		t.Fatalf("visibilites par defaut incorrectes: %#v", got)
	}
	if got.Certification != models.CertificationNone {
		t.Fatalf("certification par defaut = %q", got.Certification)
	}
	if got.NsfwEnabled == nil || !*got.NsfwEnabled {
		t.Fatal("nsfw_enabled doit etre pose a true par defaut")
	}
	if got.BirthDate == nil || !got.BirthDate.Equal(birthDate) || got.Gender != gender {
		t.Fatalf("birth_date/gender non conserves: %#v", got)
	}
}

func TestCreate_DuplicateMapsToProfilExists(t *testing.T) {
	repo := &fakeProfilRepo{insertErr: mongo.WriteException{
		WriteErrors: []mongo.WriteError{{Code: 11000}},
	}}
	svc := New(repo, 0)

	_, err := svc.Create(context.Background(), "u1", models.CreateProfilRequest{DisplayName: "Alice"})
	if !errors.Is(err, ErrProfilExists) {
		t.Fatalf("attendu ErrProfilExists, obtenu %v", err)
	}
}

func TestGetMine_NormalizesActivityAndHydratesPolicy(t *testing.T) {
	oldLogin := time.Now().UTC().Add(-2 * time.Minute)
	birthDate := time.Now().UTC().AddDate(-17, 0, 0)
	nsfwEnabled := true
	repo := &fakeProfilRepo{profil: &models.Profil{
		UserID:      "u1",
		BirthDate:   &birthDate,
		IsOnline:    true,
		LastLoginAt: &oldLogin,
		NsfwEnabled: &nsfwEnabled,
	}}
	svc := New(repo, 0)

	got, err := svc.GetMine(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetMine erreur inattendue: %v", err)
	}
	if got.IsOnline {
		t.Fatal("une activite trop ancienne doit passer hors ligne")
	}
	if got.IsAdult || got.NsfwVisible {
		t.Fatalf("un mineur ne doit pas voir le NSFW: is_adult=%v nsfw_visible=%v", got.IsAdult, got.NsfwVisible)
	}
}

func TestSearch_EscapesPatternAndNormalizesResults(t *testing.T) {
	oldLogin := time.Now().UTC().Add(-2 * time.Minute)
	repo := &fakeProfilRepo{search: []models.Profil{{
		UserID:      "u1",
		IsOnline:    true,
		LastLoginAt: &oldLogin,
	}}}
	svc := New(repo, 0)

	got, err := svc.Search(context.Background(), " a.b ", 12)
	if err != nil {
		t.Fatalf("Search erreur inattendue: %v", err)
	}
	if repo.searchTerm != `a\.b` || repo.searchLimit != 12 {
		t.Fatalf("pattern/limit incorrects: pattern=%q limit=%d", repo.searchTerm, repo.searchLimit)
	}
	if got[0].IsOnline {
		t.Fatal("Search doit normaliser l'activite des resultats")
	}
}

func TestUpdate_NoChangeSkipsRepositoryUpdate(t *testing.T) {
	repo := &fakeProfilRepo{profil: &models.Profil{UserID: "u1", DisplayName: "Alice"}}
	svc := New(repo, 0)

	got, err := svc.Update(context.Background(), "u1", models.UpdateProfilRequest{DisplayName: ptr("Alice")})
	if err != nil {
		t.Fatalf("Update erreur inattendue: %v", err)
	}
	if got.DisplayName != "Alice" {
		t.Fatalf("profil retourne incorrect: %#v", got)
	}
	if repo.updateCalled {
		t.Fatal("Update ne doit pas appeler le repository quand seul updated_at change")
	}
}

func TestUpdate_PrivateToPublicAcceptsPendingRequestsBestEffort(t *testing.T) {
	repo := &fakeProfilRepo{profil: &models.Profil{
		UserID:      "u1",
		DisplayName: "Alice",
		Visibility:  models.VisibilityPrivate,
	}}
	follows := &spyFollowChecker{acceptErr: errors.New("user-service down")}
	svc := New(repo, 0, WithFollowChecker(follows))

	got, err := svc.Update(context.Background(), "u1", models.UpdateProfilRequest{Visibility: ptr(models.VisibilityPublic)})
	if err != nil {
		t.Fatalf("Update ne doit pas echouer si accept-all echoue: %v", err)
	}
	if got.Visibility != models.VisibilityPublic {
		t.Fatalf("visibility apres update = %q", got.Visibility)
	}
	if !follows.acceptCalled {
		t.Fatal("passer de prive a public doit declencher AcceptAllFollowRequests")
	}
}

func TestSetCertification(t *testing.T) {
	repo := &fakeProfilRepo{profil: &models.Profil{UserID: "u1", Certification: models.CertificationNone}}
	svc := New(repo, 0)

	got, err := svc.SetCertification(context.Background(), "u1", models.CertificationPolitical)
	if err != nil {
		t.Fatalf("SetCertification erreur inattendue: %v", err)
	}
	if got.Certification != models.CertificationPolitical {
		t.Fatalf("certification = %q", got.Certification)
	}
	if repo.updatedUser != "u1" || repo.updateSet["certification"] != models.CertificationPolitical {
		t.Fatalf("updateSet inattendu: user=%q set=%#v", repo.updatedUser, repo.updateSet)
	}
	if _, ok := repo.updateSet["updated_at"].(time.Time); !ok {
		t.Fatalf("updated_at doit etre pose: %#v", repo.updateSet)
	}
}

func TestSetCertification_InvalidValue(t *testing.T) {
	repo := &fakeProfilRepo{profil: &models.Profil{UserID: "u1"}}
	svc := New(repo, 0)

	_, err := svc.SetCertification(context.Background(), "u1", "gold")
	if !errors.Is(err, ErrInvalidCertification) {
		t.Fatalf("attendu ErrInvalidCertification, obtenu %v", err)
	}
	if repo.updateCalled {
		t.Fatal("valeur invalide ne doit pas appeler le repository")
	}
}

func TestTouchActivityUpdatesExpectedFields(t *testing.T) {
	repo := &fakeProfilRepo{profil: &models.Profil{UserID: "u1"}}
	svc := New(repo, 0)

	got, err := svc.TouchActivity(context.Background(), "u1", false)
	if err != nil {
		t.Fatalf("TouchActivity erreur inattendue: %v", err)
	}
	if got.IsOnline {
		t.Fatal("online=false doit etre persiste")
	}
	if repo.updatedUser != "u1" || repo.updateSet["is_online"] != false {
		t.Fatalf("updateSet inattendu: user=%q set=%#v", repo.updatedUser, repo.updateSet)
	}
	if _, ok := repo.updateSet["last_login_at"].(time.Time); !ok {
		t.Fatalf("last_login_at doit etre pose: %#v", repo.updateSet)
	}
}

func TestDelete_MapsMissingProfil(t *testing.T) {
	svc := New(&fakeProfilRepo{deleteErr: mongo.ErrNoDocuments}, 0)

	err := svc.Delete(context.Background(), "missing")
	if !errors.Is(err, ErrProfilNotFound) {
		t.Fatalf("attendu ErrProfilNotFound, obtenu %v", err)
	}
}

func TestServiceErrors(t *testing.T) {
	sentinel := errors.New("db")

	if _, err := New(&fakeProfilRepo{err: sentinel}, 0).GetByUserID(context.Background(), "u1"); !errors.Is(err, sentinel) {
		t.Fatalf("GetByUserID attendu sentinel, obtenu %v", err)
	}
	if _, err := New(&fakeProfilRepo{err: sentinel}, 0).Search(context.Background(), "alice", 10); !errors.Is(err, sentinel) {
		t.Fatalf("Search attendu sentinel, obtenu %v", err)
	}
	if _, err := New(&fakeProfilRepo{profil: &models.Profil{UserID: "u1"}, updateErr: sentinel}, 0).Update(context.Background(), "u1", models.UpdateProfilRequest{Bio: ptr("bio")}); !errors.Is(err, sentinel) {
		t.Fatalf("Update attendu sentinel, obtenu %v", err)
	}
	if _, err := New(&fakeProfilRepo{updateErr: mongo.ErrNoDocuments}, 0).TouchActivity(context.Background(), "u1", true); !errors.Is(err, ErrProfilNotFound) {
		t.Fatalf("TouchActivity attendu ErrProfilNotFound, obtenu %v", err)
	}
	if err := New(&fakeProfilRepo{deleteErr: sentinel}, 0).Delete(context.Background(), "u1"); !errors.Is(err, sentinel) {
		t.Fatalf("Delete attendu sentinel, obtenu %v", err)
	}
}
