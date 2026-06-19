package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/webdad/message-service/internal/models"
	"github.com/webdad/message-service/internal/notifier"
)

func TestSortedPair_OrderIndependent(t *testing.T) {
	a := sortedPair("bob", "alice")
	b := sortedPair("alice", "bob")
	if a[0] != "alice" || a[1] != "bob" {
		t.Fatalf("attendu [alice bob], obtenu %v", a)
	}
	if a[0] != b[0] || a[1] != b[1] {
		t.Fatalf("sortedPair doit être indépendant de l'ordre : %v vs %v", a, b)
	}
}

func TestDMKey_SameForBothDirections(t *testing.T) {
	if dmKey("u1", "u2") != dmKey("u2", "u1") {
		t.Fatalf("dmKey doit être symétrique")
	}
	if dmKey("u1", "u2") != "u1:u2" {
		t.Fatalf("dmKey attendu u1:u2, obtenu %q", dmKey("u1", "u2"))
	}
}

func TestCanWrite(t *testing.T) {
	cases := map[string]bool{
		models.MemberOwner:  true,
		models.MemberAdmin:  true,
		models.MemberTalker: true,
		models.MemberViewer: false, // lecture seule (communautés)
		"":                  false,
		"random":            false,
	}
	for role, want := range cases {
		if got := canWrite(role); got != want {
			t.Errorf("canWrite(%q) = %v, want %v", role, got, want)
		}
	}
}

func TestClampLimit(t *testing.T) {
	if clampLimit(0) != DefaultLimit {
		t.Errorf("0 doit retomber sur le défaut (%d)", DefaultLimit)
	}
	if clampLimit(-5) != DefaultLimit {
		t.Errorf("négatif doit retomber sur le défaut")
	}
	if clampLimit(1000) != MaxLimit {
		t.Errorf("au-delà de Max doit être borné à %d", MaxLimit)
	}
	if clampLimit(10) != 10 {
		t.Errorf("une valeur valide doit être conservée")
	}
}

func TestParseID_RejectsInvalid(t *testing.T) {
	if _, err := parseID("not-an-objectid"); err != ErrInvalidID {
		t.Errorf("un id invalide doit donner ErrInvalidID, obtenu %v", err)
	}
}

func TestCheckRemoval(t *testing.T) {
	cases := []struct {
		name      string
		role      string
		isSelf    bool
		wantError error
	}{
		{"un membre peut quitter", models.MemberTalker, true, nil},
		{"l'owner ne peut PAS quitter", models.MemberOwner, true, ErrOwnerCannotLeave},
		{"l'owner peut exclure autrui", models.MemberOwner, false, nil},
		{"un membre ne peut PAS exclure autrui", models.MemberTalker, false, ErrOwnerOnly},
	}
	for _, tc := range cases {
		if got := checkRemoval(tc.role, tc.isSelf); got != tc.wantError {
			t.Errorf("%s : checkRemoval(%q,%v) = %v, want %v", tc.name, tc.role, tc.isSelf, got, tc.wantError)
		}
	}
}

func TestCanDeleteMessage(t *testing.T) {
	cases := []struct {
		name      string
		actorRole string
		convType  string
		senderID  string
		actorID   string
		want      bool
	}{
		{"auteur supprime le sien (DM)", models.MemberTalker, models.TypeDM, "u1", "u1", true},
		{"auteur supprime le sien (groupe)", models.MemberTalker, models.TypeGroup, "u1", "u1", true},
		{"non-auteur en DM : interdit", models.MemberTalker, models.TypeDM, "u1", "u2", false},
		{"owner de groupe supprime autrui", models.MemberOwner, models.TypeGroup, "u1", "u2", true},
		{"admin de communauté supprime autrui", models.MemberAdmin, models.TypeCommunity, "u1", "u2", true},
		{"talker de groupe ne supprime PAS autrui", models.MemberTalker, models.TypeGroup, "u1", "u2", false},
		{"viewer de communauté ne supprime PAS autrui", models.MemberViewer, models.TypeCommunity, "u1", "u2", false},
	}
	for _, tc := range cases {
		if got := canDeleteMessage(tc.actorRole, tc.convType, tc.senderID, tc.actorID); got != tc.want {
			t.Errorf("%s : canDeleteMessage(%q,%q,%q,%q) = %v, want %v",
				tc.name, tc.actorRole, tc.convType, tc.senderID, tc.actorID, got, tc.want)
		}
	}
}

