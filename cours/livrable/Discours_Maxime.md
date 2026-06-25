# Discours de démonstration — MAXIME

> Parties : **Publier un breeze (composer)** · **Feed** · **Actions sur un post** · **Apparence / Thème** · **Langue & traduction**.
> Format : `[CLIC]` = action écran · 🗣️ = ce qu'on dit · 🛡️ = le choix technique défendu.
> Principe transverse : **tout le contenu social vit dans post-service**, et **la visibilité est tranchée côté serveur**.

---

## A. PUBLIER UN BREEZE (composer) 🔊

🗣️ « Le cœur du produit : la publication. Tout passe par **post-service** (sur MongoDB), propriétaire unique des posts, commentaires, likes, sondages, hashtags. »

### A.1 Audience de réponse
1. `[CLIC]` Montrer le sélecteur **« qui peut répondre »** : **tout le monde** / **abonnés seulement**.
🛡️ « L'audience de réponse est posée **à la création** et **non modifiable** ensuite. Et elle est appliquée **côté serveur** : un post réservé aux abonnés refuse la réponse d'un non-abonné avec une erreur, pas juste en grisant un bouton. Le front grise le composer pour l'UX, mais ce n'est **jamais** lui l'autorité. »

### A.2 Emoji & GIF
1. `[CLIC]` Ajouter un **emoji** et un **GIF**.
🛡️ « Les GIFs passent par un **proxy GIPHY porté par notre media-service** : la clé d'API reste **côté serveur**. Et le GIF choisi est **recopié dans notre stockage MinIO**, on ne hotlink jamais l'URL externe de Giphy — ça nous rend **same-origin** (compatible avec notre politique de sécurité CSP) et immunisé contre les 403/404 du CDN tiers. »

