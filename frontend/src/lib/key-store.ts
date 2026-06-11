/**
 * Stockage de la clé d'identité E2EE de l'utilisateur, **par compte et par
 * appareil**.
 *
 * Décision (cf. CLAUDE.md) : la clé privée ne quitte jamais le navigateur. Elle
 * est persistée en **IndexedDB** (et non localStorage : binaire + isolé), sous
 * une clé d'enregistrement **dérivée du `userId`** → plusieurs comptes sur le
 * même navigateur n'écrasent plus leur clé mutuellement.
 *
 * Multi-appareils : la clé peut désormais suivre l'utilisateur via la
 * **sauvegarde chiffrée par phrase de passe** (`key-backup.ts`) — un autre
 * appareil restaure la même clé après déblocage. Sans sauvegarde, un nouvel
 * appareil repart d'une clé neuve (l'historique antérieur reste illisible).
 *
 * ⚠️ Module CLIENT uniquement (IndexedDB). Les primitives pures sont dans
 * `crypto.ts`.
 */

import { fromBase64, toBase64, type KeyPair } from '@/lib/crypto'

const DB_NAME = 'breezy-e2ee'
const STORE = 'identity'
/** Ancienne clé d'enregistrement GLOBALE (avant le scoping par compte). */
const LEGACY_RECORD_KEY = 'self'

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

function readRecord(db: IDBDatabase, key: string): Promise<StoredIdentity | null> {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, 'readonly')
    const req = tx.objectStore(STORE).get(key)
    req.onsuccess = () => resolve((req.result as StoredIdentity | undefined) ?? null)
    req.onerror = () => reject(req.error)
  })
}

function writeRecord(db: IDBDatabase, key: string, record: StoredIdentity): Promise<void> {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, 'readwrite')
    tx.objectStore(STORE).put(record, key)
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
}

function encode(keyPair: KeyPair): StoredIdentity {
  return {
    publicKey: toBase64(keyPair.publicKey),
    privateKey: toBase64(keyPair.privateKey),
  }
}

function decode(record: StoredIdentity): KeyPair {
  return {
    publicKey: fromBase64(record.publicKey),
    privateKey: fromBase64(record.privateKey),
  }
}

/** Renvoie l'identité stockée pour CE compte sur cet appareil, ou null. */
export async function getStoredIdentity(userId: string): Promise<KeyPair | null> {
  if (typeof indexedDB === 'undefined' || !userId) return null
  const db = await openDB()
  try {
    const record = await readRecord(db, userId)
    return record ? decode(record) : null
  } finally {
    db.close()
  }
}

/**
 * Persiste l'identité de CE compte sur cet appareil (créée à la définition de la
 * phrase de passe, ou restaurée depuis la sauvegarde chiffrée au déblocage).
 */
export async function setStoredIdentity(userId: string, keyPair: KeyPair): Promise<void> {
  if (!userId) return
  const db = await openDB()
  try {
    await writeRecord(db, userId, encode(keyPair))
  } finally {
    db.close()
  }
}

/**
 * Lit l'ancienne clé NON scopée (`self`) issue du stockage d'avant le scoping
 * par compte. Sert uniquement à la migration ponctuelle (cf. messages.ts) : on
 * ne l'adopte pour un compte que si sa clé publique correspond à celle publiée.
 */
export async function getLegacyIdentity(): Promise<KeyPair | null> {
  if (typeof indexedDB === 'undefined') return null
  const db = await openDB()
  try {
    const record = await readRecord(db, LEGACY_RECORD_KEY)
    return record ? decode(record) : null
  } finally {
    db.close()
  }
}
