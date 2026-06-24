# DECISIONS — WebDad / Breezy

> Architectural decisions and their **rationale**. Implementation minutiae live in the code and
> are queryable via `graphify`. This file answers *"why is it built this way?"* (defense prep).
> Was section §5 of CLAUDE.md.

---

## Didacticiel guidé + aide : moteur front maison, état « vu » côté profil-service (24/06/2026)

> Besoin : à la première connexion, une visite guidée (infobulle se promenant sur les pages,
> avançable au clic sur la cible ou via Suivant/Précédent, ignorable) ; un bouton « lampe » donnant
> accès à une page d'aide et à la relecture du didacticiel.

- **Moteur de tour custom maison (zéro dépendance).** Cohérent avec l'i18n maison et le thème
  clair/sombre (tokens). On garde le contrôle total du positionnement (`computeTooltipPosition` pur et
  testé), du spotlight (`box-shadow` projeté autour de la cible) et de la navigation cross-page. Pas de
  `react-joyride`/`driver.js` à styliser et maintenir.
- **Cibles via `data-tour`, instance visible choisie à l'exécution.** La même valeur `data-tour` est
  posée sur la variante desktop (sidebar) ET mobile (tab bar / cloche du header) ; le tooltip retient
  l'élément réellement visible (`offsetParent`). Une étape sans cible visible se rend centrée (fallback
  mobile pour les sections logées dans un menu). Cliquer la cible avance le tour (listener en phase
  capture qui neutralise l'action native), conformément à « avancer en cliquant ce qui est indiqué ».
- **État « didacticiel vu » persisté côté serveur, pas en localStorage.** Choix utilisateur : le flag
  doit survivre au changement d'appareil. `tutorial_done *bool` vit dans `profil-service` (comme
  `nsfw_enabled`), exposé/mis à jour via `GET`/`PATCH /profils/me` — **aucune route nouvelle**.
- **Auto-start proposé une fois à TOUS les comptes, donc pas de backfill.** Flag absent ⇒ `false` ⇒ le
  tour est proposé une fois (comptes existants compris) pour faire découvrir la fonctionnalité, sans
  migration risquée sur la prod. C'est sûr précisément parce que le champ est **optionnel, non-`required`
  et sans enum** : le validateur Mongo strict ne revalide pas les documents existants en échec (≠ les cas
  `visibility`/`likes_visibility` qui, eux, exigeaient un backfill — Rule 5b). Défaut explicite `false`
  posé à la création (compte neuf = réellement en première connexion).

## Certifications de profil : donnée décorative côté profil-service, staff dérivé du rôle (23/06/2026)

> Besoin : permettre aux modérateurs/administrateurs d'attribuer une certification visible à côté du
> `display_name` : jaune pour les personnalités politiques, bleue pour les personnalités publiques,
> logo Breezy pour les modérateurs/administrateurs.

- **La certification publique vit dans `profil-service`.** C'est une donnée décorative affichée avec
  le `display_name`, comme l'avatar ou la bio ; elle ne doit pas vivre dans auth/user-service. Le champ
  Mongo est `certification=none|political|public_figure`, avec défaut explicite à la création et
  backfill boot idempotent à `none` pour les profils existants (Rule 5b + validateur strict).
- **Le badge staff est dérivé du rôle, pas stocké comme certification.** Le rôle reste dans le JWT /
  auth-service ; dupliquer `staff` dans profil-service créerait une incohérence possible lors d'une
  promotion/rétrogradation. Le front affiche donc le logo Breezy dès qu'il connaît un rôle
  `moderator` ou `administrator`, sinon il affiche la certification stockée.
- **Lecture minimale du rôle public via auth-service.** Comme les posts/messages/recherches ne portent
  pas toujours le rôle, `GET /auth/users/roles?ids=...` renvoie uniquement `{user_id, role}` pour les
  rôles d'affichage staff. On évite de dupliquer le rôle dans user/profil-service et on n'expose pas
  d'e-mail, d'état de session ou de donnée sensible.
- **Mise à jour instantanée via le bus WS notification existant.** Après un changement de certification,
  profil-service émet `identity_updated` vers notification-service (secret interne, best-effort), qui
  broadcast aux clients connectés. Le badge local écoute cet événement et met à jour `certification` /
  `role` sans rechargement, en gardant le backend autoritaire pour les droits.
- **Écriture réservée aux modérateurs/admins.** Route `PATCH /profils/:userId/certification`
  protégée par JWT et contrôlée dans le handler (mod ou admin). Le front l'expose dans
  Modération › Comptes sous le menu `...`, afin que l'opération reste dans le flux de gouvernance
  existant.
- **Tooltip accessible web/mobile.** Le badge est un bouton/popover : `title`/hover/focus sur desktop,
  clic/tap sur mobile, même libellé i18n dans les 12 langues.

---

## Acceptation des CGU : consentement versionné persisté + modale bloquante (22/06/2026)

> Besoin : forcer les comptes **déjà existants** (local + prod) à accepter les CGU à leur prochaine
> connexion. Jusqu'ici, le consentement n'existait qu'à l'inscription (case à cocher + flag
> `localStorage breezy-cgu-read`), non persisté → aucune trace pour les comptes existants.

- **Consentement persisté côté back, pas en localStorage.** `credentials.terms_accepted_version`
  (INT) porte la version acceptée ; `models.CurrentTermsVersion` est la version EN VIGUEUR.
  *Pourquoi* : robuste, cross-device, auditable (RGPD via `terms_accepted_at`), vraiment
  « obligatoire » (Rule 6 : la sécurité ne dépend pas du front), et **rebumper la version
  re-déclenche le consentement de tout le monde** lors d'une révision des CGU. Le localStorage seul
  serait par-navigateur et non auditable.
- **Rétro-compatibilité = le mécanisme même du prompt (Rule 5b).** `ADD COLUMN ... DEFAULT 0` pose 0
  sur toutes les lignes existantes → tous les comptes déjà en base repassent sous `CurrentTermsVersion`
  (=1) et **doivent accepter au prochain login**. Surtout pas d'UPDATE inconditionnel (réimposerait
  l'acceptation à chaque boot). Les comptes ayant déjà consenti à la création (`Register`, OAuth
  `complete`) sont insérés directement à `CurrentTermsVersion` ; les comptes créés par un admin
  restent à 0 (ils n'ont jamais consenti → doivent accepter).
- **Propagation par claim JWT `terms_accepted`, calquée sur `must_change_password`.** Le claim est
  dérivé (`version >= CurrentTermsVersion`) à chaque émission de token ; le front lit le JWT
  (`session.ts`) et affiche une **modale bloquante** (`TermsAcceptGate`) tant qu'il est faux — zéro
  appel réseau supplémentaire, disparaît sans rechargement après acceptation.
- **`terms_accepted_version` re-chargé dans `issueTokens` (un SELECT PK centralisé)**, pas threadé
  dans les ~8 SELECT qui émettent un token. *Pourquoi* : sinon le claim retomberait faux après
  refresh/MFA/changement d'e-mail (même piège vécu avec `email_verified`). Best-effort : en cas
  d'erreur, on laisse 0 → on re-demande l'acceptation (côté sûr).
- **Modale AVEC issue « Refuser & se déconnecter » (logout), pas de blocage définitif.** Choix
  utilisateur : consentement *libre* (refuser déconnecte plutôt que de coincer l'utilisateur).
- `POST /auth/terms/accept` (JWT) écrit la version + `NOW()` et **ré-émet la paire de tokens** (comme
  `ChangePassword`) → le nouveau JWT porte `terms_accepted=true`.

---

## GIF GIPHY : capture serveur en MinIO, pas de hotlink (19/06/2026)

> Le picker GIPHY renvoie des URL CDN giphy. Question : que stocke-t-on dans le post ?

- **Le GIF choisi est capturé côté serveur et rangé dans MinIO** (`POST /gifs/capture`), le post
  stocke `/media/<id>` — **jamais l'URL giphy externe**. *Pourquoi* : (1) règle non-négociable
  « everything through the gateway, never external/presigned URLs » ; (2) le hotlink `media.giphy.com`
  renvoyait des **403** non déterministes (le CDN refuse certaines URL originales nues) et des **404**
  (entrées `fallbackGifCatalog` mortes) ; (3) la prod sert `CSP-Report-Only: default-src 'self'` →
  en *enforce*, tout `img-src` externe sera bloqué. La capture rend le média same-origin, immutable-cachable
  et immunisé au CDN giphy.
- **Anti-SSRF par allowlist d'hôtes**, pas de fetch d'URL arbitraire : seuls `https://` + hôtes
  `*.giphy.com` connus (`media.giphy.com`, `media0-4`, `i.giphy.com`) sont téléchargés. La taille est
  bornée (`maxImageBytes`), le type revalidé par magic-bytes (`KindImage`), l'owner = l'utilisateur
  capturant (suppression/RGPD identiques à un upload).
- **Pas de génération de variantes** à la capture : un GIF animé re-encodé en variante perdrait
  l'animation ; le post pointe sur l'original. Rejet d'Option B (proxy à la volée) : dépendance runtime
  giphy à chaque vue + cache plus délicat.
- **Résiduel assumé** : les vignettes du *picker* restent hotlinkées giphy (UI éphémère, popover
  ouverte) — à proxifier seulement si la CSP passe en enforce.
- Corollaire ops : `GIPHY_API_KEY` doit être posée en prod (sinon `fallbackGifCatalog`, dont des
  entrées rottent → 404).

---

## Contenu sensible (NSFW) & majorité (19/06/2026)

> Filtre NSFW : posts marquables « sensibles », floutés pour les mineurs et pour les majeurs
> qui désactivent l'affichage. Feature de la branche #74 (date de naissance + interdiction NSFW).

- **Majorité calculée serveur depuis `birth_date`, jamais stockée.** `IsAdultAt` (≥18 ans, calcul
  calendaire exact) + `HydrateViewerPolicy` posent `is_adult` et `nsfw_visible` à la lecture de
  `/profils/me` uniquement. *Pourquoi* : la majorité est **dynamique** (se débloque toute seule le
  jour des 18 ans, sans job de migration) et le front ne peut pas se faire passer pour majeur
  (calcul serveur = autoritaire, cf. Rule 6). Le seuil NSFW (18) est **distinct** de l'âge minimum
  d'inscription (13, inchangé) : on ne bloque pas l'inscription des mineurs, on filtre le NSFW.
- **Préférence `nsfw_enabled` à défaut ON, hors `$jsonSchema required`.** Un `*bool` omitempty +
  helper `NsfwEnabledOf` (absent ⇒ true) ⇒ **pas de revalidation stricte Mongo** donc pas de
  migration de la préférence (contraste avec `visibility`, cf. Rule 5b). Défaut ON = la prod
  existante continue de voir tout le contenu. La valeur stockée d'un **mineur est inerte** :
  `nsfw_visible = is_adult && nsfw_enabled` reste faux. La carte des paramètres est grisée+verrouillée
  pour un mineur.
- **DOB absente → backfill `1999-01-01` + traité adulte.** Migration boot `backfillBirthDate`
  (idempotente) pour les profils antérieurs au champ et les comptes sans date. *Pourquoi adulte* :
  ne pas casser l'expérience de la prod ni des comptes sociaux ; l'onboarding OAuth collecte de
  toute façon la vraie date (bloquant) et le register Breezy aussi.
- **Flou côté front, politique viewer autoritative serveur (pas de stripping).** Le post-service
  **délivre** le post marqué (`nsfw: true`) ; le front le floute si `!nsfw_visible`. *Pourquoi pas
  de stripping serveur pour les mineurs* : `nsfw_visible` est calculé serveur (le front ne triche
  pas sur la majorité), le floutage est un filtre d'affichage façon X/Twitter, et ça évite de
  complexifier le cache/feed. Le NSFW **n'est pas** une barrière de visibilité dure (le post reste
  public, juste visuellement masqué) — distinct de `is_hidden`/`visibility`.
- **Marquage : auteur à la création, modo/admin ensuite.** `nsfw` dans `CreatePostRequest` (l'auteur
  pose via le bouton du composer) ; après publication, seule la modération (dé)marque via
  `PATCH /posts/:id/nsfw` (`ModeratorOnly`). *Pourquoi* : simple, cohérent avec la modération
  (miroir de `is_hidden`/`hidden_by`), et l'auteur ne peut pas retirer un marquage posé par un modo.

---

## Sécurité — durcissement post-pentest (18/06/2026)

> Suite à un pentest boîte noire externe (« riveta ») + un audit boîte blanche interne complet
> (rapport gitignoré `PENTEST.md`). Le cœur (contrôle d'accès, JWT, injections SQL/NoSQL, XSS,
> upload, secrets at-rest, isolation réseau) tient ; les correctifs ci-dessous durcissent le
> périmètre d'authentification et la configuration.

- **Rate-limiting applicatif plutôt que Fail2Ban (RIV-001).** Limiteur en mémoire sans dépendance
  (`auth-service/internal/middleware/ratelimit.go`), par IP cliente (`c.ClientIP()` lit le
  X-Forwarded-For posé par Caddy puis la gateway). Login + `mfa/verify` = 10/5 min ; register +
  forgot + resend = 5/15 min. *Pourquoi applicatif* : portable, indépendant de l'hôte, testable,
  conscient du « par compte » à terme — là où Fail2Ban (edge, par IP) reste un filet réseau
  complémentaire. In-memory suffit (une instance par service en compose ; backend Redis si on
  passe multi-instances).
- **Anti brute-force MFA : challenge brûlé après 5 codes faux (BRZ-001).** `VerifyMFA` ne consommait
  pas le challenge sur code erroné (fenêtre 5 min) → 10⁶ codes brute-forçables même distribué. On
  compte les échecs par challenge (en mémoire) et on invalide le challenge au 5ᵉ. Le rate-limit par
  IP seul ne couvrirait pas un attaquant multi-IP.
- **`email_verified` comme claim JWT + gate `VerifiedOnly` (RIV-002).** Le blocage de vérification
  n'existait qu'au login ; le token émis à l'inscription accédait aux routes mutantes. Le claim est
  désormais propagé sur **tous** les chemins d'émission (register=false, login/verify/refresh/oauth
  = état réel/true) et un middleware `VerifiedOnly` garde la **création de contenu** (posts,
  commentaires, conversations, messages). *Choix* : gater la création de contenu (abus = spam
  public) plutôt que toutes les routes, pour ne pas casser le provisioning paresseux
  (`POST /users`/`/profils`, `GET /users/me`) qui s'exécute avec un token non encore vérifié.
  *Non-régression* : le refresh re-sélectionne `email_verified` en base, l'OAuth le force à true →
  un utilisateur vérifié n'est jamais bloqué.
