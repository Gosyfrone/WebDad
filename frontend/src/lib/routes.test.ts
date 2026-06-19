import { describe, expect, it } from 'vitest'

import { ROUTES, profilHref, postHref, hashtagHref, searchHref, navItemsForRole } from '@/lib/routes'

describe('ROUTES', () => {
  it('contient les clés essentielles', () => {
    expect(ROUTES.home).toBe('/')
    expect(ROUTES.login).toBe('/login')
    expect(ROUTES.feed).toBe('/feed')
  })
})

describe('profilHref', () => {
  it('construit /profil/<username>', () => {
    expect(profilHref('alice')).toBe('/profil/alice')
  })

  it('encode les caractères spéciaux', () => {
    expect(profilHref('alice bob')).toBe('/profil/alice%20bob')
  })
})

describe('postHref', () => {
  it('construit /posts/<id>', () => {
    expect(postHref('abc123')).toBe('/posts/abc123')
  })

  it('encode les caractères spéciaux', () => {
    expect(postHref('id/slash')).toBe('/posts/id%2Fslash')
  })
})

describe('hashtagHref', () => {
  it('utilise le tab top par défaut', () => {
    const href = hashtagHref('typescript')
    expect(href).toContain('hashtag=typescript')
    expect(href).toContain('tab=top')
  })

  it('retire le # du tag', () => {
    const href = hashtagHref('#typescript')
    expect(href).toContain('hashtag=typescript')
    expect(href).not.toContain('%23')
  })

  it('accepte un tab personnalisé', () => {
    expect(hashtagHref('go', 'recent')).toContain('tab=recent')
  })
})

describe('searchHref', () => {
  it('retourne explorer pour une chaîne vide', () => {
    expect(searchHref('')).toBe(ROUTES.explorer)
    expect(searchHref('   ')).toBe(ROUTES.explorer)
  })

  it('redirige vers hashtagHref pour un # en début', () => {
    const href = searchHref('#typescript')
    expect(href).toContain('hashtag=typescript')
  })

  it('redirige vers explorer avec ?q= pour une recherche texte', () => {
    const href = searchHref('Go lang')
    expect(href).toContain(ROUTES.explorer)
    expect(href).toContain('q=Go%20lang')
  })
})

describe('navItemsForRole', () => {
  it('retourne les items sans restriction pour tous les rôles', () => {
    const items = navItemsForRole('user')
    expect(items.some((i) => i.href === ROUTES.feed)).toBe(true)
  })

  it('exclut les items modération pour un utilisateur simple', () => {
    const items = navItemsForRole('user')
    expect(items.some((i) => i.href === ROUTES.moderation)).toBe(false)
  })

  it('inclut modération pour moderator', () => {
    const items = navItemsForRole('moderator')
    expect(items.some((i) => i.href === ROUTES.moderation)).toBe(true)
  })

  it('inclut admin pour administrator', () => {
    const items = navItemsForRole('administrator')
    expect(items.some((i) => i.href === ROUTES.admin)).toBe(true)
  })

  it('exclut admin pour moderator', () => {
    const items = navItemsForRole('moderator')
    expect(items.some((i) => i.href === ROUTES.admin)).toBe(false)
  })

  it('retourne uniquement items publics pour role null', () => {
    const items = navItemsForRole(null)
    expect(items.some((i) => i.href === ROUTES.moderation)).toBe(false)
    expect(items.some((i) => i.href === ROUTES.admin)).toBe(false)
  })
})
