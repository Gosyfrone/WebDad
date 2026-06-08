/**
 * Stockage de la clé d'identité E2EE de l'utilisateur, **par appareil**.
 *
 * Décision (cf. CLAUDE.md) : la clé privée ne quitte jamais le navigateur. Elle
 * est persistée en **IndexedDB** (et non localStorage : binaire + isolé). Une
 * clé est générée au premier usage de la messagerie sur cet appareil.
 *
 * ⚠️ Limite assumée du « par appareil » : un autre navigateur/appareil aura une
 * autre clé et ne pourra pas déchiffrer l'historique chiffré pour la première.
 * Les messages ne sont pas perdus (le ciphertext reste en base), seulement non
 * déchiffrables ailleurs. Un backup chiffré par phrase secrète est une évolution
 * possible (l'archi le permet) — non implémenté ici.
 *
 * ⚠️ Module CLIENT uniquement (IndexedDB). Les primitives pures sont dans
 * `crypto.ts`.
 */

import {
  generateIdentityKeyPair,
  fromBase64,
  toBase64,
  type KeyPair,
} from '@/lib/crypto'

const DB_NAME = 'breezy-e2ee'
const STORE = 'identity'
const RECORD_KEY = 'self'

interface StoredIdentity {
  publicKey: string // base64
  privateKey: string // base64
}

/** Ouvre (ou crée) la base IndexedDB dédiée aux clés. */
function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1)
    req.onupgradeneeded = () => {
      const db = req.result
      if (!db.objectStoreNames.contains(STORE)) db.createObjectStore(STORE)
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

function readRecord(db: IDBDatabase): Promise<StoredIdentity | null> {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, 'readonly')
    const req = tx.objectStore(STORE).get(RECORD_KEY)
    req.onsuccess = () => resolve((req.result as StoredIdentity | undefined) ?? null)
    req.onerror = () => reject(req.error)
  })
}

function writeRecord(db: IDBDatabase, record: StoredIdentity): Promise<void> {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, 'readwrite')
    tx.objectStore(STORE).put(record, RECORD_KEY)
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
}

function decode(record: StoredIdentity): KeyPair {
  return {
    publicKey: fromBase64(record.publicKey),
    privateKey: fromBase64(record.privateKey),
  }
}

/** Renvoie l'identité stockée sur cet appareil, ou null si absente. */
export async function getStoredIdentity(): Promise<KeyPair | null> {
  if (typeof indexedDB === 'undefined') return null
  const db = await openDB()
  try {
    const record = await readRecord(db)
    return record ? decode(record) : null
  } finally {
    db.close()
  }
}

/**
 * Charge l'identité de cet appareil ou en crée une nouvelle (persistée) au
 * premier appel. Idempotent : les appels suivants renvoient la même paire.
 */
export async function loadOrCreateIdentity(): Promise<KeyPair> {
  const db = await openDB()
  try {
    const existing = await readRecord(db)
    if (existing) return decode(existing)

    const keyPair = generateIdentityKeyPair()
    await writeRecord(db, {
      publicKey: toBase64(keyPair.publicKey),
      privateKey: toBase64(keyPair.privateKey),
    })
    return keyPair
  } finally {
    db.close()
  }
}
