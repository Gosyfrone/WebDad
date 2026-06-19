/**
 * Client MFA (TOTP) au-dessus d'`apiFetch` (Bearer + refresh single-flight).
 * Les appels authentifiés vont directement à la gateway (aucun cookie touché,
 * comme les autres data calls). Seuls login + verify passent par le BFF (cookie
 * refresh). À usage CLIENT.
 */
import { apiFetch } from '@/lib/auth-client'

export type MfaStatus = { enabled: boolean; configured: boolean }

export type MfaSetup = {
  secret: string
  otpauth_url: string
  qr_data_uri: string
}

/** Erreur portant le `code` machine renvoyé par l'API (ex. invalid_mfa_code). */
export class MfaError extends Error {
  code?: string
  constructor(message: string, code?: string) {
    super(message)
    this.code = code
  }
}

async function parse<T>(res: Response): Promise<T> {
  const payload = (await res.json().catch(() => null)) as
    | { data?: T; error?: string; code?: string }
    | null
  if (!res.ok) {
    throw new MfaError(payload?.error ?? 'Erreur MFA', payload?.code)
  }
  return (payload?.data as T) ?? ({} as T)
}

export async function getMfaStatus(): Promise<MfaStatus> {
  return parse<MfaStatus>(await apiFetch('/auth/mfa/status'))
}

export async function setupMfa(): Promise<MfaSetup> {
  return parse<MfaSetup>(await apiFetch('/auth/mfa/setup', { method: 'POST' }))
}

export async function enableMfa(code: string): Promise<void> {
  await parse(
    await apiFetch('/auth/mfa/enable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    })
  )
}

export async function disableMfa(proof: {
  code?: string
  password?: string
}): Promise<void> {
  await parse(
    await apiFetch('/auth/mfa/disable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(proof),
    })
  )
}
