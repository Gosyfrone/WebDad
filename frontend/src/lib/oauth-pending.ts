export const OAUTH_PENDING_STORAGE_KEY = 'breezy-oauth-pending-signup'

export type PendingOAuthSignup = {
  provider: string
  pendingToken: string
  email: string
}

export function savePendingOAuthSignup(pending: PendingOAuthSignup): void {
  if (typeof window === 'undefined') return
  window.sessionStorage.setItem(OAUTH_PENDING_STORAGE_KEY, JSON.stringify(pending))
}

export function loadPendingOAuthSignup(): PendingOAuthSignup | null {
  if (typeof window === 'undefined') return null
  const raw = window.sessionStorage.getItem(OAUTH_PENDING_STORAGE_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as Partial<PendingOAuthSignup>
    if (!parsed.provider || !parsed.pendingToken) return null
    return {
      provider: parsed.provider,
      pendingToken: parsed.pendingToken,
      email: parsed.email ?? '',
    }
  } catch {
    return null
  }
}

export function clearPendingOAuthSignup(): void {
  if (typeof window === 'undefined') return
  window.sessionStorage.removeItem(OAUTH_PENDING_STORAGE_KEY)
}
