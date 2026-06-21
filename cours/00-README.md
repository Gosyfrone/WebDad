# Breezy (WebDad) — Fiches de soutenance par domaine

> Une fiche par **domaine fonctionnel**. Chaque fiche regroupe les features de ce domaine et répond, pour chacune,
> à quatre questions de soutenance : **quoi**, **quels services concernés et pourquoi**, **comment c'est codé**,
> **quelles décisions** (avec les alternatives écartées).
>
> Sources : `PROJECT_STATUS.md` (état + tableau des features), `DECISIONS.md` (le *pourquoi*), `CHANGELOG.md`
> (historique), et le graphe `graphify-out/` (détail code). Ces fiches sont une synthèse orientée défense orale,
> pas une doc d'API (celle-ci est publiée : `https://gosyfrone.github.io/WebDad/`).

---

## Rappel d'architecture (à connaître avant tout)

**Breezy** est un réseau social façon X/Twitter, en **microservices 4 couches**, pour le projet « Développement d'applications distribuées » (FISA INFO A3).

```
[Client] → [Frontend Next.js + BFF] → [API Gateway] → 8 services Go → DBs (PostgreSQL / MongoDB / MinIO)
```

| Service | Port | DB | Domaine |
|---|---|---|---|
| api-gateway | 8080 | — | reverse proxy, routage par préfixe, validation/propagation JWT, CORS, streaming média |
| auth | 8081 | PostgreSQL | credentials, JWT, OAuth, MFA, tokens e-mail |
| user | 8082 | PostgreSQL | identité (username), graphe social (follow/block), préférence de langue |
| profil | 8083 | MongoDB | champs décoratifs, `visibility`, activité en ligne, nationalité, NSFW (majorité) |
| post | 8084 | MongoDB | posts, commentaires, likes, sondages, hashtags, signets, feed temps réel |
| message | 8085 | MongoDB | messagerie E2EE (DM/groupes/communautés), reçus, sauvegarde de clé |
| notification | 8086 | MongoDB | notifications agrégées + WebSocket |
| media | 8087 | MinIO | stockage opaque (images/vidéos/blobs chiffrés), proxy GIPHY |
| mail | 8089 *(hors gateway)* | — | transport SMTP (vérif, reset, changement d'e-mail) |
| report | 8090 | MongoDB | signalements (tickets agrégés), avertissements, auto-masquage |

**5 principes non négociables** (récurrents dans toutes les fiches) :

1. **Tout passe par la gateway** — sauf 2 canaux serveur-à-serveur off-gateway (émission de notifications, auto-masquage), authentifiés par secret partagé.
2. **La sécurité ne dépend jamais du front** — toute barrière (visibilité, blocage, audience de réponse, sondage) est appliquée **côté serveur**.
3. **Une donnée = un service** — `username`→user, `display_name`/`visibility`→profil ; les vues agrégées sont composées par l'appelant (front/BFF).
4. **Chaque service possède son schéma** (`EnsureSchema` au boot, idempotent) et sa propre DB → autonomie.
5. **Migrations rétrocompatibles** — une base peuplée existe en prod ; tout champ contraint (enum, required) exige un backfill idempotent au boot, sinon Mongo `strict` rejette toute mise à jour.

---

## Index des fiches

| Fiche | Domaine | Features couvertes |
|---|---|---|
| [01](01-authentification-et-jwt.md) | **Authentification & JWT** | inscription/connexion, OAuth social, JWT + routes protégées, gateway, rôles, comptes créés par admin |
| [02](02-securite-des-comptes.md) | **Sécurité des comptes** | MFA TOTP, vérification e-mail, mot de passe oublié, changement d'e-mail, mail-service, durcissement pentest |
| [03](03-posts-et-feed.md) | **Posts & feed** | création/lecture, barrière de visibilité, commentaires + reply audience, likes, reposts/citations, sondages, épingle, NSFW, muted words |
| [04](04-hashtags-recherche-explorer.md) | **Hashtags, recherche & Explorer** | hashtags & trends, mentions, recherche comptes/hashtags, page Explorer |
| [05](05-profil-et-graphe-social.md) | **Profil & graphe social** | profil + édition, activité en ligne, follow/followers, demandes privées, blocage |
| [06](06-messagerie-e2ee.md) | **Messagerie E2EE** | DM/groupes/communautés, reçus, typing, suppression, sauvegarde de clé, pièces jointes |
| [07](07-temps-reel-et-notifications.md) | **Temps réel & notifications** | bandeau « a posté » WS, compteurs polling, notifications agrégées, sons, feed en fond |
| [08](08-media-images-gif.md) | **Médias & GIFs** | media-service + MinIO, upload image/vidéo, GIFs GIPHY, lightbox, autoplay |
| [09](09-moderation-et-admin.md) | **Modération & admin** | report-service, tickets, warns, auto-masquage, panel admin, NSFW modération |
| [10](10-bookmarks-partage-visiteur.md) | **Signets, partage & visiteur** | collections de signets, partage post/profil, mode visiteur (fil public) |
| [11](11-i18n-theming-traduction.md) | **i18n, thème & traduction** | 12 langues, préférence de langue, traduction auto, thème clair/sombre/personnalisé, responsive |
| [12](12-infrastructure-ci-cd.md) | **Infrastructure & CI/CD** | Docker/compose, gateway, observability (slog), CI/CD, Swagger/OpenAPI |

---

## Comment lire une fiche en révision

- **Quoi** = la phrase à dire en demo.
- **Services concernés & pourquoi** = répond directement au critère « architecture microservices cohérente ».
- **Comment c'est codé** = de quoi tenir un Q&A technique.
- **Décisions & alternatives écartées** = le cœur de la note (« choix justifiés »). Toujours pouvoir dire *ce qu'on a écarté et pourquoi*.
