import { redirect } from 'next/navigation'

// Racine → fil d'actualité. `/feed` est public (mode visiteur) : un utilisateur
// non connecté y voit le fil public avec l'invite de connexion ; un membre y
// voit l'expérience complète. La garde de session vit dans le middleware et côté
// composants (cf. AuthPromptProvider).
export default function HomePage() {
  redirect('/feed')
}
