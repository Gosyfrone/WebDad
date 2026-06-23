// Package models contient les structures de données du service profil.
package models

import "time"

// Rôles valides (alignés sur l'enum du JWT / auth-service).
const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

const (
	CertificationNone         = "none"
	CertificationPolitical    = "political"
	CertificationPublicFigure = "public_figure"
)

// Profil représente un document de la collection `profiles` : uniquement les
// données « décoratives » d'un utilisateur. L'identité (username), les rôles
// et le graphe social vivent dans user-service / le JWT — pas ici (une seule
// source de vérité par donnée). La vue agrégée d'un profil (identité +
// compteurs + décoratif) est composée à la lecture par l'appelant.
type Profil struct {
	UserID          string     `json:"user_id"               bson:"user_id"`
	DisplayName     string     `json:"display_name"          bson:"display_name"`
	Bio             string     `json:"bio"                   bson:"bio"`
	AvatarURL       string     `json:"avatar_url"            bson:"avatar_url"`
	BannerURL       string     `json:"banner_url"            bson:"banner_url"`
	Website         string     `json:"website"               bson:"website"`
	Location        string     `json:"location"              bson:"location"`
	BirthDate       *time.Time `json:"birth_date,omitempty"  bson:"birth_date,omitempty"`
	Gender          string     `json:"gender,omitempty"      bson:"gender,omitempty"`      // "male" | "female"
	Nationality     string     `json:"nationality,omitempty" bson:"nationality,omitempty"` // ISO 3166-1 alpha-2
	CreatedAt       time.Time  `json:"created_at"            bson:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"            bson:"updated_at"`
	Visibility      string     `json:"visibility"       bson:"visibility,omitempty"`
	LikesVisibility string     `json:"likes_visibility"  bson:"likes_visibility,omitempty"`
	Certification   string     `json:"certification"     bson:"certification,omitempty"`
	// ActivityVisibility contrôle l'affichage public de last_login_at. Un profil
	// privé ne l'expose qu'au propriétaire et aux abonnés acceptés.
	ActivityVisibility string     `json:"activity_visibility" bson:"activity_visibility,omitempty"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty" bson:"last_login_at,omitempty"`
	IsOnline           bool       `json:"is_online" bson:"is_online,omitempty"`
	// DisplayNameChangedAt : date du dernier changement EFFECTIF de display_name
	// (nil = jamais changé depuis le provisioning). Enregistrée dès aujourd'hui
	// pour servir de base à un cooldown « X jours entre deux changements de nom »
	// (cf. config DisplayNameCooldown ; enforcement désactivé par défaut). On la
	// capture maintenant pour ne pas avoir de profils sans baseline le jour où
	// le cooldown est activé. Symétrique de users.username_changed_at.
	DisplayNameChangedAt *time.Time `json:"display_name_changed_at,omitempty" bson:"display_name_changed_at,omitempty"`
	// NsfwEnabled : préférence « afficher le contenu marqué NSFW ». Pointeur +
	// omitempty : un document antérieur au champ (nil) vaut `true` (cf.
	// NsfwEnabledOf) — défaut ON pour ne pas modifier l'expérience de la prod
	// existante. La valeur stockée d'un MINEUR est sans effet : la politique
	// effective (NsfwVisible) la combine avec la majorité calculée serveur.
	NsfwEnabled *bool `json:"nsfw_enabled,omitempty" bson:"nsfw_enabled,omitempty"`
	// IsAdult / NsfwVisible : politique « viewer » TRANSIENTE (bson:"-", jamais
	// stockée), calculée serveur depuis birth_date et exposée seulement sur la vue
	// privée /profils/me. IsAdult = ≥18 ans (dynamique : se débloque tout seul le
	// jour des 18 ans). NsfwVisible = IsAdult && NsfwEnabledOf : un mineur ne peut
	// pas se faire passer pour majeur (front non autoritatif, cf. Rule 6).
	IsAdult     bool `json:"is_adult" bson:"-"`
	NsfwVisible bool `json:"nsfw_visible" bson:"-"`
}

// nsfwMajorityAge : âge (en années) à partir duquel le contenu NSFW est
// autorisé. Distinct de l'âge minimum d'inscription (13 ans, côté front).
const nsfwMajorityAge = 18

