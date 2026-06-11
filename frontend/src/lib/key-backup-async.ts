/**
 * Chemin de production (CLIENT) de la sauvegarde par phrase de passe : la
 * dérivation Argon2id est déléguée à un **Web Worker** (`key-backup.worker.ts`)
 * pour ne pas figer le thread principal. L'emballage/déballage symétrique
 * (rapide) reste synchrone.
 *
 * Fallback : si `Worker` est indisponible (SSR, environnement de test), on
 * retombe sur la dérivation synchrone de `key-backup.ts`.
 */

import { fromBase64 } from '@/lib/crypto'
import {
  DEFAULT_KDF_PARAMS,
  deriveKEK,
  newSalt,
  parseKDFParams,
  unwrapWithKEK,
  wrapWithKEK,
  type BackupBlob,
  type KDFParams,
} from '@/lib/key-backup'

/** Dérive la KEK dans un worker (un worker éphémère par appel, terminé ensuite). */
function deriveKEKInWorker(
  passphrase: string,
  salt: Uint8Array,
  params: KDFParams,
): Promise<Uint8Array> {
  return new Promise((resolve, reject) => {
    const worker = new Worker(new URL('./key-backup.worker.ts', import.meta.url))
    worker.onmessage = (event: MessageEvent<{ ok: true; kek: Uint8Array } | { ok: false; error: string }>) => {
      worker.terminate()
      if (event.data.ok) resolve(new Uint8Array(event.data.kek))
      else reject(new Error(event.data.error))
    }
    worker.onerror = (event) => {
      worker.terminate()
      reject(event.error instanceof Error ? event.error : new Error('échec du worker KDF'))
    }
    worker.postMessage({ passphrase, salt, params })
  })
}

/** Dérive la KEK hors thread principal si possible, sinon en synchrone. */
async function deriveKEKAsync(
  passphrase: string,
  salt: Uint8Array,
  params: KDFParams,
): Promise<Uint8Array> {
  if (typeof Worker === 'undefined') return deriveKEK(passphrase, salt, params)
  try {
    return await deriveKEKInWorker(passphrase, salt, params)
  } catch {
    // Worker indisponible/échoué (CSP, bundling…) → fallback synchrone.
    return deriveKEK(passphrase, salt, params)
  }
}

/** Construit une sauvegarde chiffrée (Argon2id en worker + emballage rapide). */
export async function createBackupAsync(
  privateKey: Uint8Array,
  publicKey: Uint8Array,
  passphrase: string,
  params: KDFParams = DEFAULT_KDF_PARAMS,
): Promise<BackupBlob> {
  const salt = newSalt()
  const kek = await deriveKEKAsync(passphrase, salt, params)
  return wrapWithKEK(kek, privateKey, publicKey, salt, params)
}

/** Déballe une sauvegarde (Argon2id en worker + déballage rapide). Lève si la
 *  passphrase est incorrecte (échec d'authentification au déballage). */
export async function openBackupAsync(blob: BackupBlob, passphrase: string): Promise<Uint8Array> {
  const kek = await deriveKEKAsync(passphrase, fromBase64(blob.salt), parseKDFParams(blob))
  return unwrapWithKEK(kek, blob)
}
