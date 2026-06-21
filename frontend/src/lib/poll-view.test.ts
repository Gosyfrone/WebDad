import { describe, expect, it } from 'vitest'

import { formatPollRemaining, isPollClosed, isPollClosedAt, pollResultsView } from '@/lib/poll-view'
import type { PostPoll } from '@/lib/posts'

function makePoll(overrides: Partial<PostPoll> = {}): PostPoll {
  return {
    choices: [],
    endsAt: '2026-01-01T00:00:00Z',
    closedAt: '',
    audience: 'everyone',
    totalVotes: 0,
    votedChoiceId: '',
    winnerChoiceIds: [],
    canViewResults: false,
    canClose: false,
    ...overrides,
  }
}

describe('isPollClosedAt', () => {
  const ends = Date.parse('2026-01-01T00:00:00Z')

  it('est clos quand closedAt est renseigné, même avant la fin', () => {
    expect(isPollClosedAt(makePoll({ closedAt: '2025-12-01T00:00:00Z' }), ends - 100_000)).toBe(true)
  })

  it('est clos quand la date de fin est atteinte ou dépassée', () => {
    expect(isPollClosedAt(makePoll(), ends)).toBe(true)
    expect(isPollClosedAt(makePoll(), ends + 1)).toBe(true)
  })

  it('est ouvert avant la date de fin sans clôture explicite', () => {
    expect(isPollClosedAt(makePoll(), ends - 1)).toBe(false)
  })
})

describe('isPollClosed', () => {
  it('utilise le temps courant', () => {
    expect(isPollClosed(makePoll({ endsAt: '2000-01-01T00:00:00Z' }))).toBe(true)
    expect(isPollClosed(makePoll({ endsAt: '2999-01-01T00:00:00Z' }))).toBe(false)
  })
})

describe('pollResultsView', () => {
  it('reste en vue de vote si on n’a pas le droit de voir les résultats', () => {
    expect(pollResultsView(makePoll({ canViewResults: false, votedChoiceId: 'c1' }), true)).toBe(false)
  })

  it('passe en résultats après un vote', () => {
    expect(pollResultsView(makePoll({ canViewResults: true, votedChoiceId: 'c1' }), false)).toBe(true)
  })

  it('passe en résultats une fois le sondage clos même sans vote', () => {
    expect(pollResultsView(makePoll({ canViewResults: true, votedChoiceId: '' }), true)).toBe(true)
  })

  it('reste en vue de vote si ouvert et non voté', () => {
    expect(pollResultsView(makePoll({ canViewResults: true, votedChoiceId: '' }), false)).toBe(false)
  })
})

describe('formatPollRemaining', () => {
  const base = Date.parse('2026-01-01T00:00:00Z')
  const ends = (deltaSeconds: number) => new Date(base + deltaSeconds * 1000).toISOString()

  it('formate en jours + heures', () => {
    expect(formatPollRemaining(ends(2 * 86400 + 3 * 3600), base)).toBe('2 j 3 h')
  })

  it('formate en heures + minutes', () => {
    expect(formatPollRemaining(ends(5 * 3600 + 12 * 60), base)).toBe('5 h 12 min')
  })

  it('formate en minutes + secondes', () => {
    expect(formatPollRemaining(ends(3 * 60 + 8), base)).toBe('3 min 8 s')
  })

  it('formate en secondes seules sous la minute', () => {
    expect(formatPollRemaining(ends(42), base)).toBe('42 s')
  })

  it('ne descend jamais sous zéro (sondage déjà fini)', () => {
    expect(formatPollRemaining(ends(-100), base)).toBe('0 s')
  })
})
