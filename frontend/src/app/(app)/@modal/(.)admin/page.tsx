// Route sans overlay : intercepter `/admin` vers `null` referme l'overlay
// ouvert et laisse la page Admin (children) s'afficher au premier plan.
export default function CloseOverlayOnAdmin() {
  return null
}
