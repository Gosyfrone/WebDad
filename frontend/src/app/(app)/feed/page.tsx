// Le fil est monté en permanence dans `(app)/layout.tsx` (arrière-plan persistant).
// Cette route ne rend donc rien : le feed du layout transparaît. Les autres
// sections se rendent en overlay (`FeedOverlay`) par-dessus ce même feed.
export default function FeedPage() {
  return null
}
