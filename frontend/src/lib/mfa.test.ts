import { describe, expect, it } from 'vitest'

import { MfaError } from '@/lib/mfa'

// ─── MfaError ─────────────────────────────────────────────────────────────────

describe('MfaError', () => {
  it('stocke le message', () => {
    const err = new MfaError('TOTP invalide')
    expect(err.message).toBe('TOTP invalide')
  })

  it('est une instance d\'Error', () => {
    const err = new MfaError('test')
    expect(err instanceof Error).toBe(true)
  })

  it('stocke le code optionnel', () => {
    const err = new MfaError('Code invalide', 'invalid_totp')
    expect(err.code).toBe('invalid_totp')
  })

  it('code est undefined si absent', () => {
    const err = new MfaError('Erreur générique')
    expect(err.code).toBeUndefined()
  })
})