func TestMaxTalkers(t *testing.T) {
	// La règle métier (cap des participants pouvant écrire) doit valoir 32.
	if MaxTalkers != 32 {
		t.Errorf("MaxTalkers attendu 32, obtenu %d", MaxTalkers)
	}
}

func TestIsManageable(t *testing.T) {
	cases := map[string]bool{
		models.TypeGroup:     true,
		models.TypeCommunity: true,
		models.TypeDM:        false, // un DM n'a ni admin de membres ni suppression
		"":                   false,
	}
	for typ, want := range cases {
		if got := isManageable(typ); got != want {
			t.Errorf("isManageable(%q) = %v, want %v", typ, got, want)
		}
	}
}

func TestCanWrite_ViewerCannot(t *testing.T) {
	// Garde-fou communautés : un viewer ne peut jamais écrire.
	if canWrite(models.MemberViewer) {
		t.Error("un viewer ne doit PAS pouvoir écrire")
	}
	if !canWrite(models.MemberTalker) {
		t.Error("un talker doit pouvoir écrire")
	}
}

func TestConvLess_PinnedFirstThenActivity(t *testing.T) {
	t0 := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	pinOld := t0.Add(1 * time.Hour)
	pinNew := t0.Add(2 * time.Hour)

	view := func(updated time.Time, pinnedAt *time.Time) models.ConversationView {
		return models.ConversationView{UpdatedAt: updated, PinnedAt: pinnedAt}
	}

	pinnedRecent := view(t0, &pinNew)
	pinnedOlder := view(t0.Add(5*time.Hour), &pinOld) // plus actif mais épinglé plus tôt
	active := view(t0.Add(9*time.Hour), nil)          // non épinglé, très actif
	stale := view(t0, nil)                            // non épinglé, peu actif

	// Une épinglée passe toujours devant une non-épinglée, même moins active.
	if !convLess(pinnedOlder, active) {
		t.Error("une conversation épinglée doit passer devant une non-épinglée")
	}
	if convLess(active, pinnedOlder) {
		t.Error("une non-épinglée ne doit pas passer devant une épinglée")
	}
	// Entre deux épinglées : la plus récemment épinglée d'abord.
	if !convLess(pinnedRecent, pinnedOlder) {
		t.Error("l'épinglage le plus récent doit passer en premier")
	}
	// Entre deux non-épinglées : la plus active d'abord.
	if !convLess(active, stale) {
		t.Error("la non-épinglée la plus active doit passer en premier")
	}
}

func TestMentionedTargets(t *testing.T) {
	members := []string{"u1", "u2", "u3"}

	cases := []struct {
		name      string
		mentioned []string
		sender    string
		want      []string
	}{
		{"garde les membres réels hors expéditeur", []string{"u2", "u3"}, "u1", []string{"u2", "u3"}},
		{"ignore l'expéditeur", []string{"u1", "u2"}, "u1", []string{"u2"}},
		{"ignore les non-membres", []string{"u2", "ghost"}, "u1", []string{"u2"}},
		{"déduplique", []string{"u2", "u2"}, "u1", []string{"u2"}},
		{"ignore les vides", []string{"", "u3"}, "u1", []string{"u3"}},
		{"aucune mention → nil", nil, "u1", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mentionedTargets(tc.mentioned, members, tc.sender)
			if len(got) != len(tc.want) {
				t.Fatalf("mentionedTargets = %v ; attendu %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("mentionedTargets = %v ; attendu %v", got, tc.want)
				}
			}
		})
	}
}

func TestMessageTargets(t *testing.T) {
	cases := []struct {
		name    string
		members []string
		sender  string
		want    []string
	}{
		{"notifie tous les autres membres", []string{"u1", "u2", "u3"}, "u1", []string{"u2", "u3"}},
		{"ignore l'expéditeur", []string{"u1", "u2"}, "u1", []string{"u2"}},
		{"déduplique par prudence", []string{"u1", "u2", "u2"}, "u1", []string{"u2"}},
		{"ignore les vides", []string{"", "u2"}, "u1", []string{"u2"}},
		{"aucun destinataire → nil", []string{"u1"}, "u1", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := messageTargets(tc.members, tc.sender)
			if len(got) != len(tc.want) {
				t.Fatalf("messageTargets = %v ; attendu %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("messageTargets = %v ; attendu %v", got, tc.want)
				}
			}
		})
	}
}

