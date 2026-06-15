import { describe, it, expect } from 'vitest'

import { isSearchSectionPath } from './search-tab'

describe('isSearchSectionPath (section recherche/découverte)', () => {
  it('inclut Explorer et les profils d’autres personnes', () => {
    expect(isSearchSectionPath('/explorer')).toBe(true)
    expect(isSearchSectionPath('/profil/john')).toBe(true)
  })

  it('exclut le profil perso, le fil, les messages, les notifications', () => {
    expect(isSearchSectionPath('/profil')).toBe(false) // profil perso = son propre onglet
    expect(isSearchSectionPath('/feed')).toBe(false)
    expect(isSearchSectionPath('/messages')).toBe(false)
    expect(isSearchSectionPath('/notifications')).toBe(false)
  })
})
