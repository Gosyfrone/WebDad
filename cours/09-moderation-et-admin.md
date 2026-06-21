# 09 — Modération & administration

> Features : système de signalement (report-service) · tickets agrégés · avertissements · auto-masquage au seuil ·
> corbeille « Tweets supprimés » · panel admin (bugs, paramètres, infra) · modération NSFW · les 3 rôles.

## Vue d'ensemble

La modération matérialise les **3 rôles** (User/Modérateur/Administrateur) exigés par la grille. Un signalement porte
sur une entité possédée par un **autre** service (post, message, profil) → il n'appartient à aucun d'eux : d'où un
**report-service dédié** (port 8090). La modération **se branche sur l'existant** (soft-delete, ban, rôles) au lieu de
le réimplémenter.

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **report** (MongoDB) | tickets de signalement agrégés, warns, seuil d'auto-masquage (singleton `settings`) | un signalement ne relève d'aucun service métier → service propre |
| **post** (MongoDB) | applique l'auto-masquage (`auto_hidden`) + soft-delete (`is_hidden`) | la visibilité est la responsabilité de post-service |
| **message** (MongoDB) | suppression d'un message signalé par modération (tombstone par id) | E2EE intact : le serveur pose juste le tombstone |
| **auth/user** | ban (`is_active`), rôles, gating mod ne peut bannir un admin | l'identité/le ban vivent là |
| **Frontend** | `ReportDialog`, onglets Signalements/Comptes, `TicketDetail`, `WarningsGate` | UX modération/admin |

## Comment c'est codé

### report-service : tickets agrégés
- **Un ticket parent par entité** (modération) : index Mongo **unique partiel** `(entity_type, entity_id)` filtré sur `category=moderation` → les N signalements d'un même contenu sont empilés dans `reports[]` d'un seul ticket (`report_count` + `reason_tags` dénormalisés via `$inc`, upsert atomique). Les **bugs** (motif « Bug technique ») sont des **tickets autonomes** (`entity_type=app`).
- **Catégorie déduite du motif, formulaire unique** : « Bug technique » → catégorie `bug` (limite 500, onglet Administration) ; autres motifs → `moderation` (limite 255). Motifs **bornés** (enum fermé) car `reason_tags.<motif>` est une clé Mongo (anti-injection de champs) ; longueur comptée en **runes**.
- **Un seul signalement par (utilisateur, entité)** : `$push` conditionné par `reports.reporter_id $ne <moi>` ; doublon → E11000 → `409 ErrAlreadyReported`. Retry unique distingue course (empile) et vrai doublon.
- **Réouverture auto à seuil** : `reports_since_closed` (`$inc`, remis à 0 à chaque changement de statut) ; atteint `ReopenThreshold=2` sur un ticket `closed` → réouverture auto. Couplé à « un signalement par utilisateur » → **2 personnes distinctes** nécessaires (anti-spam).
- **Le ticket conserve l'entité même pour un bug** (`entity_type`+`entity_id`) → le détail affiche le contenu réel pour juger/investiguer ; `app` seulement pour un bug applicatif sans entité.

### Auto-masquage au seuil
- Un POST de modération atteignant un **seuil** est automatiquement masqué (sort des fils) ; les **bugs** en sont exemptés. Le report-service détient le compteur mais la visibilité est à post-service → **appel serveur-à-serveur off-gateway** `POST /internal/posts/:id/auto-hide` (`X-Internal-Secret`, best-effort mais **loggé** car sécurité-sensible).
- Champ post `auto_hidden` **distinct de `is_hidden`** (retrait manuel → corbeille) : un post auto-masqué n'apparaît PAS dans la corbeille. Idempotent.
- **Seuil réglable par l'admin** : collection **singleton `settings`** (`_id="global"`, `auto_hide_threshold` int32, **0 = désactivé**, défaut **5**), seedée par `$setOnInsert` idempotent (ne réécrase jamais un seuil déjà réglé — règle 5b). Écriture **admin seul** (`PATCH /reports/settings`), dans Admin › Paramètres.

### Validation, suppression, warns
- **Verrou de validation : statut terminal `approved`** : quand le modérateur juge l'entité conforme, le post est démasqué (`auto-unhide`) ET tout nouveau signalement est **refusé** (`409 ErrReportingLocked`). Distinct de `closed` (réouvrable). Le verrou vit dans report-service (pas de dépendance cross-service pour bloquer).
- **Message E2EE signalé** : le **signaleur** (destinataire) joint **volontairement** la copie en clair (`disclosed_content`) — le serveur ne déchiffre **jamais** de lui-même. Suppression par modération : `DELETE /messages/moderation/:messageId` (mod/admin) tombstone le message **par son seul id** (`deleted_by_moderation=true`, WS `message_updated`) → **E2EE intact**.
- **Réutilisation de l'existant** : « Tweets supprimés » (soft-delete `is_hidden`), ban (`is_active`), gating rôles (mod ne peut bannir/supprimer un admin) **préexistaient** → la modération s'y branche (retrait = soft-delete, transfert = changement de `category`).
- **Avertissement asynchrone = pull (poll)** : un Warn est persisté ; `WarningsGate` (layout `(app)`) interroge `GET /reports/warnings/pending` au montage **et au retour de focus**, affiche une **modale bloquante** acquittée par `POST .../ack`. Simple, robuste (pas de WS dédié).

### Panel admin & rôles
- `AdminView` à onglets : **Signalements (bugs)** (réception + **transfert** bug→modération) + **Paramètres** (seuil) + **Infrastructure** (monitoring). Onglet modération : filtres cumulables (statut/volume/plage), `TicketDetail` (encart « profil de risque » de la cible + Bannir, journal d'actions, cycle open/closed/reopened), corbeille « Tweets supprimés » filtrable.

## Décisions clés (et alternatives écartées)

- **report-service dédié** plutôt que polluer post/message/profil : un signalement n'appartient à aucun service métier → renforce « architecture microservices cohérente » (critère).
- **Ticket parent agrégé** (index unique partiel) : N signalements d'un contenu = un ticket, pas N.
- **Catégorie déduite du motif, un seul formulaire** : conforme à l'énoncé, motifs bornés anti-injection.
- **Auto-masquage off-gateway, visibilité chez post-service** : même esprit que l'émission de notifications ; la visibilité reste serveur (§6), jamais sur la foi du front.
- **Statut terminal `approved`** distinct de `closed` : décision produit (validation définitive vs réouvrable).
- **Modération E2EE par divulgation du signaleur**, pas déchiffrement serveur : conforme à l'E2EE (analogue WhatsApp/Signal « report »).
- **Réutiliser soft-delete/ban/rôles** plutôt que réimplémenter : moins de code, cohérence.
- **Warn en pull (poll)** plutôt que push WS : intercepte la prochaine connexion/focus, robuste sans WS dédié.

## Points de défense / Q&A anticipées

- *« Les 3 rôles sont-ils fonctionnels ? »* → User signale ; Modérateur traite tickets/warns/retraits ; Admin règle le seuil, transfère les bugs, bannit — gating mod≠admin.
- *« Modérer un DM chiffré sans casser l'E2EE ? »* → le destinataire divulgue volontairement le message ; le serveur ne déchiffre jamais, pose juste un tombstone par id.
- *« Un seul compte peut-il rouvrir un ticket en spammant ? »* → non : un signalement par (utilisateur, entité) + seuil de réouverture = 2 personnes distinctes.

## Limites assumées / perspectives

- Reset admin de passphrase de messagerie différé (cf. fiche 06).
- Nettoyage des médias orphelins à la suppression — TODO.
