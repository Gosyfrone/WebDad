import { redirect } from 'next/navigation'

// TODO : remplacer par une vraie vérification de session (cookie / JWT côté serveur).
// Pour l'instant : redirige toujours vers /login. Quand l'auth sera branchée,
// rediriger vers /feed si l'utilisateur est connecté.
export default function HomePage() {
  const isAuthenticated = false

  if (isAuthenticated) {
    redirect('/feed')
  }

  redirect('/login')
}
