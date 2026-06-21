# 04 — Hashtags, recherche & Explorer

> Features : hashtags extraits côté serveur + Top trends · mentions (@handle) · recherche comptes/hashtags ·
> page Explorer (découverte).

## Vue d'ensemble

Tout ce qui sert à **découvrir** du contenu et des comptes. Les hashtags et trends vivent dans **post-service**
(pas de service de recherche dédié), la recherche de comptes dans **user-service**, et les mentions réutilisent des
briques pures partagées entre posts, commentaires et messages. Toujours la même contrainte : **les compteurs et
résultats respectent la barrière de visibilité** (un profil privé ne fuite ni par les trends ni par la recherche).

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **post** (MongoDB) | extraction/normalisation des hashtags, `GET /posts?hashtag=`, `/posts/trends` | les hashtags sont des champs dénormalisés sur les posts → indexables sans service tiers |
| **user** (PG) | `GET /search`, `/suggestions`, `/by-username` | la recherche de comptes porte sur l'identité (username) |
| **Frontend / BFF** | composer autocomplete, page Explorer, historique de recherche local | UX de découverte |

## Comment c'est codé

### Hashtags & trends
- **Champs dénormalisés sur `posts`** (`hashtags: []string`, minuscule, sans `#`), extraits à la création/édition par post-service (extracteur/normalizer unique). Rend `GET /posts?hashtag=...` indexable en Mongo, **sans service de recherche ni cross-DB**.
- **Trends comptés après la même barrière de visibilité que le feed** → un profil privé ne fuite pas par un compteur de hashtag. `GET /posts/trends?q=...` supporte des suggestions par préfixe (même filtrage avant comptage).
- Vue résultats `?hashtag=` avec `sort=top` (likes, reposts, commentaires, puis récence) / `sort=recent` ; onglet média = mêmes résultats filtrés, seulement leurs pièces jointes.
- Explorer « posts avec hashtags » = `GET /posts?hashtag_any=true` (filtrage **serveur**, pas client → préserve la barrière et évite de télécharger des pages de feed pour les jeter).
- L'**autocomplete du composer** réutilise `/posts/trends?q=...` (pas d'endpoint séparé) ; post-service reste le seul extracteur sur soumission.

### Mentions (@handle)
- **Briques pures partagées** (`lib/mentions.ts`, regex alignée avec le back) réutilisées posts/commentaires/messages → une seule source de vérité pour regex/insertion/rendu.
- **Posts/commentaires** : le back émet déjà `mention` ; le front ajoute autocomplete + rendu cliquable, **sans changement back**.
- **Messages (E2EE)** : le serveur ne lit pas le `ciphertext` → le **client** calcule `mentioned_member_ids` (ids seulement) → message-service émet `message_mention` (cf. fiche 06).

### Recherche & Explorer
- Recherche par `@username`, username nu, ou `display_name` ; navigation directe `#hashtag`. Les suggestions proposent d'abord les trends puis les profils.
- Page **Explorer** : Top 10 trends, suggestions de profils non suivis, publications du feed, et filtres de recherche soumis (Publications/Utilisateurs, au moins un requis).
- **Historique de recherche** = `localStorage` par utilisateur (`breezy-suggestion-history`) — **état cosmétique**, pas une donnée de domaine méritant un service.

## Décisions clés (et alternatives écartées)

- **Hashtags dénormalisés sur les posts** plutôt qu'un service/collection de recherche dédié : indexable nativement en Mongo, zéro cross-DB, post-service reste l'unique extracteur.
- **Trends comptés après la barrière de visibilité** : sinon un profil privé fuiterait par les compteurs publics.
- **Filtrage hashtag côté serveur** (`hashtag_any=true`) plutôt que filtrer des pages de feed côté client : préserve la barrière et la bande passante.
- **Mentions = briques pures partagées** : une regex, une logique d'insertion/rendu, réutilisée sur 3 surfaces.
- **Historique de recherche en localStorage** : cosmétique, pas un service.

## Points de défense / Q&A anticipées

- *« Pas de service de recherche dédié ? »* → à l'échelle du projet, les hashtags dénormalisés + index Mongo suffisent ; un service full-text serait du sur-dimensionnement (perspective).
- *« Les trends peuvent-ils révéler un compte privé ? »* → non, le comptage applique `canReadAuthor` avant.

## Limites assumées / perspectives

- **Recherche full-text de posts** (au-delà des hashtags) : perspective.
