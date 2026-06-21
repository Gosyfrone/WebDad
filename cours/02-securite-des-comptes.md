# 02 — Sécurité des comptes

> Features : double authentification MFA (TOTP) · vérification d'e-mail · mot de passe oublié (reset) ·
> changement d'e-mail vérifié · mail-service · durcissement post-pentest (rate-limiting, gate `email_verified`, en-têtes).

## Vue d'ensemble

Au-delà du JWT (fiche 01), cette fiche couvre tout ce qui protège le **compte** : preuve de possession de la boîte
mail (vérif / reset / changement d'adresse), **second facteur TOTP**, et le **durcissement** issu d'un pentest. Fil
conducteur : la **logique-token reste dans auth-service (PostgreSQL)**, seul le **transport SMTP** est externalisé.

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **auth** (PG) | génère/valide les tokens opaques (vérif, reset, email_change, mfa_challenge), secret MFA chiffré, rate-limiting | propriétaire des credentials et de la signature |
| **mail** (8089, hors gateway) | transport SMTP, templates HTML de marque | isole les secrets SMTP ; symétrique de notification-service (2ᵉ consommateur d'événements internes) |
| **Caddy (reverse-proxy edge)** | en-têtes de sécurité (HSTS, CSP, nosniff…) | défense périmétrique au plus près du client |
| **Frontend** | pages `/verify-email`, `/forgot-password`, `/reset-password`, MFA dans `/parametres`, écran de challenge | maîtrise de l'UX (succès/expiré/erreur) |

## Comment c'est codé

### Tokens e-mail (vérif / reset / changement)
- Table unique `account_tokens(purpose enum 'verify'|'reset'|'email_change'|'mfa_challenge', …)`, **opaques aléatoires, stockés hachés SHA-256, usage unique** (`used_at`), TTL **vérif 24h / reset 1h / mfa_challenge 5 min**. Réutilise le pattern refresh-token : révocables, fuite de table = aucun token utilisable.
- Toute nouvelle demande **invalide les précédentes** du même `(user_id, purpose)`.
- Les **liens du mail pointent vers des pages front** (`/verify-email|reset-password?token=…`), pas l'API → maîtrise UX, API JSON-only.

### Vérification d'e-mail
- Register → mail de vérif (best-effort) + page « consulte ta boîte mail ». **Blocage dur du login non-vérifié** : `403 email_not_verified`, vérifié **après** le bcrypt (ne révèle pas l'existence du compte). Renvoi anti-énumération.
- `verify-email/confirm` = **auto-login** : consomme le token, pose `email_verified=true` **et** émet une session (comme `/login`). Justification : cliquer le lien (usage unique, 24h, envoyé à l'adresse du compte) **prouve la possession de la boîte** → facteur d'auth suffisant. Garde : un compte désactivé reste bloqué.

### Mot de passe oublié (reset)
- `forgot-password` = **anti-énumération stricte** : renvoie **toujours** `200` générique ; le mail (lien 1h) n'est envoyé que pour un compte existant ET actif.
- `reset` : token 1h usage unique → bcrypt + `email_verified=true` (preuve de possession) + **révocation de toutes les sessions** (`DELETE refresh_tokens` — un changement de mot de passe doit déconnecter partout). Rejouer le lien → `400 invalid_token`.

### Changement d'e-mail (deux temps, sans couper l'accès)
- `POST /auth/email/change/request` (JWT) → met la nouvelle adresse dans `pending_email` (index unique partiel) + envoie un jeton 24h. **L'adresse courante reste active** tant que le lien n'est pas cliqué → une faute de frappe ne verrouille jamais le compte.
- Confirmation publique = **transaction** : `FOR UPDATE`, promotion de `pending_email`, consommation du jeton, révocation des refresh + anciens jetons, puis nouvelle session (JWT avec la nouvelle adresse).
- **Remise stricte** : sans SMTP configuré, le mailer renvoie `ErrDeliveryUnavailable` → l'API répond `503` et annule le jeton exact + `pending_email` (prédicats qui protègent une demande concurrente). Annoncer un lien inexistant laisserait l'utilisateur bloqué.

### MFA (double authentification TOTP)
- **TOTP seul, opt-in, sans codes de secours** (RFC 6238 via `pquerna/otp`, compatible Google/Microsoft Authenticator). Un compte sans MFA garde le login inchangé.
- **Secret chiffré at-rest AES-256-GCM**, clé `MFA_ENCRYPTION_KEY` **hors-DB** (base64 32 o, `.env` racine). Fuite de table = aucun secret sans la clé. **Pattern « nil = MFA off »** : clé absente → endpoints `/auth/mfa/*` en 503, auth reste bootable.
- **QR généré côté serveur** (data URI) : `node_modules` du front est root-owned (interdiction d'ajouter une dépendance npm) → `key.Image()` rend le PNG, le front pose juste un `<img>`.
- **Login en deux temps** : si `mfa_enabled`, le login valide le mot de passe (et `email_verified`) mais **n'émet aucun JWT** → crée un jeton `mfa_challenge` (5 min) et renvoie `{mfa_required, challenge}`. `POST /auth/mfa/verify` (**public** : le challenge prouve que le mot de passe est passé) valide le TOTP **avant** de consommer le challenge → **un code faux ne brûle pas le challenge**.
- Activation confirmée par un premier TOTP ; désactivation par TOTP **ou** mot de passe. Skew ±1 (dérive d'horloge).
- Routes : `POST /auth/mfa/{setup,enable,disable,verify}` + `GET /auth/mfa/status`.

### mail-service
- Module Go autonome (8089), `POST /internal/send` **hors gateway**, authentifié par `MAIL_INTERNAL_SECRET` (`X-Internal-Secret`). Transport SMTP (`net/smtp`) ou **console en dev** (logge le mail + lien sur stdout → zéro dépendance Gmail en dev).
- **Templates HTML de marque** partagés (`brandedEmailHTML`, pur & testé) : layout table + styles inline (compat Outlook/Gmail/Apple Mail), bouton « bulletproof ». Vérif et reset le réutilisent.

### Durcissement post-pentest
- **Rate-limiting applicatif par IP** (`auth-service/internal/middleware/ratelimit.go`, in-memory, sans dépendance) : login + mfa/verify = **10/5 min** ; register + forgot + resend = **5/15 min**.
- **Anti brute-force MFA** : challenge **brûlé après 5 codes faux** (sinon 10⁶ codes brute-forçables dans la fenêtre 5 min, même multi-IP).
- **`email_verified` comme claim JWT + gate `VerifiedOnly`** : propagé sur tous les chemins d'émission ; un middleware garde la **création de contenu** (posts, commentaires, conversations, messages).
- **En-têtes de sécurité au reverse-proxy (Caddy)** : HSTS, X-Frame-Options:DENY, nosniff, Referrer-Policy, Permissions-Policy ; **CSP en Report-Only d'abord** (le script de thème inline serait bloqué par `script-src 'self'` → on observe avant de poser un nonce/hash).

## Décisions clés (et alternatives écartées)

- **mail-service séparé, logique-token dans auth** : isole les secrets SMTP et garde la logique métier là où vivent les credentials. *Écarté* : tout mettre dans auth (couplerait SMTP et auth).
- **Tokens opaques hachés usage-unique** plutôt que JWT pour vérif/reset : révocables, fuite sans danger.
- **Vérif & reset valent auto-login** : cliquer le lien prouve la possession de la boîte → on révise la position « pas de session avant login » (qui ne vaut plus que pour le register).
- **Provisioning au register sans session client** : le BFF utilise les tokens **uniquement côté serveur** pour provisionner user+profil puis les jette. *Écarté* : « register sans token + provisioning au 1ᵉʳ login » imposerait de charrier `birth_date`/`gender` jusqu'au login (plus lourd/fragile).
- **MFA sans codes de secours** (tranché avec l'utilisateur) : périmètre réduit pour livrer/tester vite. *Limite assumée* : perte du téléphone = lockout.
- **MFA = login mot de passe uniquement** : les comptes OAuth ont le 2FA de leur provider, ne passent pas par `/auth/login`.
- **Rate-limiting applicatif** plutôt que Fail2Ban : portable, indépendant de l'hôte, testable, conscient du « par compte » à terme. *In-memory* suffit (une instance par service ; Redis si multi-instances).
- **Remise stricte du changement d'e-mail, best-effort ailleurs** : le choix de fiabilité appartient au cas métier.

## Points de défense / Q&A anticipées

- *« Pourquoi vérifier le mot de passe AVANT de dire « non vérifié » ? »* → sinon on révèle qu'un compte existe (énumération).
- *« Comment évite-t-on de brute-forcer un TOTP ? »* → rate-limit IP **+** challenge brûlé au 5ᵉ échec (couvre l'attaquant multi-IP).
- *« Où est la clé MFA ? »* → hors-DB dans `.env` racine ; secret stocké chiffré AES-256-GCM en base.

## Limites assumées / perspectives

- MFA : **codes de secours** / reset admin (évolution).
- Mail Phase 3 : rate-limiting d'envoi, e-mails EN.
- CSP : basculer de Report-Only en **enforce** (nonce/hash sur le script de thème inline).
