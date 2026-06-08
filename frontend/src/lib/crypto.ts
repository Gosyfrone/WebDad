/**
 * Primitives de chiffrement de bout en bout (E2EE) de la messagerie.
 *
 * Modèle (cf. CLAUDE.md, feature « Messages privés ») :
 *   - Chaque utilisateur a une paire de clés d'identité **X25519**. La clé
 *     PRIVÉE ne quitte jamais le navigateur (cf. `key-store.ts`) ; la clé
 *     publique est publiée au message-service.
 *   - Chaque conversation a une **clé de contenu** symétrique aléatoire (CK).
 *     Elle est **emballée** (chiffrée) pour chaque membre avec sa clé publique
 *     → une « enveloppe » par membre, stockée côté serveur. Le serveur ne voit
 *     que des enveloppes + des ciphertexts : il ne peut RIEN déchiffrer.
 *   - Les messages sont chiffrés avec la CK (XChaCha20-Poly1305, nonce aléatoire
 *     par message).
 *
 * L'emballage de clé est une « sealed box » anonyme (façon libsodium
 * `crypto_box_seal`) : une paire éphémère par scellage → le destinataire ouvre
 * avec sa seule clé privée, sans avoir besoin de la clé publique de l'émetteur.
 *
 * ⚠️ Ce module est PUR (aucune dépendance navigateur : ni window, ni
 * localStorage, ni IndexedDB) → testable en Node. Le stockage de la clé privée
 * et les appels réseau vivent respectivement dans `key-store.ts` et
 * `messages.ts`.
 */

import { x25519 } from '@noble/curves/ed25519'
import { xchacha20poly1305 } from '@noble/ciphers/chacha'
import { sha256 } from '@noble/hashes/sha256'

/** Tailles (octets) des primitives utilisées. */
const PUBLIC_KEY_LEN = 32
const NONCE_LEN = 24 // XChaCha20 : nonce 192 bits
export const CONTENT_KEY_LEN = 32 // clé symétrique 256 bits

/** Paire de clés d'identité (X25519). */
export interface KeyPair {
  publicKey: Uint8Array
  privateKey: Uint8Array
}

/** Message chiffré symétriquement : nonce + ciphertext (authentifié). */
export interface SymmetricCipher {
  nonce: Uint8Array
  ciphertext: Uint8Array
}

// --- Aléa ---------------------------------------------------------------------

/**
 * Octets aléatoires cryptographiques. Utilise WebCrypto, disponible aussi bien
 * dans le navigateur que dans Node 18+ (`globalThis.crypto`).
 */
export function randomBytes(length: number): Uint8Array {
  const out = new Uint8Array(length)
  globalThis.crypto.getRandomValues(out)
  return out
}

// --- Clés ---------------------------------------------------------------------

/** Génère une paire de clés d'identité X25519. */
export function generateIdentityKeyPair(): KeyPair {
  const privateKey = x25519.utils.randomPrivateKey()
  const publicKey = x25519.getPublicKey(privateKey)
  return { publicKey, privateKey }
}

/** Génère une clé de contenu symétrique (256 bits) pour une conversation. */
export function generateContentKey(): Uint8Array {
  return randomBytes(CONTENT_KEY_LEN)
}

// --- Sealed box (emballage de la clé de contenu pour un destinataire) --------

/**
 * Dérive la clé symétrique d'un échange X25519 (secret partagé lié aux deux
 * clés publiques, façon `crypto_box_seal`). Déterministe : le scelleur et
 * l'ouvreur calculent la même clé.
 */
function deriveSealKey(
  sharedSecret: Uint8Array,
  ephemeralPublicKey: Uint8Array,
  recipientPublicKey: Uint8Array,
): Uint8Array {
  const material = new Uint8Array(
    sharedSecret.length + ephemeralPublicKey.length + recipientPublicKey.length,
  )
  material.set(sharedSecret, 0)
  material.set(ephemeralPublicKey, sharedSecret.length)
  material.set(recipientPublicKey, sharedSecret.length + ephemeralPublicKey.length)
  return sha256(material)
}

/**
 * Scelle `plaintext` (typiquement une clé de contenu) à l'intention du
 * détenteur de `recipientPublicKey`. Format de l'enveloppe :
 *   ephemeralPublicKey(32) || nonce(24) || ciphertext
 * Seule la clé privée du destinataire permet de l'ouvrir.
 */
