package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// secretCipher chiffre/déchiffre le secret TOTP at-rest en AES-256-GCM.
// La clé (32 octets) vient de MFA_ENCRYPTION_KEY (base64) — jamais en dur
// (règle 5). Une fuite de la table `credentials` ne livre donc pas les secrets
// TOTP en clair sans la clé, qui vit uniquement dans l'environnement.
type secretCipher struct {
	aead cipher.AEAD
}

// newSecretCipher construit le cipher depuis une clé base64 de 32 octets.
// Clé vide → (nil, nil) : la MFA est alors simplement désactivée (pattern
// « nil = fonctionnalité off », symétrique du mailer), auth reste bootable.
func newSecretCipher(b64Key string) (*secretCipher, error) {
	if b64Key == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(b64Key)
	if err != nil {
		return nil, fmt.Errorf("MFA_ENCRYPTION_KEY : base64 invalide : %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("MFA_ENCRYPTION_KEY : 32 octets attendus (AES-256), %d reçus", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("init AES : %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init GCM : %w", err)
	}
	return &secretCipher{aead: aead}, nil
}

// encrypt chiffre un texte clair et retourne nonce||ciphertext encodé base64.
func (c *secretCipher) encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("génération nonce : %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// decrypt inverse encrypt. Une altération du chiffré (ou une mauvaise clé) fait
// échouer l'authentification GCM → erreur.
func (c *secretCipher) decrypt(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("déchiffrement : base64 invalide : %w", err)
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("déchiffrement : données trop courtes")
	}
	nonce, ciphertext := raw[:ns], raw[ns:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("déchiffrement : authentification échouée : %w", err)
	}
	return string(plaintext), nil
}
