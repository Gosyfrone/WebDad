export const TERMS_READ_STORAGE_KEY = 'breezy-cgu-read'

export function hasReadTerms(): boolean {
  if (typeof window === 'undefined') return false
  return window.localStorage.getItem(TERMS_READ_STORAGE_KEY) === 'true'
}

export function markTermsRead(): void {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(TERMS_READ_STORAGE_KEY, 'true')
}
