import { describe, it, expect } from 'vitest'

import {
  CONTENT_KEY_LEN,
  decryptText,
  encryptText,
  fromBase64,
  generateContentKey,
  generateIdentityKeyPair,
  open,
  openKeyEnvelope,
  seal,
  sealKeyForRecipient,
  toBase64,
} from './crypto'

describe('encodage base64', () => {
  it('fait un aller-retour exact', () => {
    const bytes = new Uint8Array([0, 1, 2, 250, 255, 128, 42])
    expect(Array.from(fromBase64(toBase64(bytes)))).toEqual(Array.from(bytes))
  })
})

describe('génération de clés', () => {
  it('produit une paire X25519 de 32 octets', () => {
    const { publicKey, privateKey } = generateIdentityKeyPair()
    expect(publicKey).toHaveLength(32)
    expect(privateKey).toHaveLength(32)
  })

  it('produit une clé de contenu de 256 bits', () => {
    expect(generateContentKey()).toHaveLength(CONTENT_KEY_LEN)
  })

  it('produit des clés différentes à chaque appel', () => {
    expect(toBase64(generateContentKey())).not.toEqual(toBase64(generateContentKey()))
  })
})

describe('sealed box (emballage de clé)', () => {
  it("le destinataire ouvre l'enveloppe et retrouve la clé de contenu", () => {
    const bob = generateIdentityKeyPair()
    const contentKey = generateContentKey()

    const sealed = seal(bob.publicKey, contentKey)
    const opened = open(bob.privateKey, sealed)

    expect(Array.from(opened)).toEqual(Array.from(contentKey))
  })

  it("un tiers (mauvaise clé privée) ne peut PAS ouvrir l'enveloppe", () => {
    const bob = generateIdentityKeyPair()
    const eve = generateIdentityKeyPair()
    const sealed = seal(bob.publicKey, generateContentKey())

    expect(() => open(eve.privateKey, sealed)).toThrow()
  })

  it('rejette une enveloppe altérée (authentification)', () => {
    const bob = generateIdentityKeyPair()
    const sealed = seal(bob.publicKey, generateContentKey())
    sealed[sealed.length - 1] ^= 0xff // corruption du dernier octet

    expect(() => open(bob.privateKey, sealed)).toThrow()
  })

  it('helpers base64 : sealKeyForRecipient / openKeyEnvelope', () => {
    const bob = generateIdentityKeyPair()
    const contentKey = generateContentKey()

    const envelopeB64 = sealKeyForRecipient(toBase64(bob.publicKey), contentKey)
    const opened = openKeyEnvelope(envelopeB64, bob.privateKey)

    expect(Array.from(opened)).toEqual(Array.from(contentKey))
  })
})

describe('chiffrement symétrique des messages', () => {
  it('fait un aller-retour texte avec la clé de contenu', () => {
    const ck = generateContentKey()
    const message = 'Salut 👋 message privé chiffré !'

    const { ciphertext, nonce } = encryptText(ck, message)
    expect(decryptText(ck, ciphertext, nonce)).toBe(message)
  })

  it('produit un nonce différent à chaque chiffrement (non déterministe)', () => {
    const ck = generateContentKey()
    const a = encryptText(ck, 'même texte')
    const b = encryptText(ck, 'même texte')
    expect(a.nonce).not.toEqual(b.nonce)
    expect(a.ciphertext).not.toEqual(b.ciphertext)
  })

  it('une mauvaise clé de contenu ne déchiffre pas', () => {
    const { ciphertext, nonce } = encryptText(generateContentKey(), 'secret')
    expect(() => decryptText(generateContentKey(), ciphertext, nonce)).toThrow()
  })
})

describe('scénario DM bout-en-bout (Alice ↔ Bob)', () => {
  it("Alice scelle la CK pour les deux ; Bob l'ouvre et lit le message d'Alice", () => {
    const alice = generateIdentityKeyPair()
    const bob = generateIdentityKeyPair()

    // Alice crée la clé de contenu et l'emballe pour elle ET pour Bob.
    const contentKey = generateContentKey()
    const envelopeForBob = sealKeyForRecipient(toBase64(bob.publicKey), contentKey)
    const envelopeForAlice = sealKeyForRecipient(toBase64(alice.publicKey), contentKey)

    // Alice envoie un message chiffré avec la CK.
    const sent = encryptText(contentKey, 'On se voit à 18h ?')

    // Bob ouvre SON enveloppe → CK → déchiffre le message.
    const bobCK = openKeyEnvelope(envelopeForBob, bob.privateKey)
    expect(decryptText(bobCK, sent.ciphertext, sent.nonce)).toBe('On se voit à 18h ?')

    // Alice peut elle aussi rouvrir son enveloppe (même CK).
    const aliceCK = openKeyEnvelope(envelopeForAlice, alice.privateKey)
    expect(Array.from(aliceCK)).toEqual(Array.from(bobCK))
  })
})
