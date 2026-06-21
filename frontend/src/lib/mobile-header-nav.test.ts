import { describe, expect, it } from 'vitest'

import { resolveMobileHeaderNav } from '@/lib/mobile-header-nav'
import { ROUTES } from '@/lib/routes'

describe('resolveMobileHeaderNav', () => {
  it('reconnaît le feed (égalité stricte) sans titre, avec cloche', () => {
    const nav = resolveMobileHeaderNav(ROUTES.feed)
    expect(nav.section).toBe('feed')
    expect(nav.titleKey).toBeNull()
    expect(nav.showBell).toBe(true)
    expect(nav.showBack).toBe(false)
  })

  it('ne classe pas un sous-chemin du feed comme feed (égalité stricte)', () => {
    // `/feed` est une égalité, pas un préfixe → un sous-chemin retombe sur other.
    expect(resolveMobileHeaderNav(`${ROUTES.feed}/123`).section).toBe('other')
  })

  it.each([
    [ROUTES.explorer, 'explorer', 'nav.explore', true, false],
    [ROUTES.messages, 'messages', 'messages.title', true, false],
    [ROUTES.notifications, 'notifications', 'notifications.title', false, true],
    [ROUTES.parametres, 'parametres', 'settings.title', false, true],
    [ROUTES.moderation, 'moderation', 'nav.moderation', true, false],
    [ROUTES.admin, 'admin', 'nav.admin', true, false],
    [ROUTES.bookmarks, 'signets', 'bookmarks.title', true, false],
  ] as const)(
    'classe %s en section %s',
    (path, section, titleKey, showBell, showBack) => {
      const nav = resolveMobileHeaderNav(path)
      expect(nav.section).toBe(section)
      expect(nav.titleKey).toBe(titleKey)
      expect(nav.showBell).toBe(showBell)
      expect(nav.showBack).toBe(showBack)
    },
  )

  it('reconnaît les sous-chemins par préfixe (ex. notification ciblée)', () => {
    expect(resolveMobileHeaderNav(`${ROUTES.notifications}/abc`).section).toBe('notifications')
    expect(resolveMobileHeaderNav(`${ROUTES.bookmarks}/coll-1`).section).toBe('signets')
  })

  it('retombe sur other (logo, pas de cloche ni flèche) pour un chemin inconnu', () => {
    const nav = resolveMobileHeaderNav('/profil/alice')
    expect(nav.section).toBe('other')
    expect(nav.titleKey).toBeNull()
    expect(nav.showBell).toBe(false)
    expect(nav.showBack).toBe(false)
  })

  it('gère null / chaîne vide comme other', () => {
    expect(resolveMobileHeaderNav(null).section).toBe('other')
    expect(resolveMobileHeaderNav(undefined).section).toBe('other')
    expect(resolveMobileHeaderNav('').section).toBe('other')
  })
})
