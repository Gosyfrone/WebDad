/**
 * Web Worker dédié à la dérivation Argon2id de la KEK (la seule opération
 * coûteuse de la sauvegarde par phrase de passe). L'exécuter hors du thread
 * principal évite de figer l'UI (spinner animé, page réactive) sur les appareils
 * lents — Argon2id en JS pur est synchrone.
 *
 * Il n'importe QUE `@noble/hashes/argon2` (pas tout `crypto.ts`) pour rester un
 * bundle léger. Le wrap/unwrap symétrique (rapide) reste sur le thread principal.
 */

import { argon2id } from '@noble/hashes/argon2'

import type { KDFParams } from './key-backup'

interface DeriveRequest {
  passphrase: string
  salt: Uint8Array
  params: KDFParams
}

type DeriveResponse =
  | { ok: true; kek: Uint8Array }
  | { ok: false; error: string }

// `self` est typé Window via la lib DOM ; on le caste vers Worker (postMessage
// avec liste de transfert + onmessage) pour éviter d'activer la lib webworker.
const ctx = self as unknown as Worker

const KEK_LEN = 32

ctx.onmessage = (event: MessageEvent<DeriveRequest>) => {
  const { passphrase, salt, params } = event.data
  try {
    const kek = argon2id(passphrase, salt, {
      t: params.t,
      m: params.m,
      p: params.p,
      dkLen: KEK_LEN,
    })
    const message: DeriveResponse = { ok: true, kek }
    ctx.postMessage(message, [kek.buffer])
  } catch (err) {
    const message: DeriveResponse = { ok: false, error: String(err) }
    ctx.postMessage(message)
  }
}

export {}
