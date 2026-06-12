import { listHashtagTrends } from '@/lib/posts'
import type { HashtagCandidate } from '@/lib/hashtags'

export async function hashtagSearchGlobal(query: string): Promise<HashtagCandidate[]> {
  const trends = await listHashtagTrends(6, query)
  return trends.map((trend) => ({ tag: trend.tag, count: trend.count }))
}
