// Route sans overlay : intercepter `/feed` vers `null` vide le slot `@modal`
// (sur navigation soft, un slot ne revient pas seul à `default.tsx`), ce qui
// referme tout overlay ouvert et révèle le feed (children) au premier plan.
export default function CloseOverlayOnFeed() {
  return null
}
