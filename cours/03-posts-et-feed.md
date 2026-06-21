# 03 — Posts & feed

> Features : création/lecture de posts · feed infini (Pour toi / Abonnements) · **barrière de visibilité serveur** ·
> commentaires fil à 2 niveaux + **audience de réponse** · likes (posts **et** commentaires) · reposts/citations ·
> sondages · épingle · contenu sensible (NSFW) + majorité · mots masqués (muted words).

## Vue d'ensemble

Le cœur produit. Tout vit dans **post-service (MongoDB)**, qui possède posts, commentaires, likes, sondages,
hashtags et signets. Le principe structurant : **toute lecture de post applique une barrière de visibilité
côté serveur** — le front ne fait jamais autorité sur ce qu'un utilisateur a le droit de voir.

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **post** (MongoDB) | posts, commentaires, likes, sondages, épingle, NSFW, compteurs dénormalisés | propriétaire unique du contenu social |
| **profil** (MongoDB) | fournit `visibility` (public/privé) et la majorité (NSFW) | la confidentialité du profil est une donnée profil |
| **user** (PG) | fournit follow + blocage pour la barrière | le graphe social appartient à user |
| **media** (MinIO) | pièces jointes (`media []MediaRef`, ≤ 4) | post-service reste agnostique du contenu |
| **Frontend** | composer, feed infini, rendu optimiste, mots masqués (local) | UX ; jamais l'autorité de sécurité |

## Comment c'est codé

### Barrière de visibilité (le point central)
- **Sur TOUTES les lectures** (feed, détail, stats, profil, réponses, likes) : public → visible ; privé → propriétaire ou abonné accepté seulement. La fonction `canReadAuthor` interroge profil-service (visibilité) et user-service (blocage), **mémoïsée par auteur**. Un post invisible est simplement absent (pas d'erreur).
- Même logique pour le **blocage** : un viewer ayant bloqué (ou bloqué par) l'auteur ne voit pas ses posts.

### Commentaires fil & audience de réponse
- Fil **à 2 niveaux** (commentaire racine + réponses). Point d'entrée unique `CreateComment`.
- **`reply_audience`** (`everyone|followers`) posé **à la création** du post, **non modifiable** ensuite (parité avec l'audience de sondage). Appliqué serveur par `canReplyTo` : un post `followers` n'accepte que l'auteur, ses abonnés, et mods/admins ; sinon `403 ErrReplyNotAllowed`.
- **Optimisation** : le service hydrate un `can_reply` transitoire **uniquement pour les posts `followers`** (le check follow est sauté pour la majorité `everyone` → coût quasi nul sur le feed). Le front s'en sert pour désactiver le composer, jamais comme autorité.

### Likes (posts + commentaires)
- **Idempotents** via index unique (`post_id+user_id`, `comment_id+user_id`). Compteurs **dénormalisés `int32`** (`$inc`) sur le document (`likes_count`) — `int32` car le `$jsonSchema` déclare `bsonType:"int"`.
- Likes de commentaires : collection dédiée `comment_likes`, flag `liked` hydraté à la lecture (auth optionnelle) → pas de second endpoint « liked ids ». Notifications de like-commentaire **différées** (élargirait le contrat cross-service).
- Rafraîchissement des compteurs = **polling batch** (cf. fiche 07), pas de push WebSocket.

### Reposts / citations, épingle
- Reposts/quotes notifiés à l'auteur. **Épingle** : `pinned_at` sur le post, un seul par profil, propriétaire seul ; le profil trie dessus mais les **feeds renvoient une copie sans `pinned_at`** → l'épingle d'autrui ne personnalise jamais le feed global.

### Sondages
- **Métadonnée du post** + collection séparée `poll_votes` (index unique `post_id+user_id` → un vote/compte garanti par Mongo, pas l'UI). Compteurs de choix dénormalisés `int32`.
- Post-service possède l'**expiration** (durée) et l'audience `everyone|followers`. **Visibilité des résultats serveur** : avant de voter, l'électeur ordinaire reçoit des compteurs à zéro ; après son vote unique, `voted_choice_id` → `can_view_results=true`. L'auteur a toujours les compteurs live ; tout le monde après `ends_at` ou clôture manuelle (`closed_at`, auteur seul, conserve la durée d'origine).

### Contenu sensible (NSFW) & majorité
- **Majorité calculée serveur depuis `birth_date`, jamais stockée** : `IsAdultAt` (≥18 ans, calcul calendaire) + `HydrateViewerPolicy` posent `is_adult`/`nsfw_visible` à la lecture de `/profils/me`. Dynamique (se débloque le jour des 18 ans, sans job) ; le front ne peut pas se faire passer pour majeur.
- Préférence `nsfw_enabled` (défaut **ON**, `*bool` omitempty → pas de migration). Valeur **inerte** chez un mineur : `nsfw_visible = is_adult && nsfw_enabled`.
- **Flou côté front, politique serveur autoritative** : le post-service délivre le post marqué `nsfw:true` ; le front le floute si `!nsfw_visible` (filtre d'affichage façon X, pas une barrière dure). Marquage par l'auteur à la création, ou modo/admin via `PATCH /posts/:id/nsfw`. Seuil NSFW (18) **distinct** de l'âge mini d'inscription (13).

### Mots masqués (muted words)
- 100 % front : persistance locale par compte (`/parametres`), le feed masque les posts d'autrui qui matchent. Pas de backend (état cosmétique, pas une donnée de domaine).

## Décisions clés (et alternatives écartées)

- **Barrière de visibilité sur toutes les lectures, pas juste un filtre front par ids suivis** : la sécurité ne dépend jamais du front (critère de notation).
- **Compteurs dénormalisés `int32`** : rendu feed rapide + cohérence validateur. Idempotence par index unique plutôt que lecture-puis-écriture (race-safe).
- **Audience de réponse posée à la création, non éditable** (parité sondage) : règle simple, appliquée au point d'entrée unique `CreateComment` → couvre racines et réponses.
- **NSFW = flou front, majorité serveur** plutôt que stripping serveur pour mineurs : `nsfw_visible` est serveur (pas de triche), évite de complexifier cache/feed, le post reste public.
- **Migration boot des champs contraints** (`backfillReplyAudience`, `backfillBirthDate`) : règle 5b — un seul champ legacy hors-enum brique toutes les écritures futures du doc en Mongo `strict`.
- **Muted words sans backend** : état cosmétique local, pas un service.

## Points de défense / Q&A anticipées

- *« Un client custom peut-il voir un post privé ? »* → non : `canReadAuthor` côté serveur, le post est absent de la réponse.
- *« Comment garantir un seul vote par sondage ? »* → index unique Mongo `post_id+user_id`, pas une garde UI.
- *« Pourquoi `int32` et pas `int` ? »* → le `$jsonSchema` déclare `bsonType:"int"` ; un `int64` ferait échouer la validation.

## Limites assumées / perspectives

- **Édition de post** : back prêt, front à câbler.
- `post-service.visiblePage` peut scanner beaucoup de posts privés filtrés (surveillance perf/DoS).
- Compteurs followers calculés (COUNT) — dénormaliser si la charge l'exige.
