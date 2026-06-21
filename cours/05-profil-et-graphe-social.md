# 05 — Profil & graphe social

> Features : profil utilisateur (vue/édition, par username) · visibilité public/privé · activité en ligne ·
> nationalité · follow/followers + compteurs · demandes de suivi privées · blocage/déblocage utilisateur.

## Vue d'ensemble

L'identité d'un utilisateur est **scindée entre deux services** par principe « une donnée = un service » :
**user-service (PostgreSQL)** possède l'identité (username) et le **graphe social** (follow, block) ;
**profil-service (MongoDB)** possède les champs **décoratifs et éditables** (display_name, bio, avatar, nationalité,
`visibility`, activité en ligne). La vue agrégée est composée par l'appelant — aucune transaction inter-DB.

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **user** (PG) | username, `follows`, `follow_requests`, `blocks`, compteurs, suggestions | le graphe social relie deux identités |
| **profil** (MongoDB) | display_name, bio, avatar/bannière, nationalité, `visibility`, `likes_visibility`, activité | champs décoratifs avec interrupteur de confidentialité |
| **post** (MongoDB) | applique le blocage et la visibilité activité à la lecture | la barrière est serveur, côté contenu |
| **Frontend** | vue profil agrégée, édition, blocage UX, paramètres | UX ; jamais l'autorité |

## Comment c'est codé

### Identité vs décoration (data ownership)
- profil-service possède `visibility` (défaut **public** à la création, `PATCH /profils/me` pour basculer), `likes_visibility`, et l'activité. user-service possède identité + graphe. Le **rôle vit dans le JWT**.
- **Agrégation `ProfilDetails` composée par le front/BFF**, pas par le service → découple le service de l'agrégation, aucun changement de routage.
- **Nationalité** = code ISO 3166-1 alpha-2 en base (jamais un libellé traduit) ; le combobox charge le catalogue via la route BFF cachée `/api/countries`, `Intl.DisplayNames` localise. L'API externe n'est qu'une source de catalogue (profils fonctionnels si indisponible).
- **`display_name`** : jeu de caractères restreint (lettres/chiffres Unicode, espaces, `-`, `_`) à la création/vrai renommage ; legacy toléré tant qu'inchangé. `birth_date` set-once. Architecture de **cooldown** posée (timestamps), désactivée par défaut (`*_CHANGE_COOLDOWN`).

### Activité en ligne
- **L'activité appartient à profil-service**, à côté de `visibility` : c'est une décoration publique avec interrupteur de confidentialité. `last_login_at`/`is_online` **omis** si `activity_visibility=private` ; un profil privé ne révèle l'activité qu'au propriétaire ou aux abonnés acceptés.
- Marquée best-effort par les handlers BFF (login/OAuth/verify → `PATCH /profils/me/activity` ; logout → `.../offline`) ; le **refresh ne compte pas** comme une nouvelle connexion.

### Follow / followers / demandes privées
- **Table d'arêtes `follows` + table séparée `follow_requests`** : une demande à un profil privé **n'est pas une relation** (pas d'arête avant acceptation). Accept = **transaction atomique** (delete request + insert edge) → jamais « accepté mais pas suivant ».
- **Compteurs calculés (COUNT)** à la lecture détaillée ; dénormalisation différée.
- **Auto-acceptation** des demandes en attente au passage privé→public (profil-service → user-service `/internal/.../accept-all-follow-requests`, best-effort).

### Blocage / déblocage
- **Le blocage est une relation du graphe social → user-service.** `blocks(blocker_id, blocked_id)` vit avec `follows`/`follow_requests` (il relie deux identités, coupe les relations dans les deux sens). Routes publiques `POST/DELETE /users/:id/block`, `GET /users/me/blocks` ; route interne `HasBlocked` pour les services.
- **L'effet de confidentialité reste serveur dans post-service** : `canReadAuthor` interroge user-service et exclut les posts du bloqueur/bloqué du feed, détail, stats, profil, réponses, likes. Le front masque immédiatement pour l'UX, mais la vraie barrière est serveur.
- **Déblocage = retour aux règles normales** : ne recrée pas les follows supprimés ; un profil privé redevient soumis au follow/acceptation.

## Décisions clés (et alternatives écartées)

- **Identité (user) vs décoration (profil) séparées** : une donnée = un service, aucune transaction inter-DB pour un `PATCH /profils/me`.
- **Activité dans profil, pas user** : c'est une décoration publique avec switch de confidentialité, appliquée à la lecture → le switch reste un choix indépendant du profil privé.
- **`follow_requests` séparée des `follows`** : une demande n'est pas une arête ; accept atomique évite l'incohérence.
- **Blocage dans user, effet dans post** : le blocage relie deux identités (user) mais sa conséquence sur le contenu est une barrière serveur (post), comme les profils privés.
- **Nationalité en code ISO**, pas un libellé : robuste à la traduction et à l'indisponibilité de l'API catalogue.
- **Compteurs en COUNT** : suffisant à l'échelle du projet ; dénormaliser sous charge (compromis assumé).

## Points de défense / Q&A anticipées

- *« Pourquoi le blocage n'est pas dans profil ? »* → c'est une relation entre deux identités, donc graphe social = user-service ; profil ne porte que des champs décoratifs.
- *« Un bloqué peut-il voir mes posts via un client custom ? »* → non, post-service interroge `HasBlocked` à chaque lecture.
- *« L'activité d'un profil privé fuite-t-elle ? »* → non, omise sauf propriétaire/abonné accepté.

## Limites assumées / perspectives

- **Lecture agrégée user+profil+post** à câbler entièrement côté front.
- Tests d'intégration repo user-service (SQL) — actuellement couverts par e2e manuel.
- Points de revue privacy : la route interne `is-following` ne vérifie pas encore `X-Internal-Secret` (fuite de graphe si port joignable).
