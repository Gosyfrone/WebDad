// Package validate détermine la nature d'un média à partir de ses octets et
// applique l'allowlist + les caps de taille. La détection se fait par les
// « magic bytes » réels (gabriel-vasile/mimetype), JAMAIS par l'en-tête
// Content-Type fourni par le client (falsifiable).
package validate

import (
	"errors"

	"github.com/gabriel-vasile/mimetype"
)

// Kind : nature de média acceptée.
type Kind string

const (
	KindImage Kind = "image"
	KindVideo Kind = "video"
)

// Erreurs métier (traduites en codes HTTP par le handler).
var (
	// ErrUnsupportedType : type MIME hors allowlist (→ 415).
	ErrUnsupportedType = errors.New("type de média non supporté")
	// ErrTooLarge : taille au-delà du cap pour cette nature (→ 413).
	ErrTooLarge = errors.New("média trop volumineux")
)

// allowed : MIME détecté → nature. Source de vérité unique de l'allowlist.
var allowed = map[string]Kind{
	"image/jpeg": KindImage,
	"image/png":  KindImage,
	"image/webp": KindImage,
	"image/gif":  KindImage,
	"video/mp4":  KindVideo,
	"video/webm": KindVideo,
}

// Detect renvoie le MIME réel des octets (suffit d'en passer les premiers,
// ≥ 3072 recommandés par mimetype) et la nature, si le type est dans
// l'allowlist. Sinon ErrUnsupportedType.
func Detect(head []byte) (mime string, kind Kind, err error) {
	mt := mimetype.Detect(head)
	detected := stripParams(mt.String())
	k, ok := allowed[detected]
	if !ok {
		return detected, "", ErrUnsupportedType
	}
	return detected, k, nil
}

// MaxForKind renvoie le cap de taille (octets) applicable à une nature.
func MaxForKind(kind Kind, maxImage, maxVideo int64) int64 {
	if kind == KindVideo {
		return maxVideo
	}
	return maxImage
}

// CheckSize valide la taille déclarée contre le cap de la nature.
func CheckSize(kind Kind, size, maxImage, maxVideo int64) error {
	if size > MaxForKind(kind, maxImage, maxVideo) {
		return ErrTooLarge
	}
	return nil
}

// stripParams retire un éventuel paramètre du type MIME ("image/jpeg; ...").
func stripParams(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			return s[:i]
		}
	}
	return s
}