- **Annuaire complet authentifié (RIV-003).** `GET /users` (énumération de toute la base) exige une
  session. Les lookups ciblés (`/search`, `/suggestions`, `/by-username`, `/:id`) restent publics
  car nécessaires au login (résolution username→id) et aux aperçus de profil ; `:id` valide
  désormais la forme UUID (404 propre au lieu de 500, RIV-007).
- **En-têtes de sécurité au reverse-proxy (RIV-004).** HSTS, X-Frame-Options:DENY, nosniff,
  Referrer-Policy, Permissions-Policy posés par Caddy (`import securite`). **CSP en Report-Only
  d'abord** : le script de thème inline (`layout.tsx`) serait bloqué par `script-src 'self'` ; on
  observe avant de poser un nonce/hash puis de basculer en enforce. `X-Powered-By`/`Server` masqués
  (Caddy + `poweredByHeader:false`).
- **BRZ-002 écarté.** La prod utilise un vrai `JWT_SECRET` (hex), distinct du placeholder du `.env`
  gabarit (gitignoré) → pas de forge de JWT. (Note durcissement : viser 32 hex / 256 bits.)

## Store utilisateur connecté (front, 18/06/2026)

- **Un `CurrentUserProvider` (Context React), pas zustand.** L'identité JWT était déjà centralisée (`lib/session.ts`) mais le **profil** du user connecté (username/display_name/avatar/rôle) était refetché indépendamment par chaque composant visible (sidebar + header + composer = 6 requêtes au boot), et `useSession()` était instancié 10+ fois (chacun son listener + décodage JWT). Le store fait **un seul** `getMyProfil()`+`getMe()` au montage, écoute `SESSION_CHANGED` une seule fois et `subscribeProfilUpdated` (MAJ sans refetch), et expose `useCurrentUser()` (`session`, `profil`, `isAdmin`/`isModerator`, `usernamePending`, `preferredLocale`, `refresh`). **Context plutôt que zustand** : cohérent avec les providers existants (`LanguageProvider`, `NotificationsProvider`, `MessagesProvider`…), zéro dépendance ajoutée. Monté en tête de `(app)/layout.tsx` (espace authentifié) → gère le visiteur (`null`). `lib/session.ts` reste la source sync (libs non-React comme `posts.ts`/`messages.ts` lisent `currentUserId()` ; les définitions dupliquées y réexportent désormais celle de `session.ts`). `language-provider` (root layout, parent du provider) garde son `getMe()` propre.

## Blocage utilisateur (19/06/2026)

- **Le blocage est une relation du graphe social, donc user-service.** `blocks(blocker_id, blocked_id)` vit en PostgreSQL avec `follows`/`follow_requests`, pas dans profil-service : il relie deux identités et doit couper les relations sociales existantes dans les deux sens. Les routes publiques JWT sont `POST/DELETE /users/:id/block` et `GET /users/me/blocks`; une route interne expose `HasBlocked` pour les services.
- **L'effet de confidentialité reste côté serveur dans post-service.** Le front masque immédiatement les cartes pour l'UX, mais la vraie barrière est `canReadAuthor`: le post-service interroge user-service pour savoir si le viewer a bloqué l'auteur et exclut ses posts du feed, du détail, des stats, du profil, des réponses et des likes. Même logique que les profils privés : la sécurité ne dépend jamais du front.
- **Déblocage = retour aux règles normales.** Débloquer ne recrée pas les follows supprimés par le blocage ; l'utilisateur revoit les contenus publics, et un profil privé redevient soumis au follow/acceptation habituels.

## Observability

- **Structured logging — slog + X-Request-Id (user-service pilote, 13/06/2026).** `log/slog` stdlib (Go 1.21+, aucune dépendance externe). Format JSON en `release`, texte en `debug/test` (lisible humain). Niveau depuis `LOG_LEVEL`. Trois middlewares Gin dédiés : `RequestID` (lit ou génère un UUID hex 16 B, propagé dans la réponse), `Recovery` (panic → `slog.Error` + 500 sans stack exposée), `RequestLogger` (une ligne/requête avec method, path sans query, status, latency_ms, client_ip, request_id, user_id). `JWTAuth` pose `user_id` dans le contexte Gin pour que `RequestLogger` corrèle l'utilisateur. `gin.Default()` remplacé par `gin.New()` + chaîne explicite. Ce motif sera répliqué à l'identique sur les autres services.

## Moderation & reporting

- **Domaine « tickets de signalement » = nouveau `report-service` dédié (16/06/2026).** Un signalement
  porte sur une entité possédée par un AUTRE service (post, message, profil) → il n'appartient à aucun
  d'eux. Plutôt que de polluer post/message/profil (entorse à « une donnée = un service »), un service
  autonome (Go+MongoDB, port 8090, préfixe gateway `/reports`) possède les tickets et avertissements.
  Renforce la cohérence microservices (critère de notation « architecture cohérente »).