### A.3 Image (et vidéo)
1. `[CLIC]` Ajouter une **image** (jusqu'à 4 médias).
🛡️ « Le média part dans **media-service**, qui est **content-agnostic** : il ne stocke que des **octets opaques** sous un id aléatoire, sans base annexe. Le post ne garde qu'une **référence**. Et **tout le téléchargement repasse par la gateway** — jamais d'URL MinIO exposée directement. On vérifie le type par **magic-bytes** côté serveur, avec un plafond de taille. »

### A.4 Sondage
1. `[CLIC]` Créer un **sondage** avec options + durée.
2. `[CLIC]` Tenter **0 minute** → refusé. 🗣️ « Durée nulle impossible (et plafonnée à 7 jours). »
🛡️ « Un vote = une ligne garantie par un **index unique** dans Mongo (un vote par personne et par sondage), pas une garde d'interface. Et les **résultats sont cachés tant qu'on n'a pas voté** : un électeur reçoit des compteurs à zéro, et ne voit les vrais chiffres qu'**après son vote**, ou à la clôture. L'auteur, lui, a toujours les compteurs en direct. »

### A.5 NSFW
1. `[CLIC]` Cocher **contenu sensible**.
🛡️ « Je marque mon post comme sensible. Le post reste **public**, mais sera **flouté** pour qui ne doit pas le voir — et ça, c'est décidé **côté serveur** selon l'âge. »

### A.6 Publier
1. `[CLIC]` **Publier** 🔊 (son d'envoi).

---

## B. LE FEED (Pour toi / Abonnements)

1. `[CLIC]` Basculer **Pour toi** → **Abonnements**.
🗣️ « Deux fils : découverte, et le fil de mes abonnements, en **pagination infinie**. »
🛡️ « Le point central : **chaque lecture de post applique une barrière de visibilité côté serveur** — feed, détail, profil, réponses, likes. Un post privé qu'on n'a pas le droit de voir est tout simplement **absent** de la réponse. La fonction vérifie la visibilité du profil et le blocage, en **mémoïsant par auteur** pour rester rapide sur un feed. »

---

## C. ACTIONS SUR UN POST 🔊

### C.1 Liker (post et commentaire)
1. `[CLIC]` **Liker** un post 🔊, puis **liker un commentaire**.
🛡️ « Les likes sont **idempotents** via un index unique (un like par personne) et les compteurs sont **dénormalisés** sur le document pour un rendu rapide — pas de recomptage à chaque affichage. »

### C.2 Commenter & répondre
1. `[CLIC]` **Commenter**, puis **répondre à un commentaire**.
🛡️ « Fil de commentaires sur **2 niveaux** (commentaire racine + réponses), avec un **point d'entrée unique** côté serveur qui applique les mêmes règles d'audience que le post. »

### C.3 Republier / Citer
1. `[CLIC]` **Republier**, puis **Citer** (repost + commentaire).
🗣️ « Repost simple ou citation, et l'auteur d'origine est **notifié**. »

### C.4 Mentionner @quelqu'un
1. `[CLIC]` Dans un post, **mentionner @quelqu'un** → lien cliquable.
🗣️ « Les mentions sont cliquables et **génèrent une notification** au mentionné — Alex le montrera dans le centre de notifications. »
🛡️ « La logique de mention (la regex, l'insertion, le rendu) est une **brique pure partagée** entre posts, commentaires et messages → une seule source de vérité sur trois surfaces. »

### C.5 Signet & Partage
1. `[CLIC]` **Mettre en signet** (Romain le réouvrira), puis **Partager**.

### C.6 Hover & abonnement
1. `[CLIC]` **Survoler** une personne → **carte de survol** → **S'abonner** depuis le hover.
🗣️ « L'identité est **toujours cliquable et prévisualisable** — partout dans l'app. »

### C.7 Signaler
1. `[CLIC]` **Signaler un post**.
🗣️ « Philippe reprendra ce signalement côté modérateur. »
🛡️ « Un signalement porte sur le contenu d'un **autre** service → on a un **report-service dédié**, plutôt que de polluer post-service. »

---

## D. APPARENCE / THÈME *(Paramètres)*

1. `[CLIC]` Basculer **clair / sombre**, puis **thème personnalisé** (couleur d'accent).
🛡️ « Le clair/sombre repose sur des **surfaces tokenisées** (variables CSS) → le sombre vit à un seul endroit. Le thème personnalisé ajoute 3 cibles indépendantes (fond, texte, couleur primaire) avec un **contraste auto-calculé** pour rester lisible. La roue chromatique est **faite maison** — on s'interdit d'ajouter des dépendances npm — et les maths de couleur sont **pures et testées**. On pré-calcule le thème pour l'appliquer **avant le premier rendu** → pas de clignotement. »

---

## E. LANGUE & TRADUCTION *(Paramètres)*

1. `[CLIC]` Changer la **langue** (12 disponibles). *(Option wow : passer en **arabe** → l'interface bascule en **RTL**.)*
🗣️ « 12 langues, et la préférence est **attachée au compte**. »
🛡️ « L'internationalisation est **maison, sans dépendance**, avec le français comme référence et repli. Et la préférence de langue est une **donnée de compte** (dans user-service), pas une clé navigateur : elle **suit l'utilisateur d'un appareil à l'autre**. »
2. `[CLIC]` Sur un post en langue étrangère, **Traduire**.
🛡️ « La traduction passe par une **route serveur** pour garder la **clé d'API côté serveur** — jamais exposée au navigateur — et **post-service ne stocke aucune traduction**. On a aussi une garde maligne : on n'auto-traduit pas un texte déjà dans ta langue (véto même-langue) pour éviter les fausses traductions. »

---

## Annexe — Q&A (Maxime)

| Question | Réponse |
|---|---|
| Un client custom peut-il voir un post privé ? | Non : la barrière de visibilité est appliquée **à chaque lecture** côté serveur ; le post est absent de la réponse. |
| Comment garantir un seul vote par sondage ? | **Index unique** Mongo (post + utilisateur), pas une garde d'UI. |
| Pourquoi recopier le GIF Giphy au lieu de le hotlinker ? | Same-origin (compatible CSP `self`), immunisé aux 403/404 du CDN ; capture bornée et anti-SSRF par allowlist. |
| Pourquoi une roue chromatique maison ? | Interdiction d'ajouter une dépendance npm + plus simple à défendre ; maths pures testées. |
| La clé de traduction est-elle exposée ? | Non, elle reste dans la route serveur (BFF). |
| L'audience de réponse est-elle contournable côté front ? | Non : refus **serveur** (erreur), le grisage front n'est qu'UX. |
