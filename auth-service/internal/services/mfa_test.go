package services

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// validKey produit une clé base64 de 32 octets, comme MFA_ENCRYPTION_KEY.
func validKey(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand : %v", err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func TestNewSecretCipher_EmptyKeyDisablesMFA(t *testing.T) {
	c, err := newSecretCipher("")
	if err != nil {
		t.Fatalf("clé vide ne doit pas erreur : %v", err)
	}
	if c != nil {
		t.Fatal("clé vide doit donner un cipher nil (MFA désactivée)")
	}
}

func TestNewSecretCipher_InvalidKey(t *testing.T) {
	if _, err := newSecretCipher("not-base64!!!"); err == nil {
		t.Error("base64 invalide doit erreur")
	}
	// 16 octets → AES-128, refusé (on impose AES-256).
	short := base64.StdEncoding.EncodeToString(make([]byte, 16))
	if _, err := newSecretCipher(short); err == nil {
		t.Error("clé de 16 octets doit erreur (32 attendus)")
	}
}

func TestSecretCipher_RoundTrip(t *testing.T) {
	c, err := newSecretCipher(validKey(t))
	if err != nil {
		t.Fatalf("newSecretCipher : %v", err)
	}
	const secret = "JBSWY3DPEHPK3PXP"

	enc, err := c.encrypt(secret)
	if err != nil {
		t.Fatalf("encrypt : %v", err)
	}
	if enc == secret {
		t.Fatal("le chiffré ne doit pas être égal au clair")
	}

	got, err := c.decrypt(enc)
	if err != nil {
		t.Fatalf("decrypt : %v", err)
	}
	if got != secret {
		t.Fatalf("round-trip : got %q, want %q", got, secret)
	}
}

func TestSecretCipher_NonceUnique(t *testing.T) {
	c, _ := newSecretCipher(validKey(t))
	a, _ := c.encrypt("same")
	b, _ := c.encrypt("same")
	if a == b {
		t.Fatal("deux chiffrés du même clair doivent différer (nonce aléatoire)")
	}
}

func TestSecretCipher_WrongKeyFails(t *testing.T) {
	c1, _ := newSecretCipher(validKey(t))
	c2, _ := newSecretCipher(validKey(t))
	enc, _ := c1.encrypt("secret")
	if _, err := c2.decrypt(enc); err == nil {
		t.Error("déchiffrement avec une autre clé doit échouer (auth GCM)")
	}
}

func TestSecretCipher_TamperedFails(t *testing.T) {
	c, _ := newSecretCipher(validKey(t))
	enc, _ := c.encrypt("secret")
	// Altère le dernier octet du chiffré base64.
	raw, _ := base64.StdEncoding.DecodeString(enc)
	raw[len(raw)-1] ^= 0xff
	if _, err := c.decrypt(base64.StdEncoding.EncodeToString(raw)); err == nil {
		t.Error("un chiffré altéré doit échouer à l'authentification GCM")
	}
}

func TestValidateTOTP(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: mfaIssuer, AccountName: "user@breezy.test"})
	if err != nil {
		t.Fatalf("génération clé : %v", err)
	}
	secret := key.Secret()

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("génération code : %v", err)
	}

	if !validateTOTP(secret, code) {
		t.Error("un code TOTP courant doit être valide")
	}
	if validateTOTP(secret, "000000") {
		t.Error("un code arbitraire ne doit pas être valide")
	}
	if validateTOTP(secret, "") {
		t.Error("un code vide ne doit jamais être valide")
	}
	// Tolérance d'espaces (saisie utilisateur).
	if !validateTOTP(secret, "  "+code+"  ") {
		t.Error("les espaces autour du code doivent être tolérés")
	}
}