- **Agrégation : un ticket parent par entité (modération), bug autonome.** Index Mongo **unique partiel**
  `(entity_type, entity_id)` filtré sur `category=moderation` → les N signalements d'un même contenu sont
  empilés dans `reports[]` d'un seul ticket (`report_count` + `reason_tags` dénormalisés via `$inc`,
  upsert atomique). Les rapports de **bug** (motif « Bug technique ») sont des **tickets autonomes**
  (`entity_type=app`, pas de clé d'entité à dédupliquer) — décision produit validée avec l'utilisateur.
- **Auto-masquage des posts trop signalés (17/06/2026).** Un POST de modération qui atteint un **seuil** de
  signalements est automatiquement masqué (sort des fils, en attente d'une décision de modérateur) ; les
  **bugs** (onglet admin) en sont **exemptés** (déjà des tickets autonomes). Le report-service détient le
  compteur mais la visibilité est la responsabilité du post-service → **appel serveur-à-serveur off-gateway**
  `POST /internal/posts/:id/auto-hide` authentifié par `X-Internal-Secret` (même secret partagé et même esprit
  best-effort fire-and-forget que l'émission de notifications ; échecs **loggés** car la visibilité est
  sécurité-sensible). Nouveau champ post `auto_hidden` **distinct de `is_hidden`** (retrait manuel → corbeille) :
  un post auto-masqué n'apparaît PAS dans la corbeille de modération. Idempotent (action `auto_hidden` journalisée
  une fois). Le masquage réel s'applique côté post-service (barrière serveur), jamais sur la foi du front (§6).
- **Verrou de validation : statut terminal `approved` (17/06/2026).** Quand le modérateur juge l'entité
  **conforme** (« ne doit pas être signalée »), le ticket passe au statut **terminal `approved`** : le post est
  démasqué (`auto-unhide`) ET tout nouveau signalement est **refusé** (`ErrReportingLocked` → 409). Distinct de
  `closed` (qui peut se rouvrir au seuil) : `approved` est **définitif** (choix produit validé). Le verrou vit
  **dans report-service** (vérif `GetModerationByEntity` avant l'upsert) → pas de dépendance cross-service pour
  bloquer. Action front « Valider (conforme) » séparée de « Clôturer »/« Retirer ».
- **Seuil d'auto-masquage réglable par l'admin (17/06/2026).** Le seuil n'est pas une constante mais une
  **config runtime** : collection **singleton `settings`** du report-service (`_id="global"`,
  `auto_hide_threshold` int32, **0 = désactivé**, défaut **5**), seedée au boot par `$setOnInsert` idempotent
  (ne **réécrase jamais** un seuil déjà réglé au redémarrage — règle 5b). Lecture mod+admin, écriture **admin
  seul** (`PATCH /reports/settings`), réglée dans **Admin › Paramètres**. Le service propriétaire de la logique
  de signalement reste la seule source de vérité du seuil (pas de duplication).
- **Réouverture automatique à seuil (16/06/2026).** Un ticket clôturé ne doit pas être rouvert par un unique
  re-signalement (sinon la décision du modérateur est triviale à défaire), mais une récidive soutenue doit le
  rouvrir. Compteur `reports_since_closed` (`$inc` à chaque signalement, remis à 0 à **chaque** changement de
  statut) ; quand il atteint `ReopenThreshold` (=2) sur un ticket `closed`, réouverture auto (statut `reopened`,
  action `auto_reopen` « Système »). Couplé à « un signalement par utilisateur », il faut **2 personnes
  distinctes nouvelles** → un même compte ne peut pas rouvrir en spammant.
- **Un seul signalement par (utilisateur, entité) (16/06/2026).** Le `$push` du signalement enfant est
  conditionné par un filtre `reports.reporter_id $ne <moi>` ; si le rapporteur est déjà présent, l'upsert
  tente un insert → rejeté par l'index unique partiel (E11000) → **409 `ErrAlreadyReported`**. Une E11000
  peut aussi venir d'une course (deux 1ers signalements simultanés) → **retry unique** qui distingue course
  (empile) et vrai doublon (409). Atomique, sans lecture-puis-écriture vulnérable aux races.
- **Catégorie déduite du MOTIF, formulaire unique.** « Bug technique » est un motif du même `ReportDialog` :
  le choisir bascule en catégorie `bug` (limite **500**, onglet Administration) ; les autres motifs →
  `moderation` (limite **255**, onglet Modération). Un seul formulaire partout (posts/profils/messages),
  conforme à l'énoncé. Motifs **bornés** (enum fermé) car `reason_tags.<motif>` est une clé Mongo
  (anti-injection de champs arbitraires) ; longueur comptée en **runes** (caractères, pas octets).
- **Le ticket conserve l'entité, même pour un bug (16/06/2026).** Un bug signalé SUR un post/profil garde
  `entity_type`+`entity_id` (au lieu de retomber sur `app`) → le détail affiche le contenu réel (post embarqué,
  carte profil cliquable) pour que mod/admin puissent juger et investiguer. `app` n'est utilisé que pour un bug
  applicatif sans entité.
- **Messages E2EE : divulgation par le signaleur, pas déchiffrement serveur (16/06/2026).** Le serveur de
  messagerie reste **aveugle** (DM/groupes admin-proof). Pour modérer un message privé (ex. haine), le **signaleur,
  qui en est destinataire**, joint **volontairement** la copie en clair (`disclosed_content`) de CE message au
  signalement (avis de transmission affiché). Le serveur ne déchiffre jamais de lui-même et ne peut pas lire un
  message arbitraire — seul un participant peut révéler un message précis, pour ce signalement. Conforme à l'E2EE
  (analogue au « report » de WhatsApp/Signal) tout en permettant l'action de modération.
- **Suppression d'un message signalé par la modération (16/06/2026).** Nouvel endpoint message-service
  `DELETE /messages/moderation/:messageId` (mod/admin, garde de route), qui tombstone le message **par son seul
  id** — la modération n'est pas membre de la conversation et le ticket ne stocke que l'id du message. Le message
  porte `deleted_by_moderation=true` ; les participants reçoivent l'événement WS `message_updated` et voient
  « Ce message a été supprimé par la modération ». **E2EE intact** : le serveur ne lit pas le contenu, il pose juste
  le tombstone (vide `ciphertext`/`nonce`, comme la suppression « pour tous » existante).
- **Réutilisation maximale de l'existant.** « Tweets supprimés » (soft-delete post-service `is_hidden`),
  bannissement (auth `is_active` + visibilité user) et rôles/gating (mod ne peut bannir/supprimer un
  admin) **préexistaient** — la modération s'y branche (retrait = soft-delete, transfert = changement de
  `category`) au lieu de les réimplémenter.
- **Avertissement asynchrone = pull (poll), pas push.** Un Warn est persisté côté `report-service` ;
  `WarningsGate` (layout `(app)`) interroge `GET /reports/warnings/pending` au montage **et au retour de
  focus**, et affiche une **modale bloquante** acquittée par `POST .../ack`. Simple, robuste (pas de WS
  dédié), et « intercepte la prochaine requête/connexion » comme demandé.

## Platform & frontend

- **Frontend = Next.js 14 (App Router) + TS + Tailwind/shadcn (slate).** Imposed stack.
- **Route groups** `(auth)` / `(app)` / `(legal)`: separate public/private layouts without
  polluting URLs; `(legal)` stays outside the session guard so it's reachable while logged out.
- **Config single source of truth:** `lib/config.ts` (gateway URL) + `lib/routes.ts` (route constants); no hardcoded strings.
- **`apiUrl()` client vs server base:** client = `NEXT_PUBLIC_API_URL` (published port);
  server (in-container route handlers) = `API_INTERNAL_URL` (Docker service name). In a container
  `localhost` is the container itself, so server fetches must target the Docker service name.
- **Profile activity belongs to profil-service, not user-service.** The “online / last connection”
  signal is a public profile decoration with a user-controlled privacy switch, so it lives beside
  `visibility` and `likes_visibility` in Mongo. That lets profil-service enforce the public read
  rule before returning a profile: `last_login_at`/`is_online` are omitted when
  `activity_visibility=private`, and private profiles reveal activity only to the owner or accepted
  followers. Private profile updates do **not** mutate the activity preference; privacy is enforced
  at read time so the switch remains an independent choice. Login/OAuth/email-verification BFF
  handlers mark activity best-effort via `PATCH /profils/me/activity` after a real session entry;
  logout calls `PATCH /profils/me/activity/offline` before clearing the access token; refresh does
  not count as a new connection.

## Auth

- **Access (5m) localStorage + refresh (24h) httpOnly cookie via BFF.** Assumed trade-off:
  access is XSS-exposed but short-lived; refresh is non-stealable. BFF-managed cookie (same-origin)
  avoids cross-origin CORS/SameSite complexity. Refresh is **single-flight** to avoid refresh storms on simultaneous 401s.
- **Admin-created accounts = temporary password + provisional username (forced fixups via JWT/`/users/me`).**
  An admin creates an account from `/admin` (auth `POST /auth/users`, AdminOnly): credentials are written
  `must_change_password=true` (a temporary password, emailed best-effort) and **email verification is
  mandatory** (`email_verified=false`): the welcome email carries the verify link + the temp password, and
  clicking it verifies the address **and** opens the session (`VerifyEmail` now propagates the flag) → the
  user lands on the feed with the change-password modal. A **dev/local shortcut** `ADMIN_CREATE_AUTO_VERIFY=true`
  marks the account verified up-front (real mail goes online) so login works directly with the temp password.
  The flag rides in the **JWT claims** (and is re-read on
  `/refresh`) so the front shows a blocking change-password modal on any page **without an extra call**;
  `POST /auth/password/change` verifies the current password, clears the flag, revokes other sessions and
  re-issues a flagless token (modal disappears, no reload). The requested **username** is provisioned in
  user-service (`POST /users/admin`); if taken it is suffixed `_<8 hex>` with `username_pending=true`
  (exposed by `/users/me`, cleared on an effective `PATCH /users/me`), driving a second blocking modal.
  Orchestration lives client-side in `lib/admin.createAccount` (admin bearer, same pattern as the RGPD
  eraser) — one datum, one service; steps 1-2 critical, profil creation best-effort.
- **Refresh token = opaque random, stored SHA-256 hashed, rotated on `/refresh`, revoked on `/logout`.**
  Opaque + DB-stored = revocable (unlike self-contained JWT); hashing means a DB leak yields no usable token;
  rotation is the basis for future reuse-detection. The **back never auto-renews** — it signs `exp` and
  401s on expiry (`code: token_expired` distinguishes expiry→refresh from invalid→no-refresh); the **front drives** refresh.
- **Inter-service auth:** JWT in header (grading requirement).
- **Username login orchestration stays in the BFF.** `username` remains owned by user-service:
  the Next BFF resolves `username -> user_id` through the gateway, then calls auth-service with
  `user_id + password`. Auth-service still owns only credentials/JWT and never joins user data.
- **Login with Google = OIDC Authorization Code, code→tokens exchanged server-side in auth-service**
  (`golang.org/x/oauth2` + `go-oidc/v3`). The front only obtains the authorization URL and relays the callback
  `code`; the ID token (signature via JWKS, issuer, audience=client_id) is verified **server-side** — never trusting
  a front-supplied token. Anti-CSRF `state` is server-generated, stored by the front, re-checked at callback.
  `email_verified` is required (blocks account takeover by email matching): the claim is decoded as `*bool` and an
  absent or explicit-`false` value is rejected (Google always sends it).
  *(Microsoft/Entra support — multi-tenant `common` issuer with manual `tid` verification — was implemented then
  removed on owner's request; only Google remains. See CHANGELOG 10/06/2026.)*
- **Generic provider registry, two families** (`internal/oauth`, map `defs`): adding a provider is one map entry +
  env vars; a provider with no `CLIENT_ID` is skipped (→ 404). Two `kind`s behind the same `AuthURL`/`Exchange`
  interface: **`kindOIDC`** (Google) — OIDC discovery + `id_token` verification (lazy discovery at first
  use, not at boot → offline-resilient); **`kindOAuth2`** (GitHub) — plain OAuth2 (no `id_token`):
  code→`access_token`, then a **userinfo** call (`api.github.com/user`) mapped to `Identity{Subject,Email}` via a
  per-provider `userInfoMap`. This keeps non-OIDC providers as config, not code.
  *(LinkedIn — initially planned as a second `kindOIDC` provider — was dropped: its dev app requires an attached
  company Page, heavy for a project. Replaced by GitHub. **Facebook & Spotify** — implemented as `kindOAuth2`
  providers, removed on owner's request 19/06/2026; only Google + GitHub remain. The DB enum keeps the inert
  `facebook`/`spotify` values, Postgres not allowing `DROP VALUE`.)*
- **GitHub specifics within `kindOAuth2`.** GitHub's `/user` returns `email:null` when the address is private, so an
  optional `emailsURL` fallback (`/user/emails`, scope `user:email`) fetches the **primary verified** address (no
  provisioning on an unverified email). Its `id` is a JSON number (helper `idField`), and `api.github.com` requires a
  `User-Agent` header (set by the shared `authedGetJSON`). These three points are the only provider-specific code;
  everything else stays in `defs`.
- **OAuth2 (non-OIDC) email is trusted as verified.** GitHub exposes no `email_verified` claim; since
  the provider authenticated the user and returns a confirmed address (for GitHub we explicitly pick a *verified*
  one), `EmailVerified=true` is set for it (the handler still rejects an **absent** email). OIDC providers keep the
  strict `email_verified` check (added/explicit-`true`).
- **CGU acceptance gate lives in the BFF/front, and new OAuth sign-up is pending until final submit.**
  The legal read marker is local UX (`/cgu` scrolled to bottom → checkbox/button unlock) while the
  Next BFF enforces `acceptedTerms=true` for classic `/api/auth/register`. For OAuth sign-up,
  `/auth/oauth/:provider/exchange` verifies the provider identity but, when no account exists, writes only
  a short `oauth_signup_tokens` row and returns `onboarding_required + pending_token + email` — **no
  credential, no user/profile, no JWT**. The callback stores the pending data in `sessionStorage` and
  redirects to the blocking public `/auth/oauth/terms` page (back navigation trapped, unload warned) which
  collects username/date/CGU; only then
  `POST /auth/oauth/:provider/complete` consumes the pending token in a transaction to create the
  credential and issue the first session; the BFF provisions user-service/profil-service immediately
  after. Existing OAuth logins still issue tokens directly.
- **Account reconciliation by email:** existing account → connect + fill `provider_subject` (only if unset, no
  hijack); absent → create with `password NULL`. Classic login on a password-less account is refused with a clear
  409 (`ErrNoLocalPassword`) steering the user to the external provider. Schema migrated via idempotent `ALTER`
  (`provider` enum default `'local'`, `provider_subject`, `password` nullable) — `'local'` keeps prior behavior intact.

## Email (vérification, changement & reset)

- **`mail-service` dédié pour le transport SMTP ; la logique-token reste dans auth-service (PG).**
  `POST /internal/send` hors gateway, authentifié par `MAIL_INTERNAL_SECRET` (`X-Internal-Secret`),
  appelé en best-effort par auth — symétrique de notification-service (2nd consommateur d'événements
  internes). Secrets SMTP isolés dans `mail-service/.env`, mailer réutilisable.
- **Tokens vérif/reset = opaques aléatoires, stockés SHA-256 hachés, usage unique (`used_at`), TTL
  vérif 24h / reset 1h.** Réutilise le pattern refresh-token : révocables sans denylist (vs JWT
  auto-portant), une fuite de table ne livre aucun token. Table unique `account_tokens(purpose enum
  'verify'|'reset'|'email_change', …)`. Toute nouvelle demande invalide les précédents du même `(user_id, purpose)` ;
  un reset réussi **révoque toutes les sessions** (`DELETE refresh_tokens`) — un changement de
  mot de passe doit déconnecter partout. **Un reset réussi pose aussi `email_verified=true` :**
  cliquer le lien (envoyé à l'adresse du compte, TTL 1h) prouve la possession de la boîte, donc
  débloque un compte non vérifié sans vérification séparée — le reset est un second chemin de
  preuve d'adresse, équivalent au lien de vérification.
- **Changement d'e-mail en deux temps, sans couper l'accès.** `POST /auth/email/change/request`
  (JWT) place la nouvelle adresse normalisée dans `credentials.pending_email` (index unique partiel)
  et lui envoie un jeton `email_change` 24h. L'adresse courante reste l'identifiant actif tant que
  le lien n'est pas cliqué : une faute de frappe ne verrouille donc jamais le compte. La confirmation
  publique verrouille le jeton (`FOR UPDATE`), promeut `pending_email`, consomme le jeton et révoque
  les refresh tokens ainsi que les anciens jetons vérif/reset liés à l'ancienne boîte dans **une transaction** ; elle émet ensuite une nouvelle session dont le JWT
  porte la nouvelle adresse. Si l'adresse est devenue indisponible entre demande et confirmation,
  l'unicité SQL refuse la promotion sans altérer l'adresse actuelle.
- **`forgot-password` = anti-énumération stricte** : `ForgotPassword(email)` renvoie TOUJOURS `nil`
  et le handler répond TOUJOURS `200` générique, que le compte existe, soit actif, ou non ; le mail
  n'est envoyé (best-effort) que pour un compte existant ET actif. La page front affiche le même
  écran de confirmation dans tous les cas.
- **Login non-vérifié = blocage dur** (`403 email_not_verified`, aucun token émis), vérifié
  *après* le bcrypt pour ne pas révéler l'existence du compte.
- **Vérification d'e-mail = auto-login (révise la position « pas de session avant login »).**
  `POST /auth/verify-email/confirm` consomme le token, pose `email_verified=true` *et* émet une
  session (access + refresh, comme `/login`) ; le BFF Next pose le cookie refresh httpOnly et renvoie
  l'access token, la page `verify-email` stocke ce dernier et redirige vers le **feed**. Justifié par
  la même logique que le lien de reset : cliquer le lien (token usage unique, TTL 24h, envoyé à
  l'adresse du compte) **prouve la possession de la boîte** → facteur d'authentification suffisant.
  Garde : un compte désactivé (`is_active=false`) reste bloqué (`403`, aucune session). Le no-session
  ne vaut donc plus que pour le **register** (provisioning serveur, ci-dessous), pas pour la vérif.
- **Provisioning préservé au register, sans session client (décision Phase 1).** `auth/register`
  continue d'émettre les tokens, mais le BFF Next s'en sert **uniquement côté serveur** pour
  provisionner l'identité (`POST /users` + `POST /profils`, avec `username`/`birth_date`/`gender`
  du formulaire) puis les **jette** : il ne pose PAS le cookie refresh et ne renvoie PAS l'access
  token. Le client n'obtient donc **aucune session** et atterrit sur une page publique « consulte ta
  boîte mail » ; le blocage réel est appliqué au login. *Choisi plutôt que « register sans token +
  provisioning au 1er login » : cette variante imposerait de charrier `birth_date`/`gender` (qui
  n'existent que dans le formulaire register) jusqu'au login via un stockage temporaire — plus lourd
  et fragile.* **Admin seedé forcé `email_verified=true`** (pas de vraie boîte) pour garder un compte
  démo. Le renvoi de mail de vérif est accessible depuis la page login (les users existants passent
  `email_verified=false` et doivent se vérifier).
- **Liens dans le mail → pages front** (`APP_BASE_URL/verify-email|verify-email-change|reset-password?token=…`), pas
  l'API directement : maîtrise de l'UX (succès/expiré/erreur), API qui reste JSON-only.
- **Corps HTML des e-mails = coquille de marque partagée** (`services/mail_template.go`,
  `brandedEmailHTML`, pure & testée) : layout table + styles inline (compat Outlook/Gmail/Apple Mail),
  direction graphique clear mode (dégradé `#8D3DFF→#5B6CFF→#47D9FF`, logo via `APP_BASE_URL`), bouton
  « bulletproof » (repli couleur solide). Vérif et reset la réutilisent → cohérence visuelle, un seul
  point de maintenance. La version **texte** reste sobre (délivrabilité).
- **Dev sans SMTP configuré = transport console** : le mailer logge le mail + le lien sur stdout au
  lieu d'envoyer (zéro dépendance Gmail en dev, on clique le lien depuis les logs).

## MFA (double authentification TOTP, 18/06/2026)

- **TOTP seul, opt-in, sans codes de secours (tranché avec l'utilisateur).** RFC 6238 via `pquerna/otp`,
  compatible Microsoft/Google Authenticator/Authy. La MFA est **facultative** : un compte sans MFA conserve
  le login inchangé. Périmètre volontairement réduit au TOTP — **pas de codes de secours** pour livrer/tester
  vite. *Limite assumée :* perte du téléphone = verrouillage (la désactivation exige une session, donc un code) ;
  un échappatoire (codes de secours, ou reset admin) est une évolution ultérieure.
- **Secret TOTP chiffré at-rest (AES-256-GCM), clé hors-DB.** `credentials.mfa_secret` stocke le secret
  **chiffré** sous `MFA_ENCRYPTION_KEY` (base64 32 o, `.env` racine, jamais en dur — règle 5). Une fuite de la
  table ne livre donc aucun secret sans la clé. **Pattern « nil = MFA off »** (symétrique du mailer) : clé absente
  → `mfaCipher` nil → endpoints `/auth/mfa/*` en 503, auth reste bootable. Schéma rétro-compatible (règle 5b) :
  `mfa_enabled BOOL DEFAULT false` (aucun backfill) + `mfa_secret` nullable. États portés par le couple
  (secret présent + `mfa_enabled`) : setup non confirmé / actif / désactivé (secret NULL).
- **QR généré côté serveur (data URI), pas côté front.** `node_modules` du front est root-owned → interdiction
  d'ajouter une dépendance npm (même contrainte que la roue chromatique maison). `key.Image()` de `pquerna/otp`
  (via `boombuler/barcode`) rend le PNG ; `/auth/mfa/setup` renvoie `{secret, otpauth_url, qr_data_uri}` et le
  front pose juste un `<img>`. Le `secret` permet la saisie manuelle si le QR n'est pas scannable.
- **Login en deux temps via un challenge court, réutilisant `account_tokens`.** Quand `mfa_enabled`, le login
  valide le mot de passe (et `email_verified`) mais **n'émet aucun JWT** : il crée un jeton `purpose='mfa_challenge'`
  (TTL 5 min, opaque haché, même infra que verify/reset/email_change — CHECK étendu idempotemment) et renvoie
  `{mfa_required, challenge}`. `POST /auth/mfa/verify` (**publique** : le challenge, preuve que le mot de passe est
  déjà passé, tient lieu d'auth) valide le TOTP **avant** de consommer le challenge (lecture `FOR UPDATE`, validation,
  puis `used_at` dans une transaction) → **un code faux ne brûle pas le challenge**, l'utilisateur réessaie dans la
  fenêtre. `Login`/`LoginByUserID` renvoient désormais un `LoginOutcome` (session OU challenge) au lieu du quadruplet.
- **Désactivation par code OU mot de passe ; activation confirmée par un code.** `enable` valide un premier TOTP
  (preuve que le secret est bien enrôlé) avant `mfa_enabled=true`. `disable` accepte un TOTP courant **ou** le mot de
  passe du compte (un compte OAuth sans mot de passe ne peut donc se désactiver qu'au TOTP). Validation TOTP avec
  **skew ±1** (dérive d'horloge) et trim des espaces (saisie).
- **MFA = login par mot de passe uniquement.** Les comptes OAuth s'authentifient via leur provider (2FA propre au
  provider) et ne passent pas par `/auth/login` → la MFA Breezy ne s'y interpose pas. Cohérent avec « un datum, un
  service » : le second facteur vit dans auth-service à côté des credentials.
- **Front :** section « Sécurité » dans `/parametres` **au-dessus du mot de passe** (`MfaSettings` dans
  `UserAccountSettings`), pilotée par un **interrupteur on/off style iOS** (markup partagé avec `visibility-settings`) :
  off→on déploie le QR + le champ code, on→off déploie la confirmation (code/mot de passe) ; l'état coché ne bascule
  qu'après confirmation côté serveur. Client `lib/mfa.ts` sur `apiFetch` (appels authentifiés directs à la gateway, aucun cookie
  touché). Seuls login + `mfa/verify` passent par le BFF (cookie refresh) : `verify` pose le cookie et provisionne
  comme le login normal. Écran de challenge intégré à la page login (bascule mot de passe → code). i18n FR/EN.

## Gateway

- **Thin reverse proxy (stdlib).** Prefix→URL table, transparent forward, preserves prefix. Mince + "we built it"
  (good for defense) vs per-endpoint BFF handlers (reserved for multi-service aggregation).
- **No trailing-slash redirect:** each service exposed via 2 routes (bare prefix + `/*path`) so collection
  endpoints (`POST /users`) don't 307 into a route the service doesn't know.
- **Validation + propagation JWT au gateway — premier filtre « valider-si-présent » (19/06/2026).** Middleware global
  `PropagateJWT` (`api-gateway/internal/middleware/jwt.go`) appliqué AVANT le reverse proxy. Politique : aucun token →
  passe-plat (login, refresh par cookie, vue visiteur publique) ; token présent mais mal formé / invalide / expiré →
  **401 au plus tôt**, la requête n'atteint jamais le service (décharge les services, centralise le premier filtre) ;
  token valide → injecte `X-User-Id` / `X-User-Role` / `X-Email-Verified` depuis les claims vers le service.
  **Choix « valider-si-présent » plutôt qu'une allowlist publique ou une table de politique par route** : zéro config
  par route au gateway, ne casse jamais les routes publiques / `OptionalJWTAuth` (vue visiteur), et n'oblige pas à
  dupliquer le routage de chaque service (dérive). **Anti-spoof non négociable :** les en-têtes d'identité entrants
  sont TOUJOURS strippés en premier (même sans token) — seul le gateway peut les renseigner, sinon un client les
  usurpe. **Défense en profondeur conservée :** chaque service revalide le JWT lui-même (même `JWT_SECRET` partagé) ;
  les en-têtes propagés sont posés pour usage futur, pas comme unique source de confiance. Le `claims` du gateway gagne
  `email_verified` ; `AdminJWT` refactoré pour partager les helpers `parse`/`bearerToken`.

## Provisioning

- **Lazy user provisioning:** `users` is filled by `POST /users` (BFF at register, persists chosen handle)
  + upsert on `GET /users/me`, **not** by auth at register. Zero auth↔user coupling, autonomous service.
  Username pre-checked before signup; collision resolved by successive candidates.
- **Profil: `POST` = the ONLY creation (`display_name` required); `GET /profils/me` is read-only (404 if absent).**
  No write-on-GET, no guessed display_name. Front handles the 404 (POST-if-absent). profil-service never calls
  user-service at runtime; the BFF aligns `display_name = username` at register.
- **Legacy OAuth profil-absent onboarding gate (front).** Accounts created before the pending-token flow may still
  have credentials/users but no profil. The **profil-absent signal** (`GET /profils/me` 404) still drives the
  authenticated `(app)` `OnboardingGate` as a repair path. New OAuth sign-ups no longer enter this state: they stay
  unauthenticated on `/auth/oauth/terms` and are provisioned only after username/date/CGU are submitted.

## Data ownership

- **profil-service owns decorative/editable fields incl. `visibility`; user-service owns identity +
  social graph; `role` lives in the JWT.** One source of truth per datum; any profil write = one `PATCH /profils/me`,
  no inter-DB transaction.
- **Profile visibility defaults to public, then is user-configurable in settings.** `POST /profils` always stores
  `visibility=public`; `/parametres` calls `PATCH /profils/me` with `public|private`. This keeps signup friction low
  and leaves the privacy barrier enforceable server-side by post-service.
- **Nationality is stored as an ISO 3166-1 alpha-2 code in profil-service, never as a translated label.** The
  searchable edit combobox loads the public FIRST country catalogue through cached BFF route `/api/countries`;
  `Intl.DisplayNames` localizes names in FR/EN. The external API is only a catalogue source: persisted profiles
  and profile rendering remain functional if it is unavailable.
- **`ProfilDetails` aggregation (read) is composed by the caller** (front/BFF), not the service — decouples
  service from aggregation, no routing change.
- **Each service owns its schema** (`EnsureSchema` at boot, idempotent, Mongo `collMod` resync). Single source
  of truth + autonomy (`make run` against a blank DB). Resolved a validator-drift class of bug.
- **Legacy data normalization at boot (not just the validator).** A service owning its schema must also own the
  *conformity of its existing data*, because Mongo (`validationLevel: strict`) re-validates the **whole document**
  on every write: a single legacy field that violates the current `$jsonSchema` bricks *all* future edits of that
  doc (`DocumentValidationFailure` → 500). Concrete case: profiles created before `visibility` (PR #204) had
  `visibility: ""`, outside the `{public, private}` enum → every `PATCH /profils/me` failed in prod (clean local
  data hid it). Fix = an idempotent migration step in `EnsureSchema` (`normalizeLegacyProfiles`: empty/absent
  `visibility` → `public`, `bypassDocumentValidation`), same family as `dropLegacyIndexes`/`collMod` — prod
  self-heals on deploy. Chosen over `validationLevel: moderate`, which would merely *tolerate* dirty data instead
  of cleaning it. Companion: the handler's default 500 branch now logs the **raw** error so a future validation
  failure is diagnosable in seconds instead of being an opaque 500.

## Social graph

- **Edge table `follows` + separate `follow_requests`.** A pending request to a private profile is NOT a relation
  (no edge until accepted). Accept = atomic transaction (delete request + insert edge) avoiding "accepted but not following".
  Counters are **calculated (COUNT)** on detail reads; denormalization deferred until load requires it.
- **Identity cooldown architecture posed now, enforcement off by default.** `display_name_changed_at` /
  `username_changed_at` recorded only on real change; refusal (429) gated by env (`*_CHANGE_COOLDOWN`, default 0 = off).
- **Profile display names use a restricted Unicode character set.** A newly created or genuinely changed
  `display_name` accepts Unicode letters/combining marks, digits, ASCII spaces, `-` and `_` only. The same pure rule is
  applied in the edit UI and profil-service (`POST /profils`, admin create and `PATCH /profils/me`), so bypassing the
  browser cannot persist punctuation, `@`, `#` or emoji. Existing legacy names are not migrated or rejected when unchanged:
  users can still edit their bio/avatar and must choose a compliant value only when they actually rename themselves.
  Capturing the baseline today avoids a contournable cooldown later; the timestamp is free, only refusal is config-driven.

## Posts

- **Cross-service visibility barrier on ALL post reads** (not just the front filtering by followed ids): public →
  visible; private → owner or accepted follower only. Security must not depend on the front.
- **Denormalized `likes_count`/`comments_count` as `int32`** (`$inc`), idempotent likes via unique index.
  `int32` because the `$jsonSchema` validator declares `bsonType:"int"`.
- **Comment likes reuse the same domain pattern as post likes (17/06/2026).**
  A dedicated collection `comment_likes` stores the edges (`comment_id+user_id`
  unique) while `comments.likes_count` remains denormalized on the comment
  document itself (`int32`, `$inc`). Reads of comments/replies/profile-responses
  accept optional auth and hydrate a transient `liked` flag directly, avoiding
  a second “liked ids” endpoint just for comments. Counter refresh follows the
  same project-scale trade-off as posts: batch polling via
  `GET /posts/comments/stats?ids=...`, not WebSocket push. The new constrained
  field is backfilled idempotently at boot (`EnsureSchema`) to stay compatible
  with Mongo strict validation on legacy documents. Comment-like notifications
  are intentionally deferred: the requested value was the action + animation +
  auto-refresh, while a new notification type would widen the cross-service
  contract with notification-service and the front.
- **Pin:** `pinned_at` on the post, owner-only, one pin per profile; profile read sorts by it, but feeds return a
  copy **without** `pinned_at` so another's pin never personalizes the global feed (front exception `canPin` keeps
  instant visual feedback for the author).
- **Hashtags live on `posts` as normalized denormalized fields** (`hashtags: []string`, lowercase, no `#`) extracted
  on create/update. This keeps `GET /posts?hashtag=...` indexable in Mongo without a separate service or cross-DB
  search. Trends are counted in post-service after the same profile-visibility checks as the feed, so private profiles
  cannot leak through hashtag counters. The hashtag results view exposes `sort=top` (likes, reposts, comments, then
  recency) and `sort=recent`; the media tab reuses the same visibility-filtered post results and renders only their
  media attachments. Search routing is shared on the front: `#tag` opens hashtag results, while plain text remains
  account/profile search in Explorer. `/posts/trends?q=...` supports prefix suggestions for the hashtag search
  dropdown; the same visibility filtering applies before counting, so suggestions do not leak private authors.
  The right-column search stores only clicked suggestion metadata in localStorage per user (`breezy-suggestion-history`)
  because it is cosmetic UX state, not a domain datum worth a backend service.
  Composer hashtag autocomplete also reuses `/posts/trends?q=...` instead of adding a separate endpoint: suggestions
  are derived from already-visible trend data, while post-service remains the single extractor/normalizer on submit.
  Explorer's "posts with hashtags" uses `GET /posts?hashtag_any=true` rather than client-side filtering, preserving
  the server-side visibility barrier and avoiding downloading arbitrary global-feed pages just to discard non-hashtag posts.
- **Bookmarks = collections (many-to-many) + non-deletable default collection + burst model.** Bookmarks reference a
  post → same service as likes/reposts. Many-to-many because a post can be filed in several collections. **Burst model**
  (Instagram "save"): a short click within `BOOKMARK_SESSION_WINDOW` auto-files into the last collection (`filed`),
  otherwise opens a chooser (`needs_choice`, files nothing — server never guesses). Window state is server-side
  (`last_bookmark_at`, per account, multi-device, no clock cheat).
- **Polls are post-owned metadata + separate `poll_votes`.** A poll is part of the post document because it is authored,
  displayed, deleted and visibility-filtered with the post. Votes live in `poll_votes` with a unique `post_id+user_id`
  index so "one vote per account" is enforced by Mongo, not by the UI. Choice counters stay denormalized in the embedded
  poll as `int32` for fast feed rendering and validator consistency. The post-service owns duration expiry and the
  `everyone|followers` audience rule; followers-only polls reuse the existing follow client, so private/follower logic
  remains server-side and cannot be bypassed by a custom front. Results visibility is also server-side: before voting,
  ordinary viewers receive zeroed counters; after their unique vote, `voted_choice_id` makes `can_view_results=true` so
  they can see the current percentages without being able to vote again. The author always receives live counters, and
  everyone receives them after `ends_at` or manual `closed_at`. Manual close is author-only (`can_close`) and stores
  `closed_at` instead of rewriting the planned end date, preserving the original duration while making the poll final.

- **Reply audience is a post-owned setting enforced server-side (15/06/2026).** Like poll voting, *who can reply* is a
  per-post choice (`reply_audience` ∈ `everyone|followers`) made at creation and **not editable afterwards** (parity with
  `Poll.Audience`; `UpdatePostRequest` only carries `content`). The field is stored on the post document with
  `omitempty`, so an empty/absent value is never persisted and old posts (created before the field) read back as
  `everyone` via `ReplyAudienceOf` — the validator declares it as an **optional** enum property, so an absent value is
  already schema-valid and never crashes a legacy write. Even so, an **idempotent boot migration** `backfillReplyAudience`
  (in `EnsureSchema`) materializes `reply_audience="everyone"` on pre-field posts, by precaution on the prod DB and to
  honour règle 5b (the `visibility`/`likes_visibility` scars showed how a constrained field can lock legacy writes); it
  is a no-op once normalized (filter on absent/empty). The barrier is enforced in the single comment entry point `CreateComment` (covers
  both root comments and replies) by `canReplyTo`: a `followers` post lets the author, the author's followers, and
  moderators/admins reply, everyone else gets `ErrReplyNotAllowed` (403). Follower status reuses the existing follow
  client, exactly like followers-only polls, so the rule cannot be bypassed by a custom front. **Mods/admins bypass** the
  restriction because replying is not a moderation-blocked action and they already hold elevated rights. For UX the
  post-service hydrates a transient `can_reply` per viewer, but **only for `followers` posts** (the follow check is
  skipped entirely for the `everyone` majority → near-zero added cost on feeds); the front consults it solely to disable
  the composer and hide "Reply" buttons, never as the authority. A missing/`true` `can_reply` is treated as allowed so
  read paths that don't compute it (mutations) never wrongly lock the composer.

## Vue visiteur (fil public)

- **Mode visiteur = front-only, zéro backend.** Le post-service expose déjà la lecture publique (`GET /posts` et
  `GET /posts/:id` en `OptionalJWTAuth`, barrière de visibilité appliquée serveur → un appelant sans token ne voit
  que les posts publics). Décision : ne RIEN ajouter côté serveur, tout le mode visiteur vit dans le front.
- **Réutiliser le shell `(app)`, pas un groupe `(public)` séparé.** Le visiteur voit le MÊME fil avec la même
  sidebar, seulement amputée (carte user → boutons Se connecter/S'inscrire, nav réduite à Accueil). Un groupe de
  routes parallèle aurait dupliqué layout + feed pour un gain nul.
- **Le vrai risque n'était pas la garde mais la redirection forcée.** `apiFetch` redirige vers `/login` sur tout
  401 dont le refresh échoue. Or le fil enrichit chaque post via `/posts/me/{liked,reposted,bookmarked}-ids`
  (routes `auth`), et `useFollow`/`getMyProfil`/`getMe` tapent des routes `/me`. Sans token → 401 → redirection →
  le visiteur ne voit jamais le fil. **Parade : garder chaque appel `/me` derrière `getAccessToken()`** (early-return
  vide), même pattern que les providers Notifications/Messages. C'est la pièce centrale, pas le retrait du middleware.
- **`isVisitor` rendu « membre » par défaut au 1er paint.** Pas d'accès `localStorage` au SSR/hydratation → on suppose
  connecté puis on bascule au montage (même compromis que le thème / la session) pour éviter un flash de l'UI visiteur
  chez les membres.
- **Actions réservées → modale « Connecte-toi » (`requireAuth`/`promptLogin`), pas redirection ni masquage.** Choix
  produit : like/repost/citer/signet/commentaire invitent à s'inscrire sans quitter le fil. La LECTURE des commentaires
  reste libre (GET public) ; seul l'ENVOI est gaté. L'aperçu profil au survol est désactivé pour le visiteur (il
  chargeait le graphe social authentifié) et les profils restent gardés par le middleware.
- **Racine `/` → `/feed`** : `/feed` étant désormais public, c'est l'entrée naturelle du mode visiteur (membre = UI
  complète, visiteur = UI amputée).
- **Limite assumée** : garde UX côté front (le backend protège déjà les écritures par JWT ; un client pourrait lire
  les routes publiques directement, ce qui est précisément leur contrat).

## Translation

- **Translation via Next BFF route `/api/translate`** (LibreTranslate + Google fallback), keys server-side; post-service
  stores no derived translation. Conservative client gate (`shouldAttemptTranslation`): script detection + Latin markers
  avoid false positives (a lone foreign word, a typo, mixed text must not auto-translate a French post).
- **Coverage across the 12 locales:** upstream auto-detection is authoritative. The small local marker dictionaries are not
  an allow-list: any meaningful text not clearly identified as already being in the target locale reaches the BFF. The
  result is discarded when `detectedSourceLanguage === targetLanguage`. Local logic remains a cheap same-language veto;
  Han (Chinese) and Kana (Japanese) are distinct so ZH↔JA works. Target language always comes from the account locale.

## Messaging (E2EE)

- **Hybrid model.** Browser-side encryption. Each user has an X25519 keypair (private in IndexedDB, **per device**).
  Each conversation has a symmetric content key; for DM/groups it's wrapped per member (anonymous sealed box) → server
  sees only envelopes + `ciphertext`/`nonce` and can decrypt nothing = **admin-proof**. **Communities** keep the key
  **server-side** (handed to each joiner for unlimited auto-join) → semi-public, admin-readable. Strict E2EE with
  unlimited unknown viewers is impossible (needs an online key-holder) → "Telegram channel" style.
- **Groups:** owner + members, any member invites (holds the key), owner-only manage, cap 32, encrypted name, single key
  no rotation (new member reads full history; forward secrecy = future). **Communities:** viewer/talker, cap 32 talkers,
  unlimited viewers, **clear name** + public directory (name needed for discovery before joining).
- **Cursor pagination** (`before=<messageId>` on `_id`, not offset): stable on a live chat (new messages arrive via WS at
  the bottom without disturbing backward paging).
- **Schema materializes "no plaintext at rest":** `messages` have only `ciphertext`+`nonce` and, after edit,
  `original_ciphertext`+`original_nonce`; `user_keys` stores only the public key; `conversations.content_key` only for communities.
- **Message edit stays E2EE:** owner-only `PATCH /messages/conversations/:id/messages/:messageId` replaces the
  ciphertext, preserves the first encrypted version for subdued display, and broadcasts `message_updated` over WS.
- **Pin/clear are per-user, on the `members` collection** (not the shared conversation), no WS broadcast (personal).
  `cleared_at` = reversible cutoff (reappears on next message, WhatsApp-style) vs destructive delete.
- **"Read" state is server data** (`members.last_read_at`), multi-device; `GET /unread-count` computes from metadata only
  (never the `ciphertext`) → E2EE intact. **Mute** (`members.muted_at`) excludes from the badge but stays unread in the list.
  Chosen over per-device localStorage (tranché with the user).
- **Message read/delivery receipts ("Remis / Ouvert", 15/06/2026, validated with user).** Sender-side acknowledgements
  using **server metadata only** (timestamps, never the `ciphertext` → E2EE intact). Two per-member cursors: existing
  `last_read_at` (= *Ouvert*/opened) + new `last_delivered_at` (= *Remis*/delivered). **Delivery is server-driven (no client
  ack):** a message is *Remis* when the server actually delivers it — pushed to a member's open WS socket at send time
  (`hub.OnlineFrom`) or fetched via `ListMessages` (`TouchDelivered`). Offline recipient ⇒ not yet delivered (correct).
  Reading implies receiving, so `SetMemberRead` also advances delivery (remis ≥ ouvert). A new WS event `receipt`
  `{conversation_id, user_id, delivered_at?, read_at?}` is broadcast to the **other** members (the senders) on read/delivery
  so the checks update live; `ConversationView.member_receipts` exposes the other members' cursors for the initial render
  (**DM + groups only**, not communities — too many members, not meaningful). **UI convention (chosen by user, refined after
  iPhone testing):** a **single marker, always under my LAST message** (the check follows the latest sent message): 1 grey
  check = "Envoyé" (sent, not yet read) ; 2 brand-coloured checks = "Ouvert" (read by peer / by all in a group) ; group
  partially read = 1 coloured check + reader count. The label "Envoyé"/"Ouvert" is revealed by tapping the check (PC + mobile).
  (An earlier two-marker DM design — last-read vs last-delivered on different messages — was dropped: in practice the check
  appeared "stuck" on an older message.) Placement logic is a pure tested function (`planReceipts`); live updates via pure
  `applyReceipt` (monotonic). New field `last_delivered_at` is **nullable/optional** (no enum, no `required`) ⇒ no boot
  migration needed (règle 5b). The `last_delivered_at` cursor is still maintained server-side (harmless) though the simplified
  UI keys off the read cursor only.
- **Message deletion = tombstone "for everyone" (15/06/2026, validated with user).** `DELETE …/messages/:messageId`
  soft-deletes: sets `deleted_at` and **wipes** `ciphertext`/`nonce` (content really gone — the server was already blind, so
  this is honest "delete for everyone"). The client renders "Message supprimé" in place. **Authorization** (pure tested
  `canDeleteMessage`): the **author** always; in a **group/community**, the **owner/admin** can also delete others' messages
  (moderation); **no moderation in DM** (no hierarchy). Broadcast reuses the existing **`message_updated`** WS event (the
  tombstone replaces the bubble) — no new event type. `deleted_at` is nullable/optional ⇒ no migration. Encrypted media blobs
  in media-service are left orphaned (acceptable; cleanup is a TODO).
- **Typing indicator "en train d'écrire" (15/06/2026, validated with user).** Ephemeral, **not persisted**, E2EE-safe
  (metadata only: `user_id` + `conversation_id`, never content). Transport = **REST** `POST …/typing` (chosen over making the
  WS bidirectional): the client pings **throttled ~3 s** while typing; the server broadcasts a `typing` WS event to the other
  members. No "stop" signal — the receiver sets a **+6 s expiry** per member, pruned client-side, so the indicator fades on
  its own. Shows "écrit…" (DM) / "X écrit…" / "Plusieurs personnes écrivent…" (group).
- **Passphrase key backup (multi-device), zero-knowledge** (11/06/2026, validated with user). The X25519 private key was
  per-device only → unreadable on a 2nd device. Now the private key is wrapped client-side (`XChaCha20-Poly1305`) under a
  KEK derived from a **user passphrase** (Argon2id, **19 MiB / t=2 = OWASP minimum**, chosen for mobile: the pure-JS KDF is
  synchronous and 64 MiB/t=3 froze the main thread ~15-25 s on phones → "infinite spinner"; an **auto-upgrade** re-wraps any
  heavier legacy backup with the light params on the next unlock). The KDF runs in a **Web Worker** (`key-backup.worker.ts`,
  driven by `key-backup-async.ts`, sync fallback) so the UI stays responsive during derivation. Stored server-side in
  `key_backups` as an opaque blob
  (`salt`, `nonce`, `wrapped_private_key`, `kdf_params`, `public_key`). Server never sees the passphrase nor the private key
  → **admin-proof preserved**. The IndexedDB identity is now **scoped per `userId`** (was a single global `self` record
  shared by every account on the browser → wrong key reused across accounts); a one-shot migration adopts the legacy `self`
  key for an account **only if** its public key matches the one that account already published. State machine: `ready`
  (local key + backup) / `unlock` (backup but no local key → other device: download blob, Argon2id-derive, unwrap; wrong
  passphrase ⇒ Poly1305 auth fails) / **`setup`** (no backup → define passphrase; reuses the local key if present so
  **existing users back up their current key**, else generates one). UI: `PassphraseGate` blurs the messages view until the
  identity is available. Routes declared **before**
  `/keys/:userId` (else `backup` is captured as a userId). **Assumed limits:** forgotten passphrase = unrecoverable backup
  (the point of zero-knowledge); **admin reset deferred** — the only crypto-honest semantics is "wipe backup → new identity"
  (escrow would break DM admin-proof, ruled out); a device with no local key and no backup generates a fresh key (prior
  history stays unreadable there, same as before).

## Notifications

- **Emission = synchronous best-effort fire-and-forget** to `/internal/events` (off the gateway, `INTERNAL_EVENT_SECRET`).
  Temporal coupling neutralized by best-effort; an event bus (NATS/Redis) is oversized for now (report perspective).
- **Aggregation (anti-spam):** one notification = one `group_key` per recipient (`like:<post>`, `comment:<post>`,
  `reply:<rootComment>`, `mention:<source>`, `repost:<post>`, `quote:<...>`, `follow`,
  `follow_request:<actor>`, `follow_request_accepted:<actor>`,
  `follow_request_accept_confirm:<actor>`,
  `message`, `message_mention:<conv>`). Most types use `$inc count` and `retract` decrements/deletes. `message`
  is the exception: optional `actor_ids` stores unique senders and `count = len(actor_ids)`, so several messages
  from one person keep a single notification while messages from different people render "X and N others".
- **Rules by type:** like/comment/repost/quote → post (or quoted) author; reply → ROOT author; mention → each mentioned;
  private DM/group message → all other members, globally grouped per recipient and never emitted for communities;
  follow_request → private-profile owner (actionable); accepted request → persisted notification + WS decision
  to the requester, plus persisted confirmation in the owner's notification list; rejected request → no
  requester notification. Never to self.
- **Front badge:** app-wide unread counter held in memory (dedup by id), single WS, seeded by `unread-count`. Avoids a
  server request per event during spikes; self-corrects on page open / `notification_refresh`.

## Fil temps réel (WebSocket) — bandeau « a posté » (15/06/2026)

- **Goal:** X-style live feed — when someone else posts while you're reading, a floating banner (avatar + "a posté")
  appears at the top; clicking it reveals the new posts and scrolls up. (Validated scope: banner only, no live counter
  updates.)
- **Broadcast hub, not per-user.** post-service gets its own `realtime.Hub`, but unlike notification/message hubs
  (one connection indexed per recipient) the feed is public → the hub keeps a flat set of connections and broadcasts to
  all. Endpoint `GET /posts/ws?access_token=<jwt>` (token in query param: browsers can't set Authorization on a WS
  handshake). The gateway already proxies the WS upgrade under `/posts` (httputil reverse proxy) — no gateway change.
- **Ping, not content ("firehose + refetch").** The WS event carries only `{type, post_id, author_id}`. The front
  resolves the author (avatar/name) from its existing `resolveAuthor` cache and fetches the actual content through the
  normal authenticated feed endpoint. **Rationale:** keeps the WS lightweight AND keeps the visibility barrier
  server-side (security never depends on the front) — no need to duplicate author-enrichment / visibility logic in the
  hub. A click in the "Following" tab on a non-followed author simply returns nothing on refetch.
- **Public-account gating at emission.** `broadcastNewPost` only pings when the author's account is public
  (`profilClient.Visibility`); a private account's post must not signal activity to non-followers. Fire-and-forget in a
  goroutine, never blocks/fails post creation (same contract as the notifier). Hub injected via `WithFeedBroadcaster`
  (no-op default → service stays unit-testable without a hub).
- **Client relevance filter.** `FeedView` drops self-authored pings (already prepended locally), pings while a hashtag
  filter is active, and posts already shown; the "Following" tab additionally filters by the cached `getFollowingIds`
  set. Banner = latest pinged author's avatar; reveal refetches page 0 of the active tab and prepends the dedup'd new
  ones.

## Compteurs dynamiques (polling batch) — likes/commentaires/reposts (15/06/2026)

- **Goal:** the counters on displayed posts (likes/comments/reposts) refresh on their own "in a few seconds", X-style —
  deliberately NOT to the millisecond like the "a posté" banner.
- **Polling batch over WebSocket push — justified by scale.** We do NOT reuse the realtime hub for counter deltas. The
  feed hub is a flat broadcast to *all* connections; pushing every like/comment of every public post to everyone is high
  volume and would need server-side coalescing to be viable — the exact "millisecond" behavior we don't want. Instead the
  front polls **one** lightweight request `GET /posts/stats?ids=…` for the whole displayed page every ~7 s. X avoids
  polling because at 500M users even batched polling is huge; at our scale a projected `$in` over ~20 ids every 7 s is
  negligible. We keep 2 of X's 3 ideas (lazy = only displayed posts; batched) and skip deltas (absolute values are simpler
  and self-healing if a cycle is skipped). Defense angle: a choice *justified by scale*, not cargo-culted from X.
- **Same visibility barrier, reused not duplicated.** `PostStats` runs each post's author through the same
  `canReadAuthor` as the feed (memoized per author); invisible posts (private-unfollowed, moderation-hidden, deleted,
  invalid id) are simply absent from the response — no error, the front only patches what it already knows. Repo
  `StatsByIDs` is a single projected `$in` (counters + `author_id`, no content/media read). Bounded at `MaxStatsIDs=100`.
- **Never overwrite the local "me" state.** `applyStatsToPost`/`applyStatsToPosts` touch only the 3 counters; `liked`/
  `reposted`/`bookmarked` stay driven by the user's own optimistic actions. Both helpers preserve the array/object
  reference when nothing changed → a quiet polling cycle triggers zero re-render. Reusable hook `usePostStatsPolling`
  (`STATS_POLL_INTERVAL_MS=7000`) only polls while `!document.hidden`, refetches immediately on tab refocus, and guards
  against overlapping in-flight requests. Wired in `FeedView`, `PostDetail`, `ProfilView`.

## Mentions (@handle)

- **Shared pure bricks** (`lib/mentions.ts`, regex aligned with the back) reused across posts/comments/messages = one
  source of truth for regex/insertion/rendering.
- **Posts/comments:** back already emits `mention`; front adds autocomplete + clickable render — no back change.
- **Messages (E2EE):** server can't read `ciphertext`, so the **client** computes `mentioned_member_ids` (ids only, never
  text) → message-service emits `message_mention` (aggregated per conversation). Render: member → profile,
  non-member → preview card → `/explorer`. "X mentioned you" preview is 100% client (last message already decrypted) →
  no server flag, E2EE preserved.

## Media (MinIO via gateway)

- **6th microservice `media-service` + MinIO**, content-agnostic (opaque bytes under a random id, `owner_id` in MinIO
  user-metadata → no side DB). **Download through the gateway, NOT presigned MinIO URLs** — the cardinal rule is "everything
  through the gateway"; presigned URLs would expose MinIO (public port, CORS, host unresolvable by the browser) and violate
  the principle defended at the oral. The stdlib proxy already streams → video is not a memory concern. MinIO = canonical
  microservices answer (ticks "containerization"), far better at defense than "blob in DB"/disk.
- **Clear upload `POST /media`** (JWT, magic-byte MIME sniff, caps **5MB image / 5MB video** for non-admins) vs **`POST /media/encrypted`**
  (opaque E2EE blob, no sniff, own **5MB** cap `MEDIA_MAX_BLOB_BYTES`). `GET /media/:id` is **public** (unguessable id), streamed with Range/seek + immutable cache + ETag.
- **Cap d'upload média (16/06/2026) :** plafond **uniforme 5 Mo** pour tous les types (image / vidéo / blob chiffré), aligné sur le
  plus petit cap d'un service de référence (X.com photo = 5 Mo) → rien à descendre sous 5 Mo. **Administrateurs non plafonnés :**
  `Upload`/`UploadEncrypted` bypassent tous les checks de taille quand `claims.Role == "admin"` (réutilise le claim déjà lu pour
  `Delete`/purge RGPD). Choix « pas de cap admin » plutôt qu'un cap admin élevé configurable : un admin de confiance n'a pas besoin
  d'un nombre arbitraire, et le streaming MinIO (buffer `MaxMultipartMemory`, reste sur disque temp) encaisse les gros fichiers.
  Le défaut vidéo a été abaissé **50→5 Mo** (avant : asymétrie image/vidéo non justifiée par la demande). **La limite est appliquée
  côté serveur** (§6) ; le front ne fait qu'une garde UX (`exceedsMediaLimit`, admin-aware) qui écarte les fichiers trop lourds **à la
  sélection** pour un retour immédiat, sans jamais être l'autorité.
- **Posts (Phase 2):** post-service carries `media []MediaRef`, stays agnostic (relative `/media/<id>` refs); `content`
  optional if ≥1 media. **Messages (Phase 3):** file encrypted client-side → opaque blob uploaded → id/nonce/mime in the
  **encrypted envelope** (`{v:1,text,media[]}`, else plain text → backward-compatible) → server blind, zero schema change.

## Media viewer

- **Two behaviors.** Private message → `MediaLightbox` (the `src` is the already-decrypted objectURL → download stays E2EE).
  Post → `PostPhotoModal` two-pane X-style (media + comments); `PostActions` extracted from `PostCard` so card and modal
  share one optimistic logic. **Both overlays rendered via `createPortal(document.body)`** because a `backdrop-blur`/`transform`
  ancestor creates a containing block for `position:fixed`, confining the overlay. Assumed: card and modal hold distinct state
  (a like in the modal isn't live-reflected on the card behind; resync on next load).

## Feed video (Twitter-style autoplay)

- **`FeedVideo`:** muted+loop+playsInline autoplay when ≥60% visible (IntersectionObserver), pause off-screen.
  **Lazy-mount** via a second observer (`rootMargin:400px`) so a video far down an infinite feed is neither downloaded nor
  looped — the key to not loading N videos at once. Front-only, zero server cost. Autoplay mandates muted (browser policy).

## GIFs (GIPHY)

- **Proxy GIPHY porté par media-service via la gateway (`/gifs/search`)**. La clé `GIPHY_API_KEY` reste côté serveur,
  jamais dans le navigateur. On rattache ce proxy au media-service plutôt qu'à une route BFF Next pour garder la doc Swagger
  et le routage API au même endroit que les médias, sans ajouter de microservice ni de base.
- **Posts = URL externe stockée comme média image.** Le post-service reste agnostique du contenu : il persiste déjà
  `media[] {url,type}` et accepte les URLs `https:`. Aucun champ Mongo, aucun backfill. Le composer ajoute le GIF choisi
  comme `{url,type:'image'}` et réutilise les previews/rendus média existants.
- **Messages E2EE non couverts par cette phase.** Si les GIFs sont ajoutés aux messages plus tard, il faudra éviter de
  faire charger une URL tierce en clair par le destinataire ; l'option prévue reste de rapatrier les octets puis de passer
  par le pipeline de pièce jointe chiffrée.

## i18n / theming / responsive

- **i18n = home-grown, zero dependency** (`lib/i18n.ts` + dictionnaires statiques + `LanguageProvider` + `useT()`). 12 langues
  sont disponibles : FR/EN/ZH/ES/PT/RU/JA/KO/AR/HI/DE/IT ; FR reste la référence et le repli final. Les pages légales restent
  officiellement rédigées en FR/EN et retombent sur EN pour les autres locales afin de ne pas présenter une traduction
  automatique comme juridiquement fiable.
- **Préférence de langue = donnée de compte dans user-service**, pas une clé navigateur globale. `users.preferred_locale` est
  nullable et contraint aux locales supportées ; NULL signifie « utiliser `navigator.language` ». `GET/PATCH /users/me`
  transporte la préférence sans nouvelle route. Le logout ne l'efface pas : une reconnexion au même compte la restaure ; un
  autre compte charge sa propre préférence ou la langue du navigateur. Le localStorage ne sert qu'aux visiteurs anonymes.
- **Theme:** light/dark (next-themes) + brand accent. Brand gradient violet→indigo→cyan (logo "B"), glassmorphism.
  **Tokenized surfaces** (CSS vars in `globals.css`, light values = exact current state, dark declension) → dark lives in one
  place, light unchanged. ⚠️ a `bg-*`/`border-*` utility overrides `@layer components` classes.
- **Custom theme (user-chosen colors)** = a 3rd dimension over light/dark + accent. 3 targets, each independent:
  **background** repaints the whole reading space — `--background` + `--bg-page` (solid) + glow layers→`none` **and** the glass
  surfaces `--column`/`--glass`/`--glass-strong` (background-color → hex) + `--panel`/`--panel-y` (background-**image** → must be a
  `linear-gradient(hex,hex)`, a raw hex is invalid there); **text** → `--foreground`/`--card-foreground`/`--popover-foreground`;
  **primary** → `--primary`/`--ring` + auto-contrasted `--primary-foreground` (WCAG luminance) **and the brand-gradient action
  buttons**. The "Breeze"/"Follow"/FAB/badges were a hardcoded `from-[#8D3DFF] via-[#5B6CFF] to-[#47D9FF]` (≠ `--primary`, hence
  invisible to the picker); they now read `from-[var(--brand-from)] via-[var(--brand-via)] to-[var(--brand-to)]` (defaults in
  `:root`), and the primary target collapses the 3 stops onto the chosen solid color. Overrides set as inline styles on
  `<html>`, so they win over the stylesheet regardless of mode. **Zero dependency**: home-grown HSV color wheel (`ColorWheel`,
  conic-gradient hue + radius saturation + value slider + hex/RGB) — `node_modules` is root-owned so `npm i react-colorful` is
  out, and an in-repo wheel is defense-friendlier. Color math is pure/tested (`lib/color.ts`); apply/persist logic pure where it
  matters (`buildCustomThemeVars` in `lib/custom-theme.ts`). **Persist both** source hexes (to reopen the editor) **and the
  pre-computed CSS-var map** in localStorage → a tiny inline `<head>` script applies the map before first paint (no FOUC, no color
  math shipped in the blocking script). `enabled:false` keeps the chosen colors but makes the inline script skip them. UX keeps
  the light/dark switch and exposes custom as a separate appearance row: the label opens the existing `Dialog`, the small switch
  toggles `enabled` without deleting colors, and the light/dark switch disables custom (`enabled:false`) while switching to the
  selected Breezy base. The dialog is held as a **sibling** of the dropdown/Sheet (not inside), opened on a deferred
  `setTimeout(0)` to dodge the Radix close↔open focus race.
- **Responsive mobile-first, pivot `lg` (1024).** <lg: `MobileHeader` (left drawer Sheet) + bottom `MobileTabBar` + `ComposeFab`;
  ≥lg sidebar; ≥xl right column. Manual edge-swipe to open the drawer (Radix Sheet has no native swipe).
- **Identity is clickable → profile everywhere** (`UserListItem` stretched link; the Follow button is raised `z-10`).
  Convention posed now so DM/notifications respect it (profile photo = minimal guaranteed anchor).
- **Avatar fallback uses the first actual Unicode letter, globally.** Shared `initialOf(displayName, username)` skips
  spaces, digits, separators and legacy punctuation, then falls back to the first letter of the username and finally `?`.
  Feed, profile, navigation, search, moderation, notifications and messaging all use this helper, avoiding `_`/`-`/digits
  as pseudo-initials. In profile editing, clearing avatar/banner stores an empty media reference and restores the existing
  generated initial avatar or gradient banner; the old MinIO object is deliberately not deleted, matching media replacement.
  The shared Avatar wrapper keys its Radix root by image identity because Radix otherwise retains `loaded` after a conditional
  `AvatarImage` unmount and keeps the fallback hidden when an avatar URL is cleared.

## CI/CD

- **Smoke-test externe post-déploiement (`deploy.yml`, job `smoke-test`).** Tourne sur un runner GitHub (vue externe, pas depuis le VPS) après le job `deploy`. Vérifie que `API_PUBLIC_URL/health` et `FRONT_PUBLIC_URL` répondent 200 en HTTPS — ce qui aurait attrapé le bug Cloudflare (handshake TLS KO côté edge alors que les conteneurs étaient sains). 3 tentatives espacées de 10 s pour absorber le redémarrage des conteneurs ; URLs dans des variables GitHub Actions (`vars.API_PUBLIC_URL` / `vars.FRONT_PUBLIC_URL`, non-sensibles → `vars.*` et non `secrets.*`).

- **3 GitHub Actions workflows by domain:** `ci-go` (per-service matrix: gofmt + vet + build + `test -race` + tidy +
  golangci-lint v2 + govulncheck report-only), `ci-frontend` (lint + build/typecheck), `ci-integration` (docker compose up the
  stack minus the frontend, wait for all healthchecks). Separate files = independent triggers/path-filters; matrix = parallelism +
  per-module isolation; `-race` is free and catches data races; the docker smoke test materializes "DB health" + "full Docker".
  `.env` gitignored → regenerated from `.example` in CI. govulncheck = 0 vuln since Go 1.25 bump.

## Dev / infra

- **`make dev` = `docker-compose.dev.yml` overlay** (Go: `air` hot-reload + bind-mount + shared Go caches; frontend: `next dev`
  + bind-mount + `WATCHPACK_POLLING`). The `dev` Dockerfile stage sits before the runtime stage so `make up`/`build` always
  produce the prod image (last stage). `make up` runs **frozen prod images** (no rebuild on source change).
- **`.env` layering:** root = cross-cutting vars (`JWT_SECRET`, `INTERNAL_EVENT_SECRET`, `NEXT_PUBLIC_API_URL`, only the root
  `.env` is interpolable by compose); `<service>/.env` = own config via `env_file:`; `environment:` only for Docker overrides.
- **Containerization = Docker + docker-compose** (grading requirement).

## Visibilité des likes (`likes_visibility`)

- **`likes_visibility` vit dans profil-service** aux côtés de `visibility`, sur le même modèle (`public|private`, défaut `public`). Posé sur le `Profil` Mongo, exposé via `GET /profils/:id/likes-visibility` et modifiable par `PATCH /profils/me`.
- **Application côté serveur dans post-service** : `GET /posts/liked?author_id=<id>` vérifie `LikesVisibility` avant de renvoyer la liste ; si privé et caller ≠ owner → 403. Le front affiche un état vide explicite (`LikesPrivateTab`) sur 403.
- **Barrière de visibilité secondaire** : les posts hiddens (`is_hidden=true`) sont exclus directement en base (`$ne: true` dans la query Mongo). La visibilité par auteur (profil privé + non-abonné) n'est pas recheckée post par post pour éviter N appels à profil-service (acceptable à l'échelle du projet).
- **Pas de changement sur les routes existantes** (`/posts/me/liked-ids`, `POST /like`, etc.) — la préférence ne change que la visibilité de la liste publique de likes.

## Changement d'e-mail : remise stricte, autres mails best-effort (15/06/2026)

- `mail-service` ne confond plus le rendu console avec une remise : sans SMTP configuré, `Mailer.Send` retourne `ErrDeliveryUnavailable` et `/internal/send` répond 503 `delivery_unavailable`.
- Le choix de fiabilité appartient au cas métier. Inscription, reset et création admin gardent leur comportement best-effort existant ; un changement d'adresse est strict, car annoncer un lien inexistant laisserait l'utilisateur sans moyen de terminer l'opération.
- Après une erreur de remise, auth supprime dans une transaction uniquement le jeton `email_change` exact et efface `pending_email` seulement s'il correspond encore à la demande. Ces prédicats protègent une éventuelle demande concurrente plus récente.
- L'API publique répond 503 `email_delivery_failed`; le front n'affiche le succès qu'après acceptation réelle par SMTP. Les secrets SMTP restent exclusivement dans `mail-service/.env`.

## Feed en arrière-plan persistant (16/06/2026)

- **Objectif** : garder le fil monté en fond façon X pendant qu'on consulte une autre section (scroll/posts/WebSocket préservés, retour instantané).
- **Approche abandonnée — `@modal` (parallel + intercepting routes)** : un slot parallèle `@modal` + 11 intercepting routes `(.)` affichaient la section ciblée en overlay tandis que `children` « gardait » le feed. Fragile par conception : `children` est un slot unique qui contient la **page réelle**, pas « toujours le feed » ; l'illusion ne tenait que sur une nav soft *partie du feed*. D'où 4 bugs cumulés — 404 intermittent (résolution du slot), feed défilable derrière (pas de scroll-lock), blocage après refresh d'une sous-page (interception perdue → impossible de revenir au feed), admin/modération non couverts.
- **Décision — feed au niveau du layout** : `FeedView` est monté **une seule fois** dans `(app)/layout.tsx` (colonne centrale, porte le scroll fenêtre). `(app)/feed/page.tsx` rend `null` (le fond transparaît). Les autres pages s'enveloppent dans `<FeedOverlay>` (`fixed inset-0 z-40`, `messages` en `wide`) et sont **rendues au niveau racine** du layout, hors de la colonne centrale : le `backdrop-blur` de cette colonne crée un bloc conteneur qui confinerait un `position:fixed` (même contrainte que les modales rendues via `createPortal`, cf. « Media viewer »).
- **Scroll-lock** : `OverlayScrollLock` pose `overflow:hidden` sur `<html>` dès que `pathname ≠ /feed` → le feed se fige à sa position (préservée, `overflow:hidden` ne reset pas `scrollTop`) et ne défile plus derrière l'overlay.
- **Feed gelé hors `/feed`** : comme `FeedView` reste monté, il **gèle** sa lecture des query params (`hashtag`/`tab`) via des refs quand on n'est pas sur `/feed`, pour ne pas refetcher / réinitialiser le contexte hashtag en arrière-plan quand un overlay s'ouvre.
- **Conséquence** : refresh et deep-link suivent le **routing normal** (plus de distinction soft/hard) → les bugs 404 et refresh disparaissent par construction, admin/modération deviennent des overlays comme les autres. Suppression complète du dossier `@modal` et de `overlay-when-path.tsx`. La nav dure post-auth (`window.location.assign(/feed)` dans login/verify-email/callback) est conservée par choix « état d'app propre », non plus comme parade au 404.
- **Coût assumé** : `FeedView` se monte sur tout deep-link `(app)` (même `/parametres`) → un fetch feed en fond, invisible. Prix du retour instantané au feed.
