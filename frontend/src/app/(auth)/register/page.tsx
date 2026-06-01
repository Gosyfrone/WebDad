// TODO : implémenter le formulaire d'inscription
//   - champs username, email, password, password confirmation
//   - validation : contraintes de mot de passe, email unique
//   - POST vers ${NEXT_PUBLIC_API_URL}/auth/register
//   - rediriger vers /login (ou auto-login + /feed) en cas de succès
export default function RegisterPage() {
  return (
    <main className="container py-10">
      <h1 className="text-3xl font-bold">Inscription</h1>
    </main>
  )
}
