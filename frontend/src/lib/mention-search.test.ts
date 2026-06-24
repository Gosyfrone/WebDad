import { describe, expect, it } from 'vitest'

import { mentionSearchGlobal, makeMemberFirstSearch } from '@/lib/mention-search'
import type { MentionCandidate } from '@/lib/mentions'

function makeCandidate(id: string, username: string): MentionCandidate {
  return { id, username, displayName: username, avatarUrl: '', certification: 'none' }
}

// ─── mentionSearchGlobal ─────────────────────────────────────────────────────

describe('mentionSearchGlobal', () => {
  it('retourne [] pour une chaîne vide sans appel réseau', async () => {
    const results = await mentionSearchGlobal('')
    expect(results).toEqual([])
  })

  it('retourne [] pour des espaces seuls', async () => {
    const results = await mentionSearchGlobal('   ')
    expect(results).toEqual([])
  })
})

// ─── makeMemberFirstSearch ───────────────────────────────────────────────────

describe('makeMemberFirstSearch', () => {
  const members: MentionCandidate[] = [
    makeCandidate('1', 'alice'),
    makeCandidate('2', 'bob'),
    makeCandidate('3', 'carol'),
  ]

  it('retourne tous les membres pour une query vide (pas appel réseau)', async () => {
    const search = makeMemberFirstSearch(members)
    const results = await search('')
    expect(results).toHaveLength(members.length)
    expect(results.map((r) => r.id)).toEqual(['1', '2', '3'])
  })

  it('filtre les membres par username', async () => {
    const search = makeMemberFirstSearch(members)
    const results = await search('ali')
    expect(results.some((r) => r.username === 'alice')).toBe(true)
    expect(results.some((r) => r.username === 'bob')).toBe(false)
  })

  it('filtre par displayName aussi', async () => {
    const withDisplay: MentionCandidate[] = [
      { id: '4', username: 'jdupont', displayName: 'Jean Dupont', avatarUrl: '', certification: 'none' },
    ]
    const search = makeMemberFirstSearch(withDisplay)
    const results = await search('Jean')
    expect(results.some((r) => r.id === '4')).toBe(true)
  })

  it('plafonne à 6 résultats', async () => {
    const many = Array.from({ length: 10 }, (_, i) =>
      makeCandidate(String(i), `user${i}`),
    )
    const search = makeMemberFirstSearch(many)
    const results = await search('')
    expect(results.length).toBeLessThanOrEqual(6)
  })

  it('retourne [] si aucun membre et query vide', async () => {
    const search = makeMemberFirstSearch([])
    expect(await search('')).toEqual([])
  })
})
