/**
 * Client média (upload / résolution d'URL) au-dessus d'`apiFetch`.
 *
 * Le stockage des fichiers est assuré par le `media-service` (adossé à MinIO),
 * atteint UNIQUEMENT via l'API Gateway (cf. CLAUDE.md §1) :
 *   - upload    : `POST /media`  (authentifié, multipart)
 *   - download  : `GET /media/<id>` (public, l'id est non devinable)
 *
 * Le service est AGNOSTIQUE du contenu : il stocke des octets opaques. Pour les
 * profils/posts ce sont des images/vidéos en clair ; pour la messagerie E2EE,
 * on lui confie des octets DÉJÀ chiffrés côté client (cf. `uploadEncryptedMedia`)
 * → le serveur ne peut rien déchiffrer.
 */

import { apiFetch } from '@/lib/auth-client'
import { API_URL } from '@/lib/config'

/** Métadonnées renvoyées par le service après un upload. */
export interface UploadedMedia {
  id: string
  url: string // chemin relatif gateway, ex. "/media/<id>"
  mime: string
  kind: 'image' | 'video'
  size: number
}

/**
 * Construit l'URL absolue (côté navigateur) d'un média servi par la gateway.
 * On stocke l'`id` (et non une URL absolue) : un `<img src="/media/x">` taperait
 * le front (:3000) au lieu de la gateway (:8080) → on préfixe explicitement.
 */
export function mediaUrl(id: string): string {
  return `${API_URL}/media/${id}`
}

/**
 * Résout une valeur média STOCKÉE (lue depuis le profil-service, etc.) en URL
 * absolue prête pour un `<img src>`. À appliquer dans les mappers API→modèle.
 *   - vide                       → ''
 *   - data:/blob:/http(s):       → tel quel (legacy / externe)
 *   - "/media/<id>" ou "<id>"    → préfixée de l'URL de la gateway
 */
export function resolveMediaUrl(stored: string | null | undefined): string {
  if (!stored) return ''
  if (/^(https?:|data:|blob:)/.test(stored)) return stored
  const path = stored.startsWith('/') ? stored : `/media/${stored}`
  return `${API_URL}${path}`
}

/**
 * Inverse de `resolveMediaUrl` pour la PERSISTANCE : ne stocke que le chemin
 * relatif (`/media/<id>`), portable d'un environnement à l'autre (on ne fige
 * pas l'hôte de la gateway en base). Les valeurs externes/legacy passent tel quel.
 */
export function toStoredMedia(url: string | null | undefined): string {
  if (!url) return ''
  if (url.startsWith(`${API_URL}/media/`)) return url.slice(API_URL.length)
  return url
}

/**
 * Uploade un fichier (image/vidéo en clair) et renvoie ses métadonnées.
 * On ne fixe PAS le Content-Type : le navigateur pose lui-même le boundary
 * multipart (`apiFetch` n'ajoute que le Bearer).
 */
export async function uploadMedia(file: File | Blob): Promise<UploadedMedia> {
  const form = new FormData()
  form.append('file', file)

  const res = await apiFetch('/media', { method: 'POST', body: form })
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null
    throw new Error(body?.error ?? `échec de l'upload (${res.status})`)
  }
  const body = (await res.json()) as { data: UploadedMedia }
  return body.data
}

/**
 * Uploade un blob DÉJÀ chiffré (pièce jointe de messagerie E2EE). Identique à
 * `uploadMedia` mais le service ne voit que du ciphertext ; on ne récupère que
 * l'`id` (le mime/kind réel reste dans l'enveloppe chiffrée du message).
 */
export async function uploadEncryptedMedia(blob: Blob): Promise<{ id: string }> {
  const form = new FormData()
  form.append('file', blob)

  const res = await apiFetch('/media/encrypted', { method: 'POST', body: form })
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null
    throw new Error(body?.error ?? `échec de l'upload (${res.status})`)
  }
  const body = (await res.json()) as { data: { id: string } }
  return { id: body.data.id }
}

/**
 * Télécharge le contenu brut d'un média (utilisé pour les pièces jointes
 * chiffrées : on récupère le ciphertext puis on déchiffre côté client).
 */
export async function fetchMediaBytes(id: string): Promise<Uint8Array> {
  const res = await apiFetch(`/media/${id}`)
  if (!res.ok) throw new Error(`média introuvable (${res.status})`)
  const buf = await res.arrayBuffer()
  return new Uint8Array(buf)
}
