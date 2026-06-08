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

describe('scénario GROUPE (clé unique, Option A : historique complet)', () => {
  it("un membre invite un nouveau venu qui lit TOUT l'historique + le nom chiffré", () => {
    const owner = generateIdentityKeyPair()
    const member = generateIdentityKeyPair()
    const invited = generateIdentityKeyPair() // ajouté plus tard

    // L'owner crée la clé de contenu du groupe et chiffre le NOM avec.
    const groupKey = generateContentKey()
    const name = encryptText(groupKey, 'Projet WebDad 🚀')
    // …et emballe la clé pour lui + le membre initial.
    const envOwner = sealKeyForRecipient(toBase64(owner.publicKey), groupKey)
    const envMember = sealKeyForRecipient(toBase64(member.publicKey), groupKey)

    // Messages envoyés AVANT l'arrivée de l'invité.
    const m1 = encryptText(groupKey, 'message #1')
    const m2 = encryptText(groupKey, 'message #2')

    // « Tout le monde peut inviter » : le MEMBRE (pas l'owner) ouvre sa clé et
    // l'emballe pour l'invité — il n'a besoin que de la clé publique de l'invité.
    const memberKey = openKeyEnvelope(envMember, member.privateKey)
    const envInvited = sealKeyForRecipient(toBase64(invited.publicKey), memberKey)

    // L'invité ouvre SON enveloppe → clé du groupe → lit l'historique d'avant.
    const invitedKey = openKeyEnvelope(envInvited, invited.privateKey)
    expect(decryptText(invitedKey, m1.ciphertext, m1.nonce)).toBe('message #1')
    expect(decryptText(invitedKey, m2.ciphertext, m2.nonce)).toBe('message #2')
    // …et déchiffre le nom du groupe.
    expect(decryptText(invitedKey, name.ciphertext, name.nonce)).toBe('Projet WebDad 🚀')

    // Cohérence : owner, membre et invité partagent la MÊME clé.
    const ownerKey = openKeyEnvelope(envOwner, owner.privateKey)
    expect(Array.from(invitedKey)).toEqual(Array.from(ownerKey))
    expect(Array.from(memberKey)).toEqual(Array.from(ownerKey))
  })
})

describe('scénario COMMUNAUTÉ (clé détenue par le serveur, hybride)', () => {
  it("la clé remise par le serveur déchiffre l'historique pour un viewer qui rejoint", () => {
    // L'owner génère la clé et la CONFIE au serveur (transitée en base64).
    const communityKey = generateContentKey()
    const keyHandedByServer = toBase64(communityKey) // ce que le back stocke/redistribue

    // Des messages sont postés par des talkers.
    const m1 = encryptText(communityKey, 'Bienvenue dans la communauté')
    const m2 = encryptText(communityKey, 'Règle n°1 : soyez sympas')

    // Un viewer rejoint : le serveur lui remet la clé → il la décode et lit tout.
    const viewerKey = fromBase64(keyHandedByServer)
    expect(decryptText(viewerKey, m1.ciphertext, m1.nonce)).toBe('Bienvenue dans la communauté')
    expect(decryptText(viewerKey, m2.ciphertext, m2.nonce)).toBe('Règle n°1 : soyez sympas')
    // Même clé que l'originale (aller-retour base64 fidèle).
    expect(Array.from(viewerKey)).toEqual(Array.from(communityKey))
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