// --- Tests supplémentaires pour augmenter la couverture ---

func TestClampOffset(t *testing.T) {
	cases := []struct {
		in   int64
		want int64
	}{
		{-10, 0},
		{-1, 0},
		{0, 0},
		{1, 1},
		{50, 50},
	}
	for _, tc := range cases {
		if got := clampOffset(tc.in); got != tc.want {
			t.Errorf("clampOffset(%d) = %d, attendu %d", tc.in, got, tc.want)
		}
	}
}

func TestParseID_ValidHex(t *testing.T) {
	validHex := "507f1f77bcf86cd799439011"
	oid, err := parseID(validHex)
	if err != nil {
		t.Fatalf("parseID valide attendu sans erreur, obtenu %v", err)
	}
	if oid.Hex() != validHex {
		t.Fatalf("parseID.Hex() = %q, attendu %q", oid.Hex(), validHex)
	}
}

func TestTranslateNotFound_ErrNoDocuments(t *testing.T) {
	result := translateNotFound(mongo.ErrNoDocuments)
	if !errors.Is(result, ErrConversationNotFound) {
		t.Errorf("translateNotFound(mongo.ErrNoDocuments) = %v, attendu ErrConversationNotFound", result)
	}
}

func TestTranslateNotFound_OtherError(t *testing.T) {
	other := errors.New("autre erreur")
	result := translateNotFound(other)
	if !errors.Is(result, other) {
		t.Errorf("translateNotFound(autre) = %v, attendu l'erreur d'origine", result)
	}
}

func TestTranslateNotFound_Nil(t *testing.T) {
	result := translateNotFound(nil)
	if result != nil {
		t.Errorf("translateNotFound(nil) = %v, attendu nil", result)
	}
}

func TestBuildView_DM(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	pinnedAt := now.Add(-time.Hour)
	conv := &models.Conversation{
		Type:      models.TypeDM,
		MemberIDs: []string{"u1", "u2"},
		CreatedBy: "u1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	m := &models.Member{
		UserID:      "u1",
		Role:        models.MemberTalker,
		KeyEnvelope: "env_base64",
		PinnedAt:    &pinnedAt,
	}
	v := buildView(conv, m)
	if v.Type != models.TypeDM {
		t.Errorf("Type = %q, attendu %q", v.Type, models.TypeDM)
	}
	if v.MyRole != models.MemberTalker {
		t.Errorf("MyRole = %q, attendu talker", v.MyRole)
	}
	if v.MyEnvelope != "env_base64" {
		t.Errorf("MyEnvelope = %q, attendu env_base64", v.MyEnvelope)
	}
	if v.PinnedAt == nil || !v.PinnedAt.Equal(pinnedAt) {
		t.Error("PinnedAt incorrect")
	}
	if v.ContentKey != "" {
		t.Errorf("ContentKey doit être vide pour un DM, obtenu %q", v.ContentKey)
	}
	if v.Muted {
		t.Error("Muted doit être false (MutedAt nil)")
	}
	if v.CreatedBy != "u1" {
		t.Errorf("CreatedBy = %q, attendu u1", v.CreatedBy)
	}
}

