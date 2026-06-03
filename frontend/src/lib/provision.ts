import { apiUrl } from '@/lib/config'

/**
 * Provisioning paresseux via l'email : `GET /users/me` crée la ligne `users`
 * en dérivant le handle de l'email (partie locale). Sert de repli.
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

/**
 * Provisioning de l'utilisateur dans le user-service après login / register.
 *
 *   - Avec `username` (inscription) : `POST /users { username }` crée la ligne
 *     avec le handle choisi par l'utilisateur. La disponibilité est
 *     pré-vérifiée côté page register ; en cas de course rarissime (username
 *     pris entre le check et le POST), on retombe sur le handle dérivé de l'email.
 *   - Sans `username` (connexion) : `GET /users/me` (dérivé email / idempotent).
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
    }
  } catch {
    await provisionFromEmail(token)
  }
}