// NsfwEnabledOf normalise la préférence NSFW : absente (vieux document) ⇒ true
// (défaut ON, pour ne pas modifier l'expérience de la prod existante).
func NsfwEnabledOf(p *Profil) bool {
	if p == nil || p.NsfwEnabled == nil {
		return true
	}
	return *p.NsfwEnabled
}

// IsAdultAt indique si une date de naissance correspond à ≥18 ans à l'instant
// `now`. birth_date nil ⇒ true (adulte par défaut : la migration backfille les
// profils sans date au 01/01/1999, et un profil créé sans date — ex. admin — ne
// doit pas être filtré). Calcul calendaire exact (anniversaire pas encore atteint
// dans l'année ⇒ une année de moins).
func IsAdultAt(birthDate *time.Time, now time.Time) bool {
	if birthDate == nil {
		return true
	}
	b := birthDate.UTC()
	years := now.Year() - b.Year()
	if now.Month() < b.Month() || (now.Month() == b.Month() && now.Day() < b.Day()) {
		years--
	}
	return years >= nsfwMajorityAge
}

// HydrateViewerPolicy renseigne les champs transients is_adult / nsfw_visible
// depuis birth_date et la préférence NSFW. À n'appeler que sur la vue privée du
// propriétaire (/profils/me).
func HydrateViewerPolicy(p *Profil, now time.Time) {
	if p == nil {
		return
	}
	p.IsAdult = IsAdultAt(p.BirthDate, now)
	p.NsfwVisible = p.IsAdult && NsfwEnabledOf(p)
}

// CreateProfilRequest : payload de POST /profils. L'id provient TOUJOURS du
// JWT, jamais du corps. display_name est OBLIGATOIRE : le BFF y met le username
// choisi à l'inscription. On ne dérive jamais de nom depuis l'email (le
// provisioning paresseux crée un profil à display_name vide, cf. service).
type CreateProfilRequest struct {
	DisplayName string     `json:"display_name" binding:"required,max=100"`
	BirthDate   *time.Time `json:"birth_date"   binding:"omitempty"`
	Gender      *string    `json:"gender"       binding:"omitempty,oneof=male female"`
}

// AdminCreateProfilRequest : payload de POST /profils/admin (admin). Crée le
// profil d'un utilisateur donné (= credentials.id) lors du provisioning d'un
// compte créé de force par un admin. display_name = username effectif ; pas de
// birth_date (l'utilisateur la renseignera lui-même ensuite).
type AdminCreateProfilRequest struct {
	ID          string `json:"id"           binding:"required"`
	DisplayName string `json:"display_name" binding:"required,max=100"`
}

// UpdateProfilRequest : payload de PATCH /profils/me. Champs optionnels
// (pointeurs) : seuls les champs fournis sont modifiés (nil = inchangé).
// birth_date est settable UNE SEULE FOIS : une fois posée, toute tentative de
// la changer est refusée (cf. service).
type UpdateProfilRequest struct {
	DisplayName        *string    `json:"display_name" binding:"omitempty,max=100"`
	Bio                *string    `json:"bio"          binding:"omitempty,max=160"`
	AvatarURL          *string    `json:"avatar_url"   binding:"omitempty"`
	BannerURL          *string    `json:"banner_url"   binding:"omitempty"`
	Website            *string    `json:"website"      binding:"omitempty"`
	Location           *string    `json:"location"     binding:"omitempty"`
	BirthDate          *time.Time `json:"birth_date"   binding:"omitempty"`
	Gender             *string    `json:"gender"       binding:"omitempty,oneof=male female"`
	Nationality        *string    `json:"nationality"  binding:"omitempty,iso3166_1_alpha2"`
	Visibility         *string    `json:"visibility"        binding:"omitempty,oneof=public private"`
	LikesVisibility    *string    `json:"likes_visibility"   binding:"omitempty,oneof=public private"`
	ActivityVisibility *string    `json:"activity_visibility" binding:"omitempty,oneof=public private"`
	// NsfwEnabled : préférence « afficher le contenu NSFW » (toggle des paramètres).
	NsfwEnabled *bool `json:"nsfw_enabled" binding:"omitempty"`
}

// UpdateCertificationRequest : payload réservé aux modérateurs/admins pour
// attribuer ou retirer la certification décorative d'un profil.
type UpdateCertificationRequest struct {
	Certification string `json:"certification" binding:"required,oneof=none political public_figure"`
}
