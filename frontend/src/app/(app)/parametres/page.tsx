import { SettingsView } from '@/components/settings/settings-view'

// Accès direct / refresh → rendu normal (pas d'overlay).
// Navigation soft depuis l'app → intercepting route @modal/(.)parametres.
export default function ParametresPage() {
  return <SettingsView />
}
