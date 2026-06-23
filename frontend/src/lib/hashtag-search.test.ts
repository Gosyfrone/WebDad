import { afterEach, describe, expect, it, vi } from 'vitest'

const listHashtagTrends = vi.fn()
vi.mock('@/lib/posts', () => ({ listHashtagTrends: (...a: unknown[]) => listHashtagTrends(...a) }))

import { hashtagSearchGlobal } from '@/lib/hashtag-search'

afterEach(() => vi.clearAllMocks())

describe('hashtagSearchGlobal', () => {
  it('mappe les tendances en candidats (tag + count) et propage la requête', async () => {
    listHashtagTrends.mockResolvedValue([{ tag: 'js', count: 5 }, { tag: 'go', count: 3 }])
    expect(await hashtagSearchGlobal('j')).toEqual([{ tag: 'js', count: 5 }, { tag: 'go', count: 3 }])
    expect(listHashtagTrends).toHaveBeenCalledWith(6, 'j')
  })
})
