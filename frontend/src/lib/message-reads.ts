/**
 * État « lu / non-lu » de la messagerie, **côté client uniquement**.
 *
 * Le back ne gère pas d'accusés de lecture : on mémorise donc, par conversation,
 * l'id du dernier message « vu » dans `localStorage` (PAR APPAREIL, cohérent avec
 * la clé d'identité par appareil). Sert à :
 *   - la pastille « non-lu » dans la liste des conversations ;
 *   - la ligne « Nouveaux messages » dans le fil (ancrée sur le dernier message
 *     lu, capturée à l'ouverture).
 *
 * Les ids de message sont des ObjectId Mongo (préfixe horodaté) : la comparaison
 * lexicographique reflète l'ordre de création (« plus récent » = id plus grand).
 */

const STORAGE_KEY = 'breezy-msg-reads'

function readAll(): Record<string, string> {
  if (typeof window === 'undefined') return {}
  try {
    return JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? '{}') as Record<string, string>
  } catch {
    return {}
  }
}

/** Id du dernier message lu d'une conversation (null si jamais ouverte ici). */
export function getLastRead(conversationId: string): string | null {
  return readAll()[conversationId] ?? null
}

/**
 * Avance le marqueur de lecture d'une conversation (ne recule jamais : on ne
 * réécrit que si l'id fourni est plus récent que celui mémorisé).
 */
export function setLastRead(conversationId: string, messageId: string): void {
  if (typeof window === 'undefined' || !messageId) return
  const all = readAll()
  const current = all[conversationId]
  if (current && current >= messageId) return
  all[conversationId] = messageId
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(all))
}
