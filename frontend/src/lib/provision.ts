import { apiUrl } from '@/lib/config'

/**
 * Provisioning paresseux du user via l'email : `GET /users/me` crée la ligne
 * `users` (handle dérivé de l'email). Sert de repli (connexion, ou course au
 * register). NB : le profil (profil-service) n'a PAS de provisioning paresseux
 * — sa seule création est `POST /profils` (cf. provisionProfil), appelé au
 * register. Pour un compte sans profil, GET /profils/me renvoie 404 et le front
 * crée le profil via POST depuis la popup d'édition.
 */
async function provisionFromEmail(token: string): Promise<void> {
  try {
    await fetch(apiUrl('/users/me'), {
      headers: { Authorization: `Bearer ${token}` },
      cache: 'no-store',
    })
  } catch {
    // Silencieux : provisioning différé au prochain /users/me.
  }
}

/** POST /profils { display_name } : pose le nom affiché = username au register. */
async function provisionProfil(token: string, displayName: string): Promise<void> {
  await fetch(apiUrl('/profils'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ display_name: displayName }),
    cache: 'no-store',
  })
  // 409 (profil déjà créé) est sans gravité : best-effort, on n'agit pas dessus.
}

/**
 * Provisioning de l'utilisateur après login / register, dans user-service ET
 * profil-service (un compte = une identité + un profil).
 *
 *   - Avec `username` (inscription) : `POST /users { username }` (handle choisi)
 *     puis `POST /profils { display_name: username }` (nom affiché = handle).
 *     La disponibilité du username est pré-vérifiée côté page register ; en cas
 *     de course rarissime, on retombe sur les valeurs dérivées de l'email.
 *   - Sans `username` (connexion) : `GET /users/me` + `GET /profils/me`
 *     (dérivés email / idempotents).
 *
 * Best-effort : une erreur ici ne doit JAMAIS casser l'authentification.
 * À usage serveur uniquement (route handlers) : le token n'est pas exposé au client.
 */
export async function provisionUser(
  token: string,
  options?: { username?: string }
): Promise<void> {
  const username = options?.username?.trim()
  if (!username) {
    await provisionFromEmail(token)
    return
  }

  try {
    const response = await fetch(apiUrl('/users'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ username }),
      cache: 'no-store',
    })

    if (!response.ok) {
      // username pris/invalide (course après le pré-check) → repli dérivé email.
      await provisionFromEmail(token)
      return
    }

    // user OK → on pose le profil avec display_name = username choisi.
    await provisionProfil(token, username)
  } catch {
    await provisionFromEmail(token)
  }
}
