/**
 * Sauvegarde CHIFFRÉE de la clé privée d'identité E2EE, pour retrouver sa
 * messagerie sur un autre appareil (cf. la doc d'architecture, feature « phrase de passe »).
 *
 * Modèle zero-knowledge : la clé privée X25519 est *emballée* par une clé de
 * chiffrement (KEK) **dérivée d'une phrase de passe** choisie par l'utilisateur
 * (Argon2id, lent et gourmand en mémoire → résistant au bruteforce GPU). Le
 * serveur ne stocke qu'un blob opaque (`salt`, `nonce`, clé emballée, paramètres
 * KDF) : il ne voit jamais la passphrase ni la clé privée. La propriété
 * admin-proof des DM est donc préservée.
 *
 *   KEK            = Argon2id(passphrase, salt, params)        (32 octets)
 *   wrappedPrivKey = XChaCha20-Poly1305(KEK, nonce, privKey)
 *
 * Une mauvaise passphrase produit une mauvaise KEK → l'authentification
 * Poly1305 échoue au déballage (exception), sans jamais révéler la clé.
 *
 * ⚠️ Module PUR (aucune dépendance navigateur) → testable en Node. Le stockage
 * de la clé et les appels réseau vivent dans `key-store.ts` / `messages.ts`.
 */

import { argon2id } from '@noble/hashes/argon2'

import {
  decryptSymmetric,
  encryptSymmetric,
  fromBase64,
  randomBytes,
  toBase64,
} from '@/lib/crypto'

/** Longueur (octets) de la KEK dérivée (clé symétrique 256 bits). */
const KEK_LEN = 32
/** Longueur (octets) du sel Argon2id. */
const SALT_LEN = 16

/**
 * Paramètres Argon2id, persistés AVEC la sauvegarde (le déballage doit rejouer
 * exactement la même dérivation). Réglage = **minimum recommandé OWASP** pour
 * Argon2id (19 Mio / 2 passes) : coûteux à brute-forcer tout en restant rapide
 * sur mobile. ⚠️ Argon2id est SYNCHRONE (bloque le thread) → ne pas monter la
 * mémoire/passes sans raison, sous peine de figer l'UI sur les appareils lents.
 */
export interface KDFParams {
  algo: 'argon2id'
  t: number // nombre de passes (time cost)
  m: number // mémoire en KiB
  p: number // parallélisme
}

export const DEFAULT_KDF_PARAMS: KDFParams = {
  algo: 'argon2id',
  t: 2,
  m: 19 * 1024, // 19 Mio (minimum OWASP)
  p: 1,
}

/** Deux jeux de paramètres KDF sont-ils identiques ? (détection d'upgrade). */
export function sameKDFParams(a: KDFParams, b: KDFParams): boolean {
  return a.algo === b.algo && a.t === b.t && a.m === b.m && a.p === b.p
}

/**
 * Blob de sauvegarde tel qu'échangé avec le serveur. Tous les champs sont
 * opaques côté serveur (base64 / JSON), produits et consommés ici.
 */
export interface BackupBlob {
  salt: string // base64
  nonce: string // base64
  wrapped_private_key: string // base64
  kdf_params: string // JSON de KDFParams
  public_key: string // base64 (vérification post-déballage)
}

/** Longueur (octets) de la KEK dérivée. Doit rester cohérente avec le worker. */
export const KEK_LEN_BYTES = KEK_LEN

/** Dérive la KEK (32 octets) d'une passphrase + sel via Argon2id. **Synchrone et
 *  coûteux** (bloque le thread) → en production on passe par le Web Worker
 *  (`key-backup-async.ts`) ; cette version reste utilisée par les tests et comme
 *  fallback hors navigateur. */
export function deriveKEK(passphrase: string, salt: Uint8Array, params: KDFParams): Uint8Array {
  return argon2id(passphrase, salt, { t: params.t, m: params.m, p: params.p, dkLen: KEK_LEN })
}

/** Génère un sel aléatoire pour une nouvelle sauvegarde. */
export function newSalt(): Uint8Array {
  return randomBytes(SALT_LEN)
}

/** Lit les paramètres KDF persistés dans un blob. */
export function parseKDFParams(blob: BackupBlob): KDFParams {
  return JSON.parse(blob.kdf_params) as KDFParams
}

/**
 * Emballe `privateKey` à partir d'une KEK **déjà dérivée** (partie rapide,
 * symétrique). Séparée de la dérivation pour pouvoir déléguer Argon2id à un
 * worker. La `publicKey` est jointe pour vérifier la cohérence au déballage.
 */
export function wrapWithKEK(
  kek: Uint8Array,
  privateKey: Uint8Array,
  publicKey: Uint8Array,
  salt: Uint8Array,
  params: KDFParams,
): BackupBlob {
  const { nonce, ciphertext } = encryptSymmetric(kek, privateKey)
  return {
    salt: toBase64(salt),
    nonce: toBase64(nonce),
    wrapped_private_key: toBase64(ciphertext),
    kdf_params: JSON.stringify(params),
    public_key: toBase64(publicKey),
  }
}

/**
 * Déballe un blob à partir d'une KEK **déjà dérivée** (partie rapide). Lève si la
 * KEK est mauvaise (échec d'authentification Poly1305) ou si le blob est altéré.
 */
export function unwrapWithKEK(kek: Uint8Array, blob: BackupBlob): Uint8Array {
  return decryptSymmetric(kek, fromBase64(blob.nonce), fromBase64(blob.wrapped_private_key))
}

/**
 * Construit une sauvegarde chiffrée (dérivation + emballage, **synchrone**).
 * Conservée pour les tests / le fallback ; le chemin de production asynchrone
 * vit dans `key-backup-async.ts`.
 */
export function createBackup(
  privateKey: Uint8Array,
  publicKey: Uint8Array,
  passphrase: string,
  params: KDFParams = DEFAULT_KDF_PARAMS,
): BackupBlob {
  const salt = newSalt()
  const kek = deriveKEK(passphrase, salt, params)
  return wrapWithKEK(kek, privateKey, publicKey, salt, params)
}

/**
 * Déballe une sauvegarde avec `passphrase` → renvoie la clé privée (**synchrone**).
 * Lève si la passphrase est incorrecte (échec d'authentification Poly1305).
 */
export function openBackup(blob: BackupBlob, passphrase: string): Uint8Array {
  const kek = deriveKEK(passphrase, fromBase64(blob.salt), parseKDFParams(blob))
  return unwrapWithKEK(kek, blob)
}
