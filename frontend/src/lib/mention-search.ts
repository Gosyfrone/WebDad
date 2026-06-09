'use client'

import { searchUsers } from '@/lib/api'
import type { MentionCandidate } from '@/lib/mentions'
import type { RelationUser } from '@/types'

const MAX_RESULTS = 6

function toCandidate(u: RelationUser): MentionCandidate {
  return { id: u.id, username: u.username, displayName: u.displayName, avatarUrl: u.avatarUrl }
}

/**
 * Recherche globale de candidats (posts / commentaires) : par username
 * (user-service). Vide tant qu'aucun caractère n'est saisi après le « @ ».
 */
export async function mentionSearchGlobal(query: string): Promise<MentionCandidate[]> {
  const q = query.trim()
  if (!q) return []
  const users = await searchUsers('@' + q)
  return users.slice(0, MAX_RESULTS).map(toCandidate)
}

/**
 * Recherche « membres d'abord » (messagerie) : propose en priorité les membres
 * de la conversation (ceux qu'on notifie), puis complète par la recherche
 * globale pour mentionner un non-membre. Sur token vide (« @ » seul), liste les
 * membres.
 */
export function makeMemberFirstSearch(
  members: MentionCandidate[],
): (query: string) => Promise<MentionCandidate[]> {
  return async (query: string) => {
    const q = query.trim().toLowerCase()
    const local = members.filter(
      (m) =>
        !q ||
        m.username.toLowerCase().includes(q) ||
        m.displayName.toLowerCase().includes(q),
    )
    if (!q) return local.slice(0, MAX_RESULTS)

    const ids = new Set(local.map((m) => m.id))
    let global: MentionCandidate[] = []
    try {
      global = (await searchUsers('@' + q)).filter((u) => !ids.has(u.id)).map(toCandidate)
    } catch {
      global = []
    }
    return [...local, ...global].slice(0, MAX_RESULTS)
  }
}
