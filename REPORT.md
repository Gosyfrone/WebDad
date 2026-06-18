# REPORT — Système de signalement & modération (branche `91-signalement-des-breeze`)

> Journal de bord de l'implémentation du système de signalement / tickets / modération.
> Tenu à jour au fil de l'eau (exigé par l'utilisateur). FR = référence, EN en parité.
> Thème : tous les écrans gérés en **clear & dark mode** (variables CSS shadcn, jamais de couleur en dur).

## Décisions validées (avec l'utilisateur)

1. **Nouveau `report-service`** (Go + MongoDB, port 8090, préfixe gateway `/reports`) — domaine
   « tickets de signalement » à part entière (les tickets agrègent des entités possédées par
   d'autres services : posts, messages, profils → n'appartiennent à aucun). Respecte « une donnée = un service ».
2. **Rapports de bug = tickets autonomes** (pas d'agrégation par entité ; un bug = un ticket).
   L'agrégation par entité ne vaut que pour les signalements de **modération**.
3. **Périmètre complet, livré en phases.**

## Réutilisé (déjà présent, NON reconstruit)

- Suppression douce + onglet « Tweets supprimés » : `post-service` (`is_hidden`/`hidden_by`/`hidden_at`/`purge_at`),
  composant `DeletedPosts`, restore/purge.
- Bannissement (auth `is_active` + visibilité user), profil conservé visible en « Comptes ».
- Rôles user/moderator/admin + changement de rôle + effacement RGPD, gating mod≠admin front + back.
- Upload média (sniff + caps) via `media-service` `POST /media` → réutilisé pour la pièce jointe (≤5 Mb).

---

## Architecture du `report-service`

**Collections Mongo :**
- `tickets` — un ticket parent par entité signalée (modération) ; bug = ticket autonome.
  - `category` (moderation|bug), `entity_type` (post|message|group_message|profile|app), `entity_id`,
    `status` (open|closed|reopened), `report_count` (int32 dénormalisé), `reason_tags` (map motif→compte),
    `reports[]` (fil enfant : reporter_id, reason, text, attachment_id, created_at),
    `actions[]` (moderator_id, type reply|status_change|transfer, text, status, created_at),
    `last_reported_at`, `created_at`, `updated_at`.
  - Index : **unique partiel** (entity_type, entity_id) where category=moderation → un seul parent/entité ;
    (category, report_count desc), (category, status), (last_reported_at desc).
- `warnings` — avertissements asynchrones : target_user_id, ticket_id?, message, issued_by, acknowledged, created_at.

**Endpoints (`/reports`, JWT) :**
- `POST /reports` (tout utilisateur) — crée un signalement ; valide longueur texte (255 modération / 500 bug) + motif + entité ; upsert parent ou append enfant.
- `GET /reports/tickets?category=&status=&min_reports=&since=&until=` (mod/admin) — tri décroissant par nombre de signalements ; filtres cumulables.
- `GET /reports/tickets/:id` (mod/admin) — détail + fil enfant + actions.
- `POST /reports/tickets/:id/replies` (mod/admin) — réponse interne (enregistre l'id du modérateur).
- `PATCH /reports/tickets/:id/status` (mod/admin) — open|closed|reopened (enregistre l'action + l'auteur).
- `POST /reports/tickets/:id/transfer` (admin) — déplace un ticket bug → modération.
- `POST /reports/warnings` (mod/admin) — émet un Warn.
- `GET /reports/warnings/pending` (tout utilisateur) — Warns non acquittés du compte courant.
- `POST /reports/warnings/:id/ack` (tout utilisateur) — acquitte.

---

## Avancement

### Phase 1 — backend report-service + create/list — 🟢 FAIT
- [x] Étude des patterns (notification-service comme gabarit Mongo).
- [x] Scaffold service complet (go.mod, Dockerfile, Makefile, .air.toml, doc.go, config, logging, middleware JWT + ModeratorOnly + AdminOnly, db).
- [x] Modèles + schéma Mongo (`EnsureSchema` idempotent, validateurs `tickets`/`warnings`, index unique partiel modération, index tri/filtre).
- [x] Repository (upsert agrégé modération / insert bug autonome, list filtrée triée, actions, warns) + service (validation 255/500, motifs/entités/statuts bornés) + handlers + routes (TOUS les endpoints des phases 1-3).
- [x] Wiring : gateway `/reports`→8090 ; docker-compose `.yml`/`.dev.yml`/`.prod.yml` (mongo-report + service + volume + gateway env + depends_on) ; `.env.example` + `.env` dev créés ; Makefile (SERVICES, logs/sh/mongo-cli) ; CI (ci-go matrix+paths, ci-integration, deploy) ; swagger-aggregate.
- [x] `go build` + `go vet` + `go test` OK ; `docker compose config` (dev + prod) valide ; gofmt clean.

**Endpoints livrés** : `POST /reports` · `GET /reports/tickets` (filtres) · `GET /reports/tickets/:id` ·
`POST /reports/tickets/:id/replies` · `PATCH /reports/tickets/:id/status` · `POST /reports/tickets/:id/transfer` (admin) ·
`POST /reports/warnings` · `GET /reports/warnings/pending` · `POST /reports/warnings/:id/ack`.

Ports DB : mongo-report exposé en `27021` (dev). Service report-service : `8090`.

### Phase 2 — front : ReportDialog + onglet Signalement modération + détail/cycle de vie — 🟢 FAIT
- [x] `lib/reports.ts` (client createReport + tickets + warns, mappers, bornes 255/500/5 Mb).
- [x] `components/ui/textarea.tsx` (primitive manquante, tokens thème clair/sombre).
- [x] `ReportDialog` réutilisable : motif (menu déroulant), texte borné + compteur, pièce jointe image ≤ 5 Mb (upload media-service), i18n, dark mode.
- [x] Bouton **Signaler** câblé sur les **posts** (menu « … », tout utilisateur sauf l'auteur) et les **profils** (icône drapeau).
- [x] Onglet **Signalements** dans la Modération **après Comptes** (`TicketsPanel category=moderation`).
- [x] `TicketDetail` : en-tête agrégé (volume, dernier signalement, motifs récurrents, statut), fil enfant (user + motif + texte + image), journal des actions, réponse interne, cycle de vie (clôturer/rouvrir), retrait du post (soft delete → Tweets supprimés), avertir l'auteur.

### Phase 3 — front : onglet Signalement admin (bugs) + transfert + Warn modal — 🟢 FAIT
- [x] `AdminView` avec onglets **Signalements (bugs)** + **Infrastructure** (`AdminInfra` rendu `embedded`).
- [x] Onglet bugs = `TicketsPanel category=bug` ; **transfert** bug → modération (bouton admin dans `TicketDetail`).
- [x] Bouton **Signaler** câblé sur les **messages privés et de groupe** (sous l'option de suppression, `chat-pane`).
- [x] `WarnDialog` (émission) + `WarningsGate` (interception asynchrone : sondage au montage + au focus → modale **bloquante** « J'ai compris » → ack). Monté dans `(app)/layout`.

### Phase 4 — filtres + i18n + CI/build — 🟢 FAIT
- [x] Filtres tickets **cumulables** (statut, signalements min., plage temporelle) dans `TicketsPanel`.
- [x] Filtres comptes **Bannis / Modérateurs / Administrateurs** (+ Tous) dans `AccountsPanel`.
- [x] i18n FR (référence) + EN (parité) — ~70 clés `report.*`/`tickets.*`/`warn.*`/`accounts.*` ; les 10 autres langues retombent sur le FR (fallback `DEFAULT_LOCALE`).
- [x] `tsc --noEmit` : 0 erreur sur les fichiers touchés (clair/sombre via tokens shadcn, aucune couleur en dur).
- [x] CI (ci-go matrix+paths, ci-integration, deploy) + swagger-aggregate incluent `report-service`.
- [x] `make swagger` régénéré → `doc/openapi.{json,yaml}` contient les routes `/reports` (drift CI OK).

---

## 🧪 Flow de test complet (toutes les fonctionnalités)

**Prérequis** : `make env` puis `make dev`. Front sur http://localhost:3000.
**Comptes nécessaires** : il faut **au moins 3 « users » distincts** (U1, U2, U3) pour tester l'agrégation et la
réouverture à seuil, **+ 1 modérateur (MOD)** et **+ 1 administrateur (ADMIN)**. Promouvoir les rôles via
**Modération → Comptes** (avec un admin) ou via l'admin seed.

> Rappel comportements clés : 1 signalement par (user, entité) · agrégation par entité (modération) · bug = ticket
> autonome mais entité conservée · tri 2 paliers (actifs puis clôturés, par volume) · **jamais de clôture auto** ·
> réouverture auto à **2** re-signalements · warns asynchrones · E2EE messages (divulgation volontaire) · thème clair/sombre · i18n FR/EN.

### A. Déposer des signalements
1. **Post (modération)** — U1 : sur n'importe quel post → menu « … » (haut à droite) → **Signaler** → motif
   (« Comportement inapproprié »…), texte (le compteur **bloque à 255**), pièce jointe image **> 5 Mo** (refusée) puis
   **< 5 Mo** (acceptée) → **Envoyer** → toast succès. ✅ **Le post RESTE dans le feed** (un signalement ne supprime rien).
2. **Doublon interdit** — U1 re-signale **le même** post → toast « Vous avez déjà signalé cet élément » (**409**), rien empilé.
3. **Agrégation** — U2 signale **le même** post (motif différent) → **un seul ticket**, **2 signalements** empilés, 2 étiquettes de motif.
4. **Profil** — U1 : profil d'un autre → icône **drapeau** → Signaler.
5. **Message privé / groupe (E2EE)** — U1 : sur un message **reçu** → « … » → **Signaler le message** (item **sous** Supprimer).
   Un **avis de divulgation** s'affiche : en signalant, U1 (destinataire) transmet la **copie en clair** de CE message ;
   le serveur ne déchiffre jamais de lui-même.
6. **Bug** — depuis n'importe quel formulaire, choisir le motif **« Bug technique »** → bascule en mode bug
   (**limite 500**), le ticket part dans **Administration → Signalements (bugs)**. **L'entité est conservée** : un bug
   signalé sur un tweet garde le tweet visible côté admin.
7. **Visiteur** — déconnecté, tenter de signaler → **modale de connexion** (dépôt réservé aux comptes).

### B. Traiter les tickets de modération (MOD)
8. **Modération → onglet « Signalements »** (placé **après Comptes**). Tri **2 paliers** : tickets **actifs**
   (ouverts/rouverts) par nombre de signalements décroissant, **puis** tickets **clôturés** (même tri).
9. **Filtres cumulables** : **statut**, **signalements min.** (ex. 2 → ne montre que le post signalé 2×),
   **plage temporelle** (Depuis/Jusqu'à) ; cumuler puis **Réinitialiser**.
10. **Détail = vue pleine** (pas une modale) avec **flèche retour** en haut à gauche. Vérifier :
    - **Contenu original** : post réel **embarqué** (juger sur pièce) / profil → **carte cliquable** pour visiter le compte /
      message → note E2EE + **copie divulguée** visible dans le fil (« Message divulgué par le signaleur ») ;
    - **En-tête agrégé** : volume total, **date du dernier signalement qui s'incrémente en direct (1s, 2s, 3s…)**, motifs récurrents, statut ;
    - **Fil** des signalements (auteur + motif + texte + image) ; **journal des actions**.
11. **Réponse interne** → apparaît dans le journal avec **le nom du modérateur** + horodatage.
12. **Cycle de vie manuel** : **Clôturer** → « Clôturé » ; **Rouvrir** → « Rouvert ». Chaque changement est tracé (auteur + heure).
13. **Avertir l'auteur** (post / profil / **message**) → saisir un message → envoyé (cf. section D).
14. **Retirer la publication** (ticket de post) → le post part dans **Modération → Tweets supprimés** ; le retrait est
    **journalisé** (« Contenu signalé retiré par la modération » + MOD) **MAIS le ticket n'est PAS clôturé** (clôture manuelle).
15. **Supprimer le message** (ticket de message) → dans la conversation des participants, le message devient en direct
    **« Ce message a été supprimé par la modération »** ; retrait **journalisé**, ticket **non clôturé**, E2EE préservé.

### C. Réouverture automatique à seuil (jamais de clôture auto)
16. Sur un ticket de post : **Clôturer** (MOD). Puis U3 (qui n'avait pas signalé) signale ce post → ticket **reste clôturé**
    (1 seul re-signalement). Un **2ᵉ nouveau** signaleur (encore un autre compte) → le ticket **se rouvre automatiquement**
    (« Rouvert automatiquement », attribué à **« Système »** dans le journal). *(Comme 1 user = 1 signalement, il faut bien
    2 personnes distinctes → impossible de rouvrir seul.)*

### D. Avertissement asynchrone (modale bloquante)
17. Se connecter avec le **compte averti** (ou simplement **revenir sur l'onglet** s'il est déjà connecté) → une **modale
    bloquante « Avertissement de la modération »** s'affiche → **« J'ai compris »** l'acquitte (ne réapparaît plus).

### E. Administration (ADMIN)
18. **Administration → onglet « Signalements (bugs) »** : ne montre que les **rapports de bug** (entité visible).
19. Ouvrir un ticket de bug → **Transférer vers la modération** → il quitte les bugs et apparaît dans **Modération → Signalements**.
20. Onglet **Infrastructure** : monitoring des microservices (statut/latence/uptime).

### F. Comptes (Modération, partagé MOD + ADMIN)
21. Filtres **Tous / Bannis / Modérateurs / Administrateurs**.
22. **Bannir** un user → il **reste visible** dans Comptes (profil conservé), badge « Banni » ; **Réactiver** le débannit.
23. En **MOD**, tenter de bannir/supprimer un **ADMIN** → **interdit** (bouton bloqué + **403** côté back).
24. En **ADMIN** : **changement de rôle** + **effacement RGPD** (re-saisie du username) disponibles.
25. Onglet **Tweets supprimés** : **Restaurer** (réversible) / **Supprimer définitivement** (confirmation), tri par retrait le plus récent.

### G. Contrôle d'accès
26. Avec un **user** simple, ouvrir `/moderation` ou `/admin` → **accès refusé** (et le back renvoie **403** sur `/reports/tickets`).
27. Le **dépôt** (`POST /reports`) reste ouvert à tout compte connecté ; **transfert** réservé admin ; gestion des tickets réservée mod/admin.

### H. Transverse
28. **Thème** : basculer clair ↔ sombre → tous les écrans signalement/tickets suivent (tokens shadcn, aucune couleur en dur).
29. **i18n** : basculer FR ↔ EN → libellés traduits (les 10 autres langues retombent sur le FR).

### Vérifs techniques rapides
- `curl localhost:8090/health` → `{"status":"ok","service":"report-service"}`.
- Routage gateway : `curl -o /dev/null -w "%{http_code}" localhost:8080/reports/warnings/pending` → **401** (et non 503).
- Base tickets : `make mongo-report-cli` → `db.tickets.find().pretty()` · `db.warnings.find()`.
- Message tombstoné par la modération : `make mongo-message-cli` → `db.messages.find({deleted_by_moderation:true})`.
- Aucune clôture auto : après un « Retirer », vérifier `status` inchangé sur le ticket (`db.tickets.findOne(...)`).
