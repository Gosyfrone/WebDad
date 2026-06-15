/**
 * Mémoire de navigation de l'onglet « Recherche » (la loupe), façon Instagram :
 * l'onglet retient le dernier chemin de la « section découverte » visité, pour y
 * revenir quand on retape la loupe depuis un autre onglet (messages, fil…).
 *
 * La section = la page Explorer (`/explorer`, avec sa requête `?q=`) et les
 * profils d'autres personnes (`/profil/<username>`). Le profil PERSO (`/profil`,
 * sans username) en est exclu : c'est son propre onglet.
 *
 * Stockage : variable EN MÉMOIRE (module) — fiable y compris en **navigation
 * privée iOS** où `sessionStorage.setItem` lève une erreur. Elle survit aux
 * navigations du SPA (le module reste chargé) ; `sessionStorage` est utilisé EN
 * BONUS quand il est disponible, pour survivre aussi à un rechargement complet.
 */

const STORAGE_KEY = 'breezy.searchTabPath'
const DEFAULT_PATH = '/explorer'

/** Mémoire principale (module) : indépendante de tout stockage navigateur. */
let memoryPath: string | null = null

/** Le chemin appartient-il à la section recherche/découverte ? (pathname seul). */
export function isSearchSectionPath(pathname: string): boolean {
  return pathname === '/explorer' || pathname.startsWith('/profil/')
}

/** Mémorise le dernier chemin (avec sa query) visité dans la section recherche. */
export function rememberSearchPath(pathWithQuery: string): void {
  memoryPath = pathWithQuery
  try {
    sessionStorage.setItem(STORAGE_KEY, pathWithQuery)
  } catch {
    /* sessionStorage indisponible (nav privée iOS) : la mémoire module suffit. */
  }
}

/** Chemin à ouvrir quand on (re)tape la loupe ; `/explorer` par défaut. */
export function getSearchPath(): string {
  if (memoryPath) return memoryPath
  try {
    const stored = sessionStorage.getItem(STORAGE_KEY)
    if (stored) {
      memoryPath = stored
      return stored
    }
  } catch {
    /* sessionStorage indisponible : on retombe sur le défaut. */
  }
  return DEFAULT_PATH
}

export const SEARCH_TAB_DEFAULT = DEFAULT_PATH
