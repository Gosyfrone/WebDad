# Scénario de démonstration — Soutenance Breezy

> **Présentateurs :** Alex · Maxime · Philippe · Romain — *(assignez-vous les blocs ci-dessous).*
> **Environnement :** prod `https://breezy.philippeluu.fr` — vérifiée opérationnelle (front + API gateway + données seedées).
> **Légende :** `[CLIC]` = action à l'écran · 🗣️ = ce qu'on dit · 🛡️ = argument de défense / choix technique · 🔊 = un son est joué (laisser entendre).
> **Conseil de rythme :** une personne parle, une autre conduit la souris. Comptes de démo prêts : un user principal (`@…`), `@zervixt` (privé / activité masquée), un 2ᵉ device/onglet pour les events live.

> **⏱️ Avant la démo :** Maxime & Philippe présentent ~5 min un **PowerPoint technico-commercial** de Breezy → ils ont donc la part la plus légère de nouvelles tâches dans la démo.

---

## Répartition des présentateurs

> Chaque étape ci-dessous est taggée **[Alex] / [Maxime] / [Philippe] / [Romain]**. Synthèse :

| Présentateur | Bloc principal (démo) | Nouveautés ajoutées |
|---|---|---|
| **Romain** | Auth · Profil (3 onglets, édition, abonnés, nationalité/sexe/localisation, bannière/PP GIF) · **Visibilité/online + compte privé** · Recherche · Signets | Épingler un post · Collections de signets · Explorer/Découverte · Hashtag cliquable · Partage d'un profil |
| **Maxime** *(+PPT)* | Composer un breeze · Actions sur post · Paramètres (thème, langue/traduction) | Liker un commentaire · Mention @handle *(notif montrée par Alex)* · *(opt.)* RTL arabe |
| **Alex** | Visiteur · NSFW/mots filtrés/bloqués/MFA/changement mail · Admin (bugs, seuil, Docker, création compte) | **Centre de notifications** · **Événements live** (a posté / follower / like) · **Vidéo + lightbox** · Transférer bug→modération |
| **Philippe** *(+PPT)* | Messagerie E2EE · Modération (tickets, corbeille, NSFW, ban, certif) | Reçus/pièce jointe/édition/suppression/mute · Live messages (typing/online) · Avertir un user · Approuver un ticket |

ℹ️ Charge **équilibrée** : Maxime & Philippe ont la part démo la plus légère (ils assurent le PPT de 5 min en amont) ; Alex & Romain portent les blocs autonomes les plus volumineux (pas de PPT). Les étapes *(ajout)* / *(opt.)* sont les premières à alléger si le timing serre.

---

## 0. Intro (15 s)

🗣️ « Breezy est un réseau social distribué type X/Twitter, en **architecture microservices** : un frontend Next.js, une API Gateway unique, et une dizaine de services Go indépendants avec leurs propres bases. On va dérouler le parcours d'un **visiteur**, puis d'un **utilisateur**, puis les rôles **modérateur** et **administrateur** — les trois rôles sont fonctionnels. »

---

## 1. VISITEUR (déconnecté) — **[Alex]**

> But : montrer qu'on peut consulter sans compte, et que les actions sensibles redirigent vers le login.

1. `[CLIC]` Aller sur `https://breezy.philippeluu.fr` (fenêtre privée / déconnecté).
2. 🗣️ « Sans être connecté, on accède déjà au **fil public** : un visiteur peut découvrir le contenu. »
3. `[CLIC]` **Scroller le feed** → 🗣️ « Pagination infinie : on ne charge pas tout d'un coup. »
4. `[CLIC]` **Ouvrir un post** → 🗣️ « On peut lire un post et son fil. »
5. `[CLIC]` Montrer les **Tendances** (#hashtags) → 🗣️ « Les hashtags sont extraits **côté serveur** et classés en tendances. »
6. `[CLIC]` Tenter d'aller sur le **profil de quelqu'un** (ou de liker) → **redirection vers `/login`**.
   🗣️ « Dès qu'une action demande une identité, on est redirigé vers la connexion. »
   🛡️ « La sécurité ne dépend jamais du front : **chaque service revalide le JWT** et applique la barrière de visibilité côté serveur. »

---

## 2. AUTHENTIFICATION — **[Romain]**

> But : montrer inscription sécurisée + connexions sociales + garde-fous légaux.

1. 🗣️ « On peut se connecter de façon classique, ou via **Google** ou **GitHub** (OAuth). » `[CLIC]` montrer les boutons OAuth (sans forcément aller au bout).
2. `[CLIC]` **Créer un compte** :
   - 🗣️ « Inscription classique, e-mail + mot de passe. »
   - `[CLIC]` Montrer qu'on **doit cocher l'acceptation des CGU** pour valider. 🗣️ « L'acceptation des CGU est **obligatoire** (on n'oblige plus à scroller les mentions, mais à les accepter). »
   - `[CLIC]` Tester une **date de naissance < 13 ans** → refus. 🗣️ « Âge minimum **13 ans**, sinon inscription impossible. »
     🛡️ « Le seuil **13 ans** (inscription) est distinct du seuil **18 ans** (NSFW), et la majorité est **recalculée serveur** depuis la date de naissance — le front ne peut pas mentir. »
