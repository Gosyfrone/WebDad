import { describe, it, expect } from 'vitest'

import { getSearchPath, isSearchSectionPath, noteSearchNavigation } from './search-tab'

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

describe('noteSearchNavigation (mémoire loupe consciente de l’origine)', () => {
  it('ignore un profil atteint depuis le fil (ne pollue pas la loupe)', () => {
    // Une vraie recherche d’abord, pour avoir une mémoire à préserver.
    noteSearchNavigation('/explorer', '/explorer?q=jane')
    expect(getSearchPath()).toBe('/explorer?q=jane')

    // Fil → profil → retour fil : le profil NE doit PAS devenir la cible loupe.
    noteSearchNavigation('/feed', '/feed')
    noteSearchNavigation('/profil/bob', '/profil/bob')
    noteSearchNavigation('/feed', '/feed')
    expect(getSearchPath()).toBe('/explorer?q=jane')
  })

  it('mémorise un profil atteint en prolongeant la recherche', () => {
    noteSearchNavigation('/explorer', '/explorer?q=alice')
    noteSearchNavigation('/profil/alice', '/profil/alice')
    expect(getSearchPath()).toBe('/profil/alice')
  })

  it('le profil perso ne fait jamais partie de la section', () => {
    noteSearchNavigation('/explorer', '/explorer?q=zoe')
    noteSearchNavigation('/profil', '/profil')
    expect(getSearchPath()).toBe('/explorer?q=zoe')
  })
})
