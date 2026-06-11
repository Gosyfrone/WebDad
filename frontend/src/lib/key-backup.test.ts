import { describe, it, expect } from 'vitest'

import { generateIdentityKeyPair, derivePublicKey, toBase64 } from './crypto'
import {
  createBackup,
  openBackup,
  deriveKEK,
  sameKDFParams,
  DEFAULT_KDF_PARAMS,
} from './key-backup'

// Paramètres Argon2id allégés pour des tests rapides (la sécurité réelle vient
// des DEFAULT_KDF_PARAMS en production).
const FAST_PARAMS = { algo: 'argon2id' as const, t: 1, m: 8, p: 1 }

describe('deriveKEK', () => {
  it('est déterministe pour la même passphrase + sel', () => {
    const salt = new Uint8Array(16).fill(7)
    const a = deriveKEK('phrase-de-passe-solide', salt, FAST_PARAMS)
    const b = deriveKEK('phrase-de-passe-solide', salt, FAST_PARAMS)
    expect(toBase64(a)).toBe(toBase64(b))
    expect(a).toHaveLength(32)
  })

  it('diffère pour une passphrase différente', () => {
    const salt = new Uint8Array(16).fill(7)
    const a = deriveKEK('phrase-A', salt, FAST_PARAMS)
    const b = deriveKEK('phrase-B', salt, FAST_PARAMS)
    expect(toBase64(a)).not.toBe(toBase64(b))
  })
})

describe('sauvegarde chiffrée de la clé privée', () => {
  it('fait un aller-retour : openBackup restaure la clé privée', () => {
    const id = generateIdentityKeyPair()
    const blob = createBackup(id.privateKey, id.publicKey, 'ma-phrase-secrete-42', FAST_PARAMS)
    const restored = openBackup(blob, 'ma-phrase-secrete-42')
    expect(toBase64(restored)).toBe(toBase64(id.privateKey))
    // La clé publique recalculée correspond bien à l'originale.
    expect(toBase64(derivePublicKey(restored))).toBe(toBase64(id.publicKey))
  })

  it('rejette une mauvaise passphrase (échec d’authentification)', () => {
    const id = generateIdentityKeyPair()
    const blob = createBackup(id.privateKey, id.publicKey, 'bonne-phrase', FAST_PARAMS)
    expect(() => openBackup(blob, 'mauvaise-phrase')).toThrow()
  })

  it('persiste les paramètres KDF dans le blob', () => {
    const id = generateIdentityKeyPair()
    const blob = createBackup(id.privateKey, id.publicKey, 'phrase', FAST_PARAMS)
    expect(JSON.parse(blob.kdf_params)).toEqual(FAST_PARAMS)
  })

  it('produit un sel et un nonce différents à chaque sauvegarde', () => {
    const id = generateIdentityKeyPair()
    const a = createBackup(id.privateKey, id.publicKey, 'phrase', FAST_PARAMS)
    const b = createBackup(id.privateKey, id.publicKey, 'phrase', FAST_PARAMS)
    expect(a.salt).not.toBe(b.salt)
    expect(a.nonce).not.toBe(b.nonce)
  })

  it('expose des paramètres par défaut conformes au minimum OWASP (≥19 Mio, ≥2 passes)', () => {
    expect(DEFAULT_KDF_PARAMS.m).toBeGreaterThanOrEqual(19 * 1024)
    expect(DEFAULT_KDF_PARAMS.t).toBeGreaterThanOrEqual(2)
    expect(DEFAULT_KDF_PARAMS.algo).toBe('argon2id')
  })
})

describe('sameKDFParams (détection d’upgrade)', () => {
  it('égalité stricte sur algo/t/m/p', () => {
    expect(sameKDFParams(DEFAULT_KDF_PARAMS, { ...DEFAULT_KDF_PARAMS })).toBe(true)
  })

  it('détecte une sauvegarde plus lourde à ré-emballer', () => {
    const heavy = { algo: 'argon2id' as const, t: 3, m: 64 * 1024, p: 1 }
    expect(sameKDFParams(heavy, DEFAULT_KDF_PARAMS)).toBe(false)
  })
})
