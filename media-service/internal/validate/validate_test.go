package validate

import (
	"errors"
	"testing"
)

// En-têtes « magic bytes » minimaux pour la détection.
var (
	pngHead  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	gifHead  = []byte("GIF89a\x00\x00")
	jpegHead = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0, 'J', 'F', 'I', 'F'}
	textHead = []byte("just some plain text, definitely not a media file")
)

func TestDetect(t *testing.T) {
	cases := []struct {
		name     string
		head     []byte
		wantMime string
		wantKind Kind
		wantErr  error
	}{
		{"png", pngHead, "image/png", KindImage, nil},
		{"gif", gifHead, "image/gif", KindImage, nil},
		{"jpeg", jpegHead, "image/jpeg", KindImage, nil},
		{"texte rejeté", textHead, "", "", ErrUnsupportedType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mime, kind, err := Detect(tc.head)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, attendu %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if mime != tc.wantMime {
				t.Errorf("mime = %q, attendu %q", mime, tc.wantMime)
			}
			if kind != tc.wantKind {
				t.Errorf("kind = %q, attendu %q", kind, tc.wantKind)
			}
		})
	}
}

func TestCheckSize(t *testing.T) {
	const maxImg, maxVid = 100, 1000
	cases := []struct {
		name    string
		kind    Kind
		size    int64
		wantErr error
	}{
		{"image sous le cap", KindImage, 100, nil},
		{"image au-delà", KindImage, 101, ErrTooLarge},
		{"vidéo sous le cap", KindVideo, 1000, nil},
		{"vidéo au-delà", KindVideo, 1001, ErrTooLarge},
		{"image jugée au cap vidéo refusée", KindImage, 500, ErrTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := CheckSize(tc.kind, tc.size, maxImg, maxVid); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, attendu %v", err, tc.wantErr)
			}
		})
	}
}

func TestMaxForKind(t *testing.T) {
	if got := MaxForKind(KindImage, 5, 50); got != 5 {
		t.Errorf("image: got %d", got)
	}
	if got := MaxForKind(KindVideo, 5, 50); got != 50 {
		t.Errorf("vidéo: got %d", got)
	}
}