export function seal(recipientPublicKey: Uint8Array, plaintext: Uint8Array): Uint8Array {
  const ephemeral = generateIdentityKeyPair()
  const shared = x25519.getSharedSecret(ephemeral.privateKey, recipientPublicKey)
  const key = deriveSealKey(shared, ephemeral.publicKey, recipientPublicKey)
  const nonce = randomBytes(NONCE_LEN)
  const ciphertext = xchacha20poly1305(key, nonce).encrypt(plaintext)

  const out = new Uint8Array(ephemeral.publicKey.length + nonce.length + ciphertext.length)
  out.set(ephemeral.publicKey, 0)
  out.set(nonce, ephemeral.publicKey.length)
  out.set(ciphertext, ephemeral.publicKey.length + nonce.length)
  return out
}

/**
 * Ouvre une enveloppe scellée avec `seal` à l'aide de la clé privée du
 * destinataire. Lève si l'enveloppe est malformée ou l'authentification échoue
 * (mauvaise clé / altération).
 */
export function open(recipientPrivateKey: Uint8Array, sealed: Uint8Array): Uint8Array {
  if (sealed.length < PUBLIC_KEY_LEN + NONCE_LEN) {
    throw new Error('enveloppe scellée invalide (trop courte)')
  }
  const ephemeralPublicKey = sealed.subarray(0, PUBLIC_KEY_LEN)
  const nonce = sealed.subarray(PUBLIC_KEY_LEN, PUBLIC_KEY_LEN + NONCE_LEN)
  const ciphertext = sealed.subarray(PUBLIC_KEY_LEN + NONCE_LEN)

  const shared = x25519.getSharedSecret(recipientPrivateKey, ephemeralPublicKey)
  const recipientPublicKey = x25519.getPublicKey(recipientPrivateKey)
  const key = deriveSealKey(shared, ephemeralPublicKey, recipientPublicKey)
  return xchacha20poly1305(key, nonce).decrypt(ciphertext)
}

// --- Chiffrement symétrique des messages (avec la clé de contenu) ------------

/** Chiffre `plaintext` avec la clé de contenu (nonce aléatoire par message). */
export function encryptSymmetric(contentKey: Uint8Array, plaintext: Uint8Array): SymmetricCipher {
  const nonce = randomBytes(NONCE_LEN)
  const ciphertext = xchacha20poly1305(contentKey, nonce).encrypt(plaintext)
  return { nonce, ciphertext }
}

/** Déchiffre un message chiffré avec `encryptSymmetric`. Lève si altéré. */
export function decryptSymmetric(
  contentKey: Uint8Array,
  nonce: Uint8Array,
  ciphertext: Uint8Array,
): Uint8Array {
  return xchacha20poly1305(contentKey, nonce).decrypt(ciphertext)
}

// --- Encodage (frontières stockage / transport : tout en base64 + UTF-8) -----

/** Encode des octets en base64 (Node `Buffer` ou `btoa` navigateur). */
export function toBase64(bytes: Uint8Array): string {
  if (typeof Buffer !== 'undefined') return Buffer.from(bytes).toString('base64')
  let binary = ''
  for (const b of bytes) binary += String.fromCharCode(b)
  return btoa(binary)
}

/** Décode une chaîne base64 en octets. */
export function fromBase64(value: string): Uint8Array {
  if (typeof Buffer !== 'undefined') return new Uint8Array(Buffer.from(value, 'base64'))
  const binary = atob(value)
  const out = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) out[i] = binary.charCodeAt(i)
  return out
}

const encoder = new TextEncoder()
const decoder = new TextDecoder()

export function utf8ToBytes(text: string): Uint8Array {
  return encoder.encode(text)
}

export function bytesToUtf8(bytes: Uint8Array): string {
  return decoder.decode(bytes)
}

// --- Helpers de plus haut niveau (chaînes base64, utilisés par messages.ts) ---

/** Emballe une clé de contenu pour un destinataire → enveloppe base64. */
export function sealKeyForRecipient(recipientPublicKeyB64: string, contentKey: Uint8Array): string {
  return toBase64(seal(fromBase64(recipientPublicKeyB64), contentKey))
}

/** Ouvre une enveloppe (base64) avec sa clé privée → clé de contenu. */
export function openKeyEnvelope(envelopeB64: string, privateKey: Uint8Array): Uint8Array {
  return open(privateKey, fromBase64(envelopeB64))
}

/** Chiffre un texte avec la clé de contenu → {ciphertext, nonce} en base64. */
export function encryptText(
  contentKey: Uint8Array,
  text: string,
): { ciphertext: string; nonce: string } {
  const { nonce, ciphertext } = encryptSymmetric(contentKey, utf8ToBytes(text))
  return { ciphertext: toBase64(ciphertext), nonce: toBase64(nonce) }
}

/** Déchiffre un texte chiffré (champs base64) avec la clé de contenu. */
export function decryptText(contentKey: Uint8Array, ciphertextB64: string, nonceB64: string): string {
  return bytesToUtf8(decryptSymmetric(contentKey, fromBase64(nonceB64), fromBase64(ciphertextB64)))
}
