// TODO : implémenter le formulaire de connexion
//   - champs email + password (validation côté client)
//   - POST vers ${NEXT_PUBLIC_API_URL}/auth/login via l'API Gateway
//   - stocker le JWT (cookie httpOnly côté serveur de préférence)
//   - rediriger vers /feed en cas de succès
export default function LoginPage() {
  return (
    <main className="container py-10">
      <h1 className="text-3xl font-bold">Connexion</h1>
    </main>
  )
}
