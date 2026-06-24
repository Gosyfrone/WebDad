import { afterEach, describe, expect, it, vi } from 'vitest'

const searchUsers = vi.fn()
vi.mock('@/lib/api', () => ({ searchUsers: (...a: unknown[]) => searchUsers(...a) }))

import { mentionSearchGlobal, makeMemberFirstSearch } from '@/lib/mention-search'
import type { MentionCandidate } from '@/lib/mentions'

afterEach(() => vi.clearAllMocks())

const rel = (id: string, username: string) => ({
  id,
  username,
  displayName: username.toUpperCase(),
  avatarUrl: '',
  bio: '',
  certification: 'none',
})
const member = (id: string, username: string): MentionCandidate => ({
  id,
  username,
  displayName: username,
  avatarUrl: '',
  certification: 'none',
})

describe('mentionSearchGlobal (réseau)', () => {
  it('cherche par @username, mappe et plafonne à 6', async () => {
    searchUsers.mockResolvedValue(Array.from({ length: 8 }, (_, i) => rel('u' + i, 'al' + i)))
    const res = await mentionSearchGlobal('al')
    expect(res).toHaveLength(6)
    expect(searchUsers).toHaveBeenCalledWith('@al')
    expect(res[0]).toEqual({ id: 'u0', username: 'al0', displayName: 'AL0', avatarUrl: '', certification: 'none' })
  })
})

describe('makeMemberFirstSearch (réseau)', () => {
  const members = [member('m1', 'alice'), member('m2', 'bob')]

  it('membres d’abord puis complète par le global (sans doublon)', async () => {
    searchUsers.mockResolvedValue([rel('m1', 'alice'), rel('x9', 'alien')])
    const res = await makeMemberFirstSearch(members)('al')
    expect(res.map((m) => m.id)).toEqual(['m1', 'x9'])
  })

  it('recherche globale en échec → seulement les membres', async () => {
    searchUsers.mockRejectedValue(new Error('net'))
    const res = await makeMemberFirstSearch(members)('ali')
    expect(res.map((m) => m.id)).toEqual(['m1'])
  })
})
