# 07 — Temps réel & notifications

> Features : notifications temps réel agrégées (style Instagram) · bandeau « a posté » (WebSocket) ·
> compteurs dynamiques (polling batch) · sons d'action/temps réel · feed en arrière-plan persistant.

## Vue d'ensemble

Quatre mécanismes temps réel distincts, **choisis selon l'échelle** — c'est l'angle de défense : on ne « cargo-culte »
pas du WebSocket partout. **notification-service** pousse les notifications par WS ; le **bandeau « a posté »** est un
WS léger ; les **compteurs** sont volontairement en **polling** (pas WS) ; le **feed** reste monté en fond.

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **notification** (MongoDB) | ingestion `/internal/events`, agrégation, WS, badge | service dédié, 2ᵉ consommateur d'événements internes |
| **post** (MongoDB) | hub de broadcast feed (`/posts/ws`), `GET /posts/stats` | le feed est public → broadcast plat |
| **autres services** | émettent les événements (like/comment/follow/message…) best-effort | émission fire-and-forget off-gateway |
| **Frontend** | badge en mémoire, filtres de pertinence, polling, sons | UX temps réel |

## Comment c'est codé

### Notifications agrégées
- **Émission = best-effort fire-and-forget** vers `/internal/events` (**off-gateway**, `INTERNAL_EVENT_SECRET`). Couplage temporel neutralisé par le best-effort ; un bus d'événements (NATS/Redis) est sur-dimensionné ici (perspective rapport).
- **Agrégation anti-spam** : une notification = un `group_key` par destinataire (`like:<post>`, `comment:<post>`, `reply:<rootComment>`, `mention:<source>`, `repost`, `quote`, `follow`, `follow_request:<actor>`, `message`…). La plupart `$inc count` ; `retract` décrémente/supprime.
- **`message` est l'exception** : `actor_ids` stocke les expéditeurs uniques, `count = len(actor_ids)` → plusieurs messages d'une personne = une notif, de personnes différentes = « X et N autres ». Groupé globalement par destinataire, **jamais émis pour les communautés**.
- **Règles par type** : like/comment/repost/quote → auteur du post ; reply → auteur **racine** ; mention → chaque mentionné ; follow_request → propriétaire du profil privé (actionnable) ; jamais à soi-même.
- **Badge front** : compteur non-lu **en mémoire** (dédup par id), **un seul WS**, amorcé par `unread-count` → évite une requête serveur par événement pendant les pics ; s'auto-corrige à l'ouverture / `notification_refresh`.

### Bandeau « a posté » (WebSocket)
- **Hub de broadcast** (pas par utilisateur) : le feed étant public, le hub garde un **set plat** de connexions et diffuse à tous. `GET /posts/ws?access_token=<jwt>` (token en query : un navigateur ne peut pas poser `Authorization` sur un handshake WS). Le gateway proxy déjà l'upgrade WS sous `/posts` → **aucun changement gateway**.
- **Ping, pas contenu (« firehose + refetch »)** : l'événement WS ne porte que `{type, post_id, author_id}`. Le front résout l'auteur depuis son cache et **refetch le contenu via le endpoint feed authentifié normal** → WS léger ET barrière de visibilité **côté serveur** (jamais dupliquée dans le hub).
- **Gating compte public à l'émission** : `broadcastNewPost` ne ping que si l'auteur est public → un compte privé ne signale pas son activité. Fire-and-forget en goroutine, ne bloque jamais la création.
- **Filtre de pertinence client** : `FeedView` jette les pings de soi-même, ceux pendant un filtre hashtag, les posts déjà affichés ; l'onglet « Abonnements » filtre en plus par `getFollowingIds`.

### Compteurs dynamiques (polling batch — PAS de WS)
- **Choix justifié par l'échelle** : on **ne réutilise pas** le hub pour les deltas de compteurs. Pousser chaque like/commentaire de chaque post public à tout le monde = haut volume nécessitant de la coalescence serveur — exactement le comportement « milliseconde » qu'on ne veut pas.
- À la place : le front poll **une** requête légère `GET /posts/stats?ids=…` pour toute la page affichée **toutes les ~7 s**. On garde 2 des 3 idées de X (lazy = posts affichés seulement ; batché) et on **abandonne les deltas** (valeurs absolues, plus simples et auto-cicatrisantes).
- **Même barrière de visibilité réutilisée** (`canReadAuthor` mémoïsé) ; un `$in` projeté borné (`MaxStatsIDs=100`). `applyStatsToPost` ne touche **que** les 3 compteurs (jamais le `liked`/`reposted`/`bookmarked` optimiste local) et préserve la référence si rien ne change → zéro re-render sur un cycle calme. Hook `usePostStatsPolling` (poll seulement si `!document.hidden`, refetch au refocus, anti-chevauchement).

### Sons & feed en arrière-plan
- **Sons** d'action/temps réel : publication d'un breeze, notification reçue, message reçu (front).
- **Feed monté une seule fois** dans `(app)/layout.tsx` (façon X) : `feed/page.tsx` rend `null`, les autres pages s'enveloppent dans `<FeedOverlay>` (`fixed inset-0`) rendu au niveau racine du layout. `OverlayScrollLock` fige le feed (`overflow:hidden` sur `<html>`) sans perdre le scroll. Hors `/feed`, `FeedView` **gèle** sa lecture des query params (refs) pour ne pas refetcher en fond.

## Décisions clés (et alternatives écartées)

- **WS pour les notifications et le bandeau, polling pour les compteurs** : choix *justifié par l'échelle*, pas calqué sur X aveuglément (angle de défense fort).
- **Ping + refetch** plutôt que pousser le contenu : WS léger + barrière de visibilité serveur non dupliquée.
- **Hub plat (broadcast)** pour le feed vs hub indexé par destinataire (notif/message) : le feed est public.
- **Émission off-gateway best-effort** : le couplage temporel est neutralisé ; un bus d'événements serait sur-dimensionné.
- **Badge en mémoire, un seul WS** : évite une requête par événement durant les pics ; s'auto-corrige.
- **Feed au niveau layout** plutôt que `@modal` (parallel/intercepting routes) : ces dernières étaient fragiles (404 intermittents, scroll non bloqué, refresh cassé) — abandonné après 4 bugs cumulés.

## Points de défense / Q&A anticipées

- *« Pourquoi pas du WebSocket pour les likes ? »* → à grande échelle, pousser chaque like à chaque connexion est ingérable sans coalescence ; un `$in` sur ~20 ids toutes les 7 s est négligeable à notre échelle.
- *« Le bandauu fuite-t-il les posts privés ? »* → non : gating compte-public à l'émission **et** refetch via l'endpoint à barrière serveur.
- *« Que se passe-t-il si notif-service tombe ? »* → émission best-effort → l'événement est perdu (assumé), pas de blocage de l'action métier.

## Limites assumées / perspectives

- Notifications : badge en mémoire peut sur-compter de 1 (s'auto-corrige) ; incrémente même `/notifications` ouvert.
- Bus d'événements (NATS/Redis) = perspective si la charge l'exige.
- `FeedView` se monte sur tout deep-link `(app)` (fetch feed invisible) — prix du retour instantané.