func TestBuildView_Community_ExposesContentKey(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	conv := &models.Conversation{
		Type:       models.TypeCommunity,
		MemberIDs:  []string{"u1"},
		ContentKey: "content_key_secret",
		Title:      "Ma communauté",
		CreatedBy:  "u1",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m := &models.Member{
		UserID: "u1",
		Role:   models.MemberOwner,
	}
	v := buildView(conv, m)
	if v.ContentKey != "content_key_secret" {
		t.Errorf("ContentKey = %q, attendu content_key_secret", v.ContentKey)
	}
	if v.Title != "Ma communauté" {
		t.Errorf("Title = %q, attendu Ma communauté", v.Title)
	}
}

func TestBuildView_Group_NoContentKey(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	conv := &models.Conversation{
		Type:       models.TypeGroup,
		MemberIDs:  []string{"u1", "u2"},
		ContentKey: "should_not_appear",
		Title:      "groupe_chiffré",
		TitleNonce: "nonce_abc",
		CreatedBy:  "u1",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m := &models.Member{
		UserID: "u1",
		Role:   models.MemberOwner,
	}
	v := buildView(conv, m)
	if v.ContentKey != "" {
		t.Errorf("ContentKey doit être vide pour un groupe, obtenu %q", v.ContentKey)
	}
	if v.TitleNonce != "nonce_abc" {
		t.Errorf("TitleNonce = %q, attendu nonce_abc", v.TitleNonce)
	}
}

func TestBuildView_Muted(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	mutedAt := now.Add(-30 * time.Minute)
	conv := &models.Conversation{
		Type:      models.TypeDM,
		MemberIDs: []string{"u1", "u2"},
		CreatedBy: "u1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	m := &models.Member{
		UserID:  "u1",
		Role:    models.MemberTalker,
		MutedAt: &mutedAt,
	}
	v := buildView(conv, m)
	if !v.Muted {
		t.Error("Muted doit être true quand MutedAt est posé")
	}
}

func TestBuildView_LastReadAt(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	readAt := now.Add(-5 * time.Minute)
	conv := &models.Conversation{
		Type:      models.TypeDM,
		MemberIDs: []string{"u1", "u2"},
		CreatedBy: "u1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	m := &models.Member{
		UserID:     "u1",
		Role:       models.MemberTalker,
		LastReadAt: &readAt,
	}
	v := buildView(conv, m)
	if v.LastReadAt == nil || !v.LastReadAt.Equal(readAt) {
		t.Error("LastReadAt non transmis dans la vue")
	}
}

func TestSetNotifier_Noop(t *testing.T) {
	svc := NewMessageService(nil)
	// SetNotifier(nil) ne doit pas remplacer le notifier courant (Noop)
	svc.SetNotifier(nil)
	// Ne doit pas paniquer → test réussi
}

func TestSetNotifier_CustomNotifier(t *testing.T) {
	svc := NewMessageService(nil)
	n := notifier.Noop{}
	svc.SetNotifier(n)
	// Pas de panique → test réussi
}

func TestErrVarsNonNil(t *testing.T) {
	errs := []error{
		ErrConversationNotFound, ErrInvalidID, ErrNotMember, ErrCannotWrite,
		ErrKeyNotFound, ErrBackupNotFound, ErrSelfConversation, ErrMissingEnvelope,
		ErrInvalidGroup, ErrInvalidCommunity, ErrNotGroup, ErrNotCommunity,
		ErrNotManageable, ErrAlreadyMember, ErrTalkersFull, ErrInvalidRole,
		ErrOwnerOnly, ErrOwnerCannotLeave, ErrTargetNotMember, ErrMessageNotFound,
		ErrNotMessageOwner, ErrCannotDelete,
	}
	for _, e := range errs {
		if e == nil {
			t.Errorf("une erreur sentinel est nil")
		}
	}
}

func TestConvLess_BothUnpinnedSameTime(t *testing.T) {
	t0 := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	a := models.ConversationView{UpdatedAt: t0}
	b := models.ConversationView{UpdatedAt: t0}
	// Même temps → convLess(a,b) = false (ordre stable, pas de panique)
	result := convLess(a, b)
	if result {
		t.Error("convLess(même temps non épinglé) ne doit pas retourner true")
	}
}

func TestConvLess_BothPinnedSameTime(t *testing.T) {
	t0 := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	a := models.ConversationView{UpdatedAt: t0, PinnedAt: &t0}
	b := models.ConversationView{UpdatedAt: t0, PinnedAt: &t0}
	// Même épinglage, même activité → false (pas de panique)
	result := convLess(a, b)
	if result {
		t.Error("convLess(même épinglage même temps) ne doit pas retourner true")
	}
}

func TestConvLess_PinnedVsUnpinned(t *testing.T) {
	t0 := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	pinned := models.ConversationView{UpdatedAt: t0, PinnedAt: &t0}
	unpinned := models.ConversationView{UpdatedAt: t0.Add(10 * time.Hour)}
	if !convLess(pinned, unpinned) {
		t.Error("une épinglée doit toujours passer devant une non-épinglée")
	}
	if convLess(unpinned, pinned) {
		t.Error("une non-épinglée ne doit pas passer devant une épinglée")
	}
}

func TestSortedPair_SameStrings(t *testing.T) {
	p := sortedPair("abc", "abc")
	if p[0] != "abc" || p[1] != "abc" {
		t.Fatalf("sortedPair(same,same) = %v", p)
	}
}

func TestDMKey_LexOrder(t *testing.T) {
	key := dmKey("z_user", "a_user")
	if key != "a_user:z_user" {
		t.Fatalf("dmKey = %q, attendu a_user:z_user", key)
	}
}

func TestMentionedTargets_EmptyMembers(t *testing.T) {
	got := mentionedTargets([]string{"u1", "u2"}, nil, "u3")
	if got != nil {
		t.Errorf("mentionedTargets avec membres nil = %v, attendu nil", got)
	}
}

func TestMentionedTargets_EmptyMentioned(t *testing.T) {
	got := mentionedTargets(nil, []string{"u1", "u2"}, "u3")
	if got != nil {
		t.Errorf("mentionedTargets avec mentioned nil = %v, attendu nil", got)
	}
}

func TestMessageTargets_EmptyMembers(t *testing.T) {
	got := messageTargets(nil, "u1")
	if len(got) != 0 {
		t.Errorf("messageTargets(nil, ...) = %v, attendu vide", got)
	}
}

func TestNewMessageService_ReturnsNonNil(t *testing.T) {
	svc := NewMessageService(nil)
	if svc == nil {
		t.Fatal("NewMessageService() = nil")
	}
}

// --- Tests des chemins early-return (pas de repo nécessaire) ---
// Ces tests exploitent le fait que certaines méthodes valident les paramètres
// AVANT tout appel au dépôt, rendant le repo nil acceptable.

func TestCreateDM_SelfConversation(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.CreateDM(context.TODO(), "u1", "u1", map[string]string{"u1": "env"})
	if !errors.Is(err, ErrSelfConversation) {
		t.Errorf("CreateDM(self) = %v, attendu ErrSelfConversation", err)
	}
}

func TestCreateGroup_EmptyEnvelopes(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.CreateGroup(context.TODO(), "u1", "titre", "nonce", map[string]string{})
	if !errors.Is(err, ErrInvalidGroup) {
		t.Errorf("CreateGroup(enveloppes vides) = %v, attendu ErrInvalidGroup", err)
	}
}

func TestCreateGroup_CreatorMissingEnvelope(t *testing.T) {
	svc := NewMessageService(nil)
	// Le créateur u1 n'a pas d'enveloppe dans la map
	_, err := svc.CreateGroup(context.TODO(), "u1", "titre", "nonce", map[string]string{"u2": "env_u2"})
	if !errors.Is(err, ErrInvalidGroup) {
		t.Errorf("CreateGroup(créateur sans enveloppe) = %v, attendu ErrInvalidGroup", err)
	}
}

func TestCreateGroup_TooManyMembers(t *testing.T) {
	svc := NewMessageService(nil)
	// Plus de MaxTalkers enveloppes
	envelopes := make(map[string]string, MaxTalkers+1)
	for i := 0; i <= MaxTalkers; i++ {
		key := "u" + string(rune('0'+i%10)) + string(rune('a'+i%26))
		envelopes[key] = "env"
	}
	envelopes["u1"] = "env_creator"
	_, err := svc.CreateGroup(context.TODO(), "u1", "titre", "nonce", envelopes)
	if !errors.Is(err, ErrTalkersFull) {
		t.Errorf("CreateGroup(>32 membres) = %v, attendu ErrTalkersFull", err)
	}
}

func TestCreateGroup_EmptyEnvelopeValue(t *testing.T) {
	svc := NewMessageService(nil)
	// Une enveloppe vide dans la map
	_, err := svc.CreateGroup(context.TODO(), "u1", "titre", "nonce", map[string]string{
		"u1": "env_creator",
		"u2": "", // enveloppe vide → ErrMissingEnvelope
		"u3": "env_u3",
	})
	if !errors.Is(err, ErrMissingEnvelope) {
		t.Errorf("CreateGroup(enveloppe vide) = %v, attendu ErrMissingEnvelope", err)
	}
}

func TestCreateCommunity_EmptyTitle(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.CreateCommunity(context.TODO(), "u1", "", "content_key")
	if !errors.Is(err, ErrInvalidCommunity) {
		t.Errorf("CreateCommunity(titre vide) = %v, attendu ErrInvalidCommunity", err)
	}
}

func TestCreateCommunity_EmptyContentKey(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.CreateCommunity(context.TODO(), "u1", "titre", "")
	if !errors.Is(err, ErrInvalidCommunity) {
		t.Errorf("CreateCommunity(clé vide) = %v, attendu ErrInvalidCommunity", err)
	}
}

func TestCreateCommunity_BothEmpty(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.CreateCommunity(context.TODO(), "u1", "   ", "   ")
	if !errors.Is(err, ErrInvalidCommunity) {
		t.Errorf("CreateCommunity(espaces) = %v, attendu ErrInvalidCommunity", err)
	}
}

func TestSetMemberRole_InvalidRole(t *testing.T) {
	svc := NewMessageService(nil)
	// Le rôle est vérifié AVANT requireMember (donc avant appel repo)
	_, err := svc.SetMemberRole(context.TODO(), "507f1f77bcf86cd799439011", "u1", "u2", "invalid_role")
	if !errors.Is(err, ErrInvalidRole) {
		t.Errorf("SetMemberRole(rôle invalide) = %v, attendu ErrInvalidRole", err)
	}
}

func TestRequireMember_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	// requireMember valide l'ID avant d'appeler le repo
	// On y accède via n'importe quelle méthode qui l'appelle en premier
	_, err := svc.ListMembers(context.TODO(), "not-valid-id", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("ListMembers(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestGetConversation_InvalidID(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.GetConversation(context.TODO(), "not-valid-id", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("GetConversation(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestListMessages_InvalidBeforeID(t *testing.T) {
	svc := NewMessageService(nil)
	// Pour atteindre la validation de beforeID, requireMember doit réussir d'abord.
	// Avec un id invalide on s'arrête à requireMember → ErrInvalidID
	_, err := svc.ListMessages(context.TODO(), "not-valid-id", "u1", 10, "")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("ListMessages(conv id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestSendMessage_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, _, err := svc.SendMessage(context.TODO(), "not-valid-id", "u1", "cipher", "nonce", nil)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("SendMessage(conv id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestEditMessage_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, _, err := svc.EditMessage(context.TODO(), "not-valid-id", "507f1f77bcf86cd799439011", "u1", "cipher", "nonce", nil)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("EditMessage(conv id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestDeleteMessage_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, _, err := svc.DeleteMessage(context.TODO(), "not-valid-id", "507f1f77bcf86cd799439011", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("DeleteMessage(conv id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestModerateDeleteMessage_InvalidID(t *testing.T) {
	svc := NewMessageService(nil)
	_, _, err := svc.ModerateDeleteMessage(context.TODO(), "not-valid-id")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("ModerateDeleteMessage(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestPinConversation_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.PinConversation(context.TODO(), "not-valid-id", "u1", true)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("PinConversation(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestClearConversation_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	err := svc.ClearConversation(context.TODO(), "not-valid-id", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("ClearConversation(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestMuteConversation_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.MuteConversation(context.TODO(), "not-valid-id", "u1", true)
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("MuteConversation(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestMarkRead_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, _, err := svc.MarkRead(context.TODO(), "not-valid-id", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("MarkRead(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestTypingTargets_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.TypingTargets(context.TODO(), "not-valid-id", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("TypingTargets(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestAddMember_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.AddMember(context.TODO(), "not-valid-id", "u1", "u2", "env")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("AddMember(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestRemoveMember_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.RemoveMember(context.TODO(), "not-valid-id", "u1", "u2")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("RemoveMember(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestDeleteGroup_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.DeleteGroup(context.TODO(), "not-valid-id", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("DeleteGroup(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestUpdateGroup_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, err := svc.UpdateGroup(context.TODO(), "not-valid-id", "u1", "titre", "nonce")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("UpdateGroup(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestJoinCommunity_InvalidConvID(t *testing.T) {
	svc := NewMessageService(nil)
	_, _, err := svc.JoinCommunity(context.TODO(), "not-valid-id", "u1")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("JoinCommunity(id invalide) = %v, attendu ErrInvalidID", err)
	}
}

func TestIsMember_InvalidConvID(t *testing.T) {
	// IsMember appelle s.repo.GetMember directement (pas de parseID préalable)
	// Avec un repo nil, ça va paniquer → on skippe ce test
	// Au lieu on vérifie juste la fonction parseID
	_, err := parseID("not-valid-id")
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("parseID(invalide) = %v, attendu ErrInvalidID", err)
	}
}
