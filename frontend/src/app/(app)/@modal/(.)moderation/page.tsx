// Route sans overlay : intercepter `/moderation` vers `null` referme l'overlay
// ouvert et laisse la page Modération (children) s'afficher au premier plan.
export default function CloseOverlayOnModeration() {
  return null
}