3. 🗣️ « À l'inscription, un **e-mail de validation** est envoyé par notre mail-service ; tant qu'il n'est pas validé, l'accès est bloqué (gate `email_verified`). »
   🛡️ (si question) « Après le pentest, on a durci : **rate-limiting** sur login/2FA/inscription/reset, en-têtes de sécurité (HSTS, CSP, anti-clickjacking). »
4. `[CLIC]` **Se connecter** avec le compte de démo principal.

---

## 3. UTILISATEUR

### 3.1 Publier un breeze (composer) 🔊 — **[Maxime]**

1. `[CLIC]` Ouvrir le **composer** de post.
2. `[CLIC]` Montrer l'**audience de réponse** : 🗣️ « Je choisis qui peut répondre — **tout le monde**, ou **seulement mes abonnés**. » Montrer les deux options.
3. `[CLIC]` Ajouter un **emoji** et un **GIF** (GIPHY). 🗣️ « Emojis et GIFs intégrés via GIPHY. »
4. `[CLIC]` Ajouter une **image** (puis 🗣️ « jusqu'à 4 médias »). *(Optionnel : ajouter une **vidéo** — voir 3.2.)*
5. `[CLIC]` Créer un **sondage** :
   - 🗣️ « Sondage avec options, et durée. »
   - `[CLIC]` Tenter **0 minute** → refusé. 🗣️ « Durée nulle impossible (et plafonnée à 7 jours). »
   - 🛡️ « Les résultats sont **cachés tant qu'on n'a pas voté** (compteurs à zéro), révélés après le vote unique — visibilité gérée serveur. »
6. `[CLIC]` Cocher **contenu sensible (NSFW)** sur le post. 🗣️ « Je peux marquer mon post comme sensible. »
7. `[CLIC]` **Publier** 🔊 (son d'envoi).

### 3.2 Médias : vidéo & lightbox *(ajout)* — **[Alex]**

1. `[CLIC]` Sur un post avec **vidéo**, montrer l'**autoplay façon Twitter** (lecture auto, muet).
2. `[CLIC]` **Cliquer une image** d'un post → ouverture en **lightbox** (visionneuse plein écran).
   🛡️ « Le média-service est **agnostique du contenu** (octets opaques dans MinIO) et tout est **streamé via la gateway** — jamais d'URL MinIO exposée directement. »

### 3.3 Le feed (Pour toi / Abonnements) — **[Maxime]** *(NSFW flouté : [Alex])*

1. **[Maxime]** `[CLIC]` En haut du feed, basculer **Pour toi** → **Abonnements**. 🗣️ « Deux fils : découverte (Pour toi) et le fil de mes abonnements. »
2. **[Alex]** `[CLIC]` En scrollant, montrer un **post NSFW flouté** → 🗣️ « Le contenu sensible est flouté selon la politique d'âge ; on reviendra sur le réglage dans les paramètres. »

### 3.4 Actions sur un post 🔊 — **[Maxime]** *(partage de profil : [Romain])*

1. `[CLIC]` **Liker** un post 🔊. 🗣️ « Like (sur les posts **et** les commentaires). »
2. `[CLIC]` **Commenter** un post.
3. `[CLIC]` **Répondre à un commentaire** → 🗣️ « Fil de commentaires sur **2 niveaux**. »
4. `[CLIC]` **Liker un commentaire** *(ajout, rapide)*.
5. `[CLIC]` **Republier** puis **Citer** (faire la **citation** : repost avec commentaire). 🗣️ « Repost simple ou citation. »
6. `[CLIC]` **Mettre en signet** un post (on le retrouvera plus tard).
7. `[CLIC]` **Partager** un post (lien). *(Mentionner aussi : partage d'un **profil**.)*
8. `[CLIC]` **Survoler** l'avatar/nom d'une personne → **carte de survol (hover card)** → `[CLIC]` **S'abonner** depuis le hover. 🗣️ « L'identité est **toujours cliquable** et prévisualisable. »
9. `[CLIC]` **Mentionner @quelqu'un** dans un nouveau post *(ajout)* → 🗣️ « Les mentions sont cliquables et **génèrent une notification** au mentionné (on le verra dans les notifs). »
10. `[CLIC]` **Signaler un post** → choisir un motif. 🗣️ « On reviendra dessus en tant que modérateur. »
    🛡️ « Un signalement porte sur le contenu d'un **autre** service → on a un **report-service dédié** : ça renforce la cohérence microservices. »

### 3.5 Recherche, hashtags & Explorer *(enrichi)* — **[Romain]**

1. `[CLIC]` **Cliquer un #hashtag** dans un post → arrive sur les **résultats filtrés** *(ajout)*.
2. `[CLIC]` Ouvrir la **recherche** : rechercher **« breez »**.
3. `[CLIC]` Filtrer par **Publications**, puis par **Comptes** (utilisateurs), puis montrer les deux. 🗣️ « Recherche unifiée comptes + hashtags + posts. »
4. `[CLIC]` Montrer la page **Explorer / Découverte** *(ajout)* → 🗣️ « Page de découverte de contenus et de comptes. »

### 3.6 Profil — **[Romain]**

1. `[CLIC]` Aller sur **son propre profil**.
2. `[CLIC]` Montrer les **3 onglets** : **Publications**, **Réponses**, **J'aime**. 🗣️ « Les contenus sont séparés sans multiplier les pages. »
3. `[CLIC]` **Épingler un post** en haut du profil *(ajout)* → 🗣️ « On peut épingler un breeze en tête de profil. »
4. `[CLIC]` Montrer **Abonnements / Abonnés** (compteurs + listes).
5. `[CLIC]` **Éditer le profil** :
   - `[CLIC]` Changer **bannière** et **photo de profil**, montrer qu'elles acceptent un **GIF animé** → montrer la **photo de profil de `@faker`** (animée).
   - `[CLIC]` Renseigner **nationalité**, **sexe**, **localisation**.
   - 🗣️ « Le **username** et le **display name** ont un **délai d'attente** (cooldown) entre deux changements — anti-abus. »
   🛡️ « L'identité technique (username, rôle, statut, graphe social) vit dans **user-service** ; le décoratif (nom affiché, bio, avatar, nationalité, visibilité) dans **profil-service** — une donnée = un service. »

### 3.7 Paramètres — **[Maxime]** (apparence/langue) → **[Romain]** (visibilité/privé) → **[Alex]** (NSFW/mots filtrés/bloqués/MFA/e-mail)

1. **[Maxime] Apparence :** `[CLIC]` **mode clair / sombre**, puis **thème personnalisé** (couleur d'accent). *(Optionnel wow : passer en **arabe** → l'interface bascule en **RTL**.)*
2. **[Maxime] Langue & traduction :** `[CLIC]` changer la **langue** (12 dispo) → 🗣️ « Préférence de langue par compte. » `[CLIC]` sur un post en langue étrangère, **Traduire** → 🗣️ « Traduction automatique selon la langue source. »
3. **[Romain] Visibilité / activité en ligne :** 🗣️ « On peut afficher ou masquer son **statut en ligne**. » `[CLIC]` montrer un profil **en ligne**, puis montrer **`@zervixt`** dont l'activité est **désactivée** (pas de "en ligne / dernière connexion").
4. **[Romain] Compte privé :** `[CLIC]` passer son compte en **privé** → depuis un autre compte, montrer que l'abonnement devient une **demande à accepter**.
5. **[Alex] Contenu sensible (NSFW) :** 🗣️ « Réglage d'affichage du NSFW. » 🛡️ « Pour un **mineur**, le réglage est **inerte** : `nsfw_visible = majeur ET activé` — le NSFW reste **toujours flouté** pour les −18 ans, décidé serveur. »
6. **[Alex] Mots masqués :** `[CLIC]` ajouter un **mot filtré** → les posts le contenant sont masqués.
7. **[Alex] Utilisateurs bloqués :** `[CLIC]` montrer la liste / **bloquer** un utilisateur.
8. **[Alex] MFA (2FA) :** `[CLIC]` montrer l'activation **TOTP** (QR code). 🗣️ « Double authentification par appli (TOTP). »
9. **[Alex] Changement e-mail / mot de passe :** `[CLIC]` lancer un **changement d'e-mail** → montrer le **mail de confirmation**. 🛡️ « Un changement de mot de passe / e-mail / un ban **révoque toutes les sessions**. »

### 3.8 Notifications *(ajout — gros manque comblé)* 🔊 — **[Alex]**

1. `[CLIC]` Ouvrir la **cloche / centre de notifications**.
2. 🗣️ « Les notifications sont **agrégées style Instagram** (likes, commentaires, réponses, mentions, follows, demandes de follow, reposts, citations) — on ne sature pas l'utilisateur. Un **badge** compte les non-lues, le tout en **temps réel via WebSocket**. »
   🛡️ « L'émission de notifications est **best-effort, hors gateway** (secret interne) : si le service est lent, le reste de l'app n'est jamais bloqué. »

### 3.9 Messagerie chiffrée (E2EE) 🔊 — **[Philippe]**

> *Voir aussi le script détaillé `Discours_Demo_Soutenance.md` pour les phrases longues de sécurité.*

1. `[CLIC]` Ouvrir **Messages** → l'écran de **passphrase** apparaît (vue floutée tant que l'identité n'est pas déverrouillée).
2. 🗣️ « Notre messagerie est **chiffrée de bout en bout**. Tout le chiffrement se fait **dans le navigateur** ; le serveur ne voit que des **enveloppes** (ciphertext + nonce) et des **métadonnées** — jamais de clair. Le serveur est **aveugle** : même un admin **ne peut pas lire un DM**. »
   🛡️ « Paire **X25519** par utilisateur (clé privée en IndexedDB), clé de conversation **scellée par membre**, contenu en **XChaCha20-Poly1305**. »
3. `[CLIC]` Saisir la **passphrase** → déverrouillage.
   🗣️ « La passphrase chiffre une **sauvegarde de clé** en **zero-knowledge** (KDF **Argon2id**, minimum OWASP, dans un Web Worker) → je retrouve mes conversations sur un autre appareil **sans** que le serveur voie ni la passphrase ni la clé privée. »
   🛡️ « Passphrase oubliée = sauvegarde **irrécupérable** : c'est le prix du zero-knowledge, on a refusé tout séquestre. »
4. `[CLIC]` Envoyer un message ; montrer les **reçus** sous mon dernier message : **1 coche = Envoyé**, **2 coches couleur = Ouvert** *(ajout)*.
   🛡️ « Les reçus sont des **timestamps serveur** (Remis / Ouvert), **jamais** le contenu → l'E2EE reste intact, et l'état "lu" est **cohérent multi-appareils**. »
5. `[CLIC]` **Envoyer une pièce jointe (image)** dans la conversation *(ajout)* → 🗣️ « La pièce jointe est **chiffrée côté client** : un blob opaque pour le serveur. »
6. `[CLIC]` **Éditer** un de mes messages *(ajout)* → 🗣️ « Édition possible (la 1ʳᵉ version chiffrée est conservée, affichage atténué). »
7. `[CLIC]` **Supprimer pour tous** un message *(ajout)* → 🗣️ « Suppression réelle : le ciphertext est effacé — le serveur était déjà aveugle, donc c'est une suppression honnête. »
8. `[CLIC]` **Signaler un message** *(reviendra côté modo)* → 🗣️ « Pour modérer sans casser l'E2EE : **c'est le destinataire qui divulgue volontairement** la copie en clair. Le serveur ne déchiffre **jamais** de lui-même. »
9. `[CLIC]` **Mute** une conversation *(ajout)* → 🗣️ « Mettre en sourdine : exclu du badge, mais toujours marqué non-lu dans la liste. »
10. **Groupes :** `[CLIC]` ouvrir un **groupe** → **Inviter** des membres. 🗣️ « Tout membre peut inviter : il **emballe la clé** pour la clé publique du nouvel arrivant → groupe **admin-proof** comme les DM. »
11. **Communautés :** `[CLIC]` **Découvrir les communautés** (annuaire public) → **Rejoindre**. 🗣️ « Les communautés sont **semi-publiques** (modèle canal Telegram) : annuaire public, viewers illimités. »
    🛡️ « Compromis **assumé et borné** : l'E2EE strict est **impossible** avec des viewers inconnus illimités → pour les communautés **seulement**, la clé est côté serveur (admin-readable **par conception**). DM et groupes restent admin-proof. »

### 3.10 Signets (bookmarks) *(enrichi)* — **[Romain]**

1. `[CLIC]` Ouvrir **Signets** → retrouver le post mis en signet en 3.4.
2. `[CLIC]` Montrer les **collections** de signets *(ajout)* → 🗣️ « Les signets se rangent en collections. »

### 3.11 Temps réel — À FAIRE EN LIVE 🔊 — **[Alex]** (feed) + **[Philippe]** (messages)

> Nécessite un 2ᵉ intervenant / 2ᵉ compte sur un autre écran.

1. **[Alex] « A posté » :** rester sur le feed ; le 2ᵉ compte **publie un post** → un **bandeau temps réel "X a posté"** apparaît 🔊. 🗣️ « Bandeau temps réel via WebSocket exposé par post-service et relayé par la gateway. »
2. **[Alex] Nouveau follower :** le 2ᵉ compte **s'abonne** → **notification** en direct 🔊.
3. **[Alex] Like en direct :** le 2ᵉ compte **like** un de mes posts → **notification** 🔊.
4. **[Philippe] Messages live :** ouvrir une conversation avec le 2ᵉ compte → il **écrit** : montrer **« en train d'écrire… »** et le statut **en ligne** 🔊, puis réception du message en direct.

---

## 4. MODÉRATEUR — **[Philippe]**

> 🗣️ « Je passe en **modérateur**. » `[CLIC]` ouvrir le **centre de modération**.

1. **Tickets de signalement :**
   - `[CLIC]` Montrer la **liste des tickets** et les **filtres cumulables** : **statut (ouvert / réouvert / fermé)**, **dates**, **nombre minimum de signalements**.
   - 🛡️ « N signalements d'un même contenu = **un seul ticket agrégé** (index unique partiel). Anti-spam : **1 signalement par personne et par contenu**, et il faut **2 personnes distinctes** pour rouvrir un ticket. »
   - `[CLIC]` Ouvrir le **ticket du message** signalé en 3.9 → 🗣️ « Le serveur n'a pas déchiffré : c'est la **copie divulguée** par le destinataire. »
2. **Avertir un utilisateur** *(ajout)* : `[CLIC]` envoyer un **avertissement** → 🗣️ « L'utilisateur recevra une **modale bloquante** à sa prochaine connexion (acquittement requis). »
3. **Valider / approuver un ticket** *(ajout)* : `[CLIC]` marquer conforme → 🗣️ « Statut terminal **approuvé** : le contenu est démasqué et **tout nouveau signalement est refusé** (anti-harcèlement par signalement). »
4. **Supprimer un tweet → corbeille → restaurer :**
   - `[CLIC]` **Supprimer** un post. 🗣️ « Ce n'est **pas définitif** : c'est un **soft-delete**. »
   - `[CLIC]` Onglet **« Tweets supprimés »** → montrer le post → `[CLIC]` **Restaurer**.
   - 🗣️ « On garde le contenu pour pouvoir le **restaurer en cas de poursuites** ou le conserver comme preuve. Purge définitive possible aussi. »
5. **Marquer comme NSFW :** `[CLIC]` marquer un post **sensible** → il reste public mais **flouté** selon la politique d'âge serveur.
6. **Bannir un utilisateur :**
   - `[CLIC]` Sur la cible (encart **profil de risque**) → **Bannir**.
   - `[CLIC]` *(si possible)* se reconnecter avec le compte banni → message **« Compte désactivé, veuillez contacter le support de Breezy »** (403) ; le profil affiche « banni, plus accessible ».
   - 🛡️ « Le ban **révoque toutes les sessions**. Et un **modérateur ne peut pas bannir un admin** — gating de rôles côté serveur. »
7. **Attribuer une certification :**
   - `[CLIC]` Sur un profil → **Attribuer une certification** : **Politique** (sceau **doré**) ou **Personnalité publique / influenceur** (sceau **bleu**).
   - `[CLIC]` Montrer le **badge** sur le profil + le **hover** (libellé).
   - 🗣️ « Et il existe un **3ᵉ badge, Staff** (logo Breezy), **automatique** pour modo/admin — **non attribuable à la main**, pour éviter l'usurpation d'un statut officiel. »
   - 🛡️ « Route réservée modo/admin, **enum fermé** (anti-injection). »

---

## 5. ADMINISTRATEUR — **[Alex]**

> 🗣️ « En **administrateur**, j'ai des droits de gouvernance supplémentaires. » `[CLIC]` ouvrir le **panel admin** (onglets Signalements/bugs · Paramètres · Infrastructure).

1. **Tickets de bug :**
   - `[CLIC]` Onglet **Signalements (bugs)** → filtres **ouvert / réouvert / dates**.
   - `[CLIC]` *(ajout)* **Transférer un bug → modération** → 🗣️ « Un signalement classé bug peut être requalifié en modération. »
   - 🛡️ « Le formulaire de signalement est **unique** : le motif "Bug technique" range automatiquement le ticket côté Administration, les autres motifs côté modération. »
2. **Seuil d'auto-masquage :**
   - `[CLIC]` Onglet **Paramètres** → régler le **seuil** (ex. 5).
   - 🗣️ « Au-delà de **X signalements**, un post est **automatiquement masqué** (sort des fils). **0 = désactivé.** »
   - 🛡️ « Compteur dans report-service, mais la **visibilité reste chez post-service** via un appel serveur-à-serveur **hors gateway** (secret interne) — jamais sur la foi du front. »
3. **Infrastructure / Docker :**
   - `[CLIC]` Onglet **Infrastructure** → état des **conteneurs / services** (monitoring séparé de la modération de contenu).
   - 🗣️ « Surveillance de l'état des services ; **conteneurisation complète** (docker-compose), en prod seul le reverse-proxy TLS est exposé, bases et services sur **réseau interne**. »
4. **Création d'un compte par l'admin :**
   - `[CLIC]` Créer un compte (ex. pseudo **`zervixt`**) → l'admin **génère un mot de passe**.
   - 🗣️ « L'admin peut **créer un compte** ; un **mot de passe temporaire** est généré et envoyé par mail-service. »

---

## 6. Conclusion (20 s)

🗣️ « Pour résumer : une **architecture microservices cohérente** (une donnée = un service, tout via la gateway), une **sécurité** sérieuse (JWT revalidé partout, sessions courtes, messagerie **E2EE** à serveur aveugle, durcissement post-pentest), **toutes les fonctionnalités primaires** plus de nombreuses secondaires (sondages, médias, traduction, signets, communautés…), et **les trois rôles pleinement fonctionnels** — User, Modérateur, Administrateur. Le tout **entièrement conteneurisé** et déployé en production. »

---

## Annexe — Réponses aux questions pièges (rappel)

| Question | Réponse courte |
|---|---|
| Un admin peut-il lire un DM ? | Non (clé scellée par membre, serveur aveugle). Communautés = oui, par conception annoncée. |
| L'état "lu" / les reçus cassent-ils l'E2EE ? | Non : **timestamps / métadonnées**, jamais le contenu. |
| Passphrase oubliée ? | Irrécupérable — prix du zero-knowledge, pas d'escrow. |
| Modérer un message chiffré ? | Le **destinataire divulgue** volontairement ; le serveur pose juste un tombstone par id. |
| Spam de réouverture de ticket ? | 1 signalement / personne / contenu + seuil = **2 personnes distinctes**. |
| Les 3 rôles sont-ils fonctionnels ? | User signale ; Modo traite/avertit/retire/bannit/certifie ; Admin règle seuil, transfère bugs, crée des comptes, monitore — gating mod ≠ admin. |
| NSFW protège-t-il les mineurs ? | Majorité **calculée serveur** (date de naissance), jamais sur la foi du front. |
| Suppression = vraie suppression ? | Post : soft-delete réversible (corbeille, juridique). Message : ciphertext **réellement effacé** (serveur aveugle). |
| Pourquoi un report-service séparé ? | Un signalement n'appartient à aucun service métier → cohérence microservices. |
