import { describe, expect, it, afterEach } from 'vitest'

import { apiUrl, API_URL } from '@/lib/config'

describe('API_URL', () => {
  it('est défini', () => {
    expect(typeof API_URL).toBe('string')
    expect(API_URL.length).toBeGreaterThan(0)
  })
})

describe('apiUrl', () => {
  afterEach(() => {
    delete process.env.API_INTERNAL_URL
  })

  it('préfixe un chemin relatif avec un slash', () => {
    const url = apiUrl('auth/login')
    expect(url).toMatch(/\/auth\/login$/)
  })

  it('ne double pas le slash si le chemin commence déjà par /', () => {
    const url = apiUrl('/auth/login')
    expect(url).not.toMatch(/\/\/auth/)
    expect(url).toMatch(/\/auth\/login$/)
  })

  it('utilise API_INTERNAL_URL côté serveur si définie', () => {
    process.env.API_INTERNAL_URL = 'http://api-gateway:8080'
    const url = apiUrl('/health')
    expect(url).toBe('http://api-gateway:8080/health')
  })

  it('revient sur API_URL si API_INTERNAL_URL est absente', () => {
    delete process.env.API_INTERNAL_URL
    const url = apiUrl('/health')
    expect(url).toMatch(/\/health$/)
  })
})
