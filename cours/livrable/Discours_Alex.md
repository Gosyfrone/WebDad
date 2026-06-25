# Discours de démonstration — ALEX

> Parties : **Visiteur** · **Vidéo & lightbox** · **NSFW (feed)** · **Paramètres sécurité** (NSFW/mots filtrés/bloqués/MFA/e-mail) · **Notifications** · **Temps réel (feed)** · **Administration**.
> Format : `[CLIC]` = action écran · 🗣️ = ce qu'on dit · 🛡️ = le choix technique défendu.
> Principe transverse : **zéro confiance au front** + **on choisit le bon mécanisme selon l'échelle** (pas de WebSocket partout).

---

## A. VISITEUR (déconnecté)

1. `[CLIC]` Aller sur `https://breezy.philippeluu.fr` (déconnecté).
2. `[CLIC]` **Scroller le feed** → 🗣️ « Sans compte, on accède déjà au **fil public**, en pagination infinie. »
3. `[CLIC]` **Ouvrir un post**, puis montrer les **tendances**.
4. `[CLIC]` Tenter d'aller sur un **profil** ou de **liker** → redirection / modale **« Connecte-toi »**.
🛡️ « La vue visiteur est **100 % front, zéro backend ajouté** : le post-service exposait **déjà** la lecture publique. Sans token, la **barrière de visibilité serveur** ne renvoie que les posts publics. La vraie subtilité technique, ce n'était pas la garde mais la **redirection** : on garde chaque appel enrichi derrière la présence d'un token, et on remplace la redirection forcée par une **modale d'invitation** — le visiteur reste sur le fil. La **lecture** est libre, seule l'**écriture** est gatée. »

---

## B. VIDÉO & LIGHTBOX

1. `[CLIC]` Sur un post **vidéo**, montrer l'**autoplay façon Twitter** (muet, en boucle).
2. `[CLIC]` **Cliquer une image** → ouverture en **lightbox** plein écran.
🛡️ « La vidéo est **streamée par la gateway** (avec Range/seek), donc pas de problème mémoire, et elle est **montée paresseusement** : une vidéo loin dans un feed infini n'est ni téléchargée ni jouée tant qu'elle n'approche pas de l'écran. L'autoplay se déclenche quand elle est visible à 60 %. Tout ça est front, **zéro coût serveur**. »

---

## C. NSFW DANS LE FEED

1. `[CLIC]` En scrollant, montrer un **post NSFW flouté**.
🗣️ « Le contenu sensible est flouté ; le réglage est dans les paramètres. »
🛡️ « Le serveur **délivre** le post marqué sensible, et le front le **floute** si la politique ne l'autorise pas — c'est un filtre d'affichage façon X. Mais la **politique d'âge, elle, est autoritative côté serveur** : voir le point suivant. »

---

## D. PARAMÈTRES — SÉCURITÉ DU COMPTE

### D.1 Contenu sensible (NSFW)
🗣️ « Réglage d'affichage du NSFW. »
🛡️ « La règle est `nsfw_visible = majeur ET activé`. La **majorité est recalculée par le serveur** depuis la date de naissance — **jamais stockée** — donc ça se débloque tout seul le jour des 18 ans, sans tâche planifiée, et **un mineur ne peut pas se faire passer pour majeur** en bidouillant le front. Chez un mineur, le réglage est **inerte**. »

### D.2 Mots masqués
1. `[CLIC]` Ajouter un **mot filtré** → les posts qui le contiennent sont masqués.
🛡️ « C'est volontairement **100 % front, sans backend** : c'est un état **cosmétique** par compte, pas une donnée de domaine. Savoir où **ne pas** créer de service fait partie de l'architecture. »

### D.3 Utilisateurs bloqués
1. `[CLIC]` Montrer la liste / **bloquer** un utilisateur.
🛡️ « Le blocage est une **relation du graphe social** → il vit dans **user-service**, à côté des follows. Mais son **effet** est appliqué **côté serveur dans post-service** : un utilisateur bloqué (ou qui m'a bloqué) ne voit plus mes posts — dans le feed, le détail, les réponses, les likes. Le front masque pour l'UX, mais la **vraie barrière est serveur**. »

### D.4 MFA (2FA)
1. `[CLIC]` Montrer l'activation **TOTP** (QR code).
🗣️ « Double authentification par appli (compatible Google/Microsoft Authenticator). »
🛡️ « Le **QR est généré côté serveur** (on s'interdit d'ajouter une dépendance npm au front). Le secret est **chiffré en base (AES-256-GCM)** avec une clé **hors base**. Au login, si le 2FA est actif, on valide le mot de passe mais **on n'émet aucun JWT** : on crée un challenge court, et on vérifie le code **avant** de le consommer → **un code faux ne brûle pas le challenge**. Et anti-brute-force : le challenge est **détruit après 5 codes faux** — ce qui couvre même un attaquant multi-IP. »

### D.5 Changement d'e-mail / mot de passe
1. `[CLIC]` Lancer un **changement d'e-mail** → montrer le **mail de confirmation**.
🛡️ « Le changement d'e-mail se fait **en deux temps sans couper l'accès** : la nouvelle adresse reste "en attente" tant que le lien n'est pas cliqué → une faute de frappe ne verrouille jamais le compte. La confirmation est une **transaction**, et un changement de mot de passe / e-mail / un **ban révoque toutes les sessions**. Si l'envoi du mail échoue, on **annule proprement** la demande plutôt que d'annoncer un lien inexistant. »

---

## E. NOTIFICATIONS 🔊

1. `[CLIC]` Ouvrir la **cloche / centre de notifications**.
🗣️ « Notifications **agrégées style Instagram** : likes, commentaires, réponses, **mentions**, follows, demandes de follow, reposts, citations. On ne sature pas l'utilisateur. »
🛡️ « Techniquement : un **service de notifications dédié** reçoit les événements et les **regroupe par clé** — plusieurs likes sur un post = **une** notification qui s'incrémente. Le badge de non-lus est tenu **en mémoire côté front** avec **un seul WebSocket**, au lieu d'une requête serveur par événement pendant les pics. Et l'émission est **best-effort, hors gateway** : si le service de notif est lent ou tombe, l'action métier (le like, le follow) **n'est jamais bloquée**. »

---

## F. TEMPS RÉEL — EN LIVE (feed) 🔊

> Nécessite un 2ᵉ compte piloté par un coéquipier.

1. **« A posté » :** le 2ᵉ compte **publie** → bandeau **« X a posté »** 🔊.
2. **Nouveau follower / like en direct :** le 2ᵉ compte **s'abonne** puis **like** → notifications en direct 🔊.
🛡️ « Trois mécanismes temps réel, **choisis selon l'échelle** — c'est notre angle de défense : on ne met pas du WebSocket partout aveuglément.
 • Le **bandeau "a posté"** est un WebSocket **léger** : il ne transporte **pas le contenu**, juste un ping `(post, auteur)`. Le front **refetch** ensuite via l'endpoint normal → le WS reste léger **et** la barrière de visibilité n'est pas dupliquée. Un compte privé ne ping personne.
 • Les **notifications** passent par WebSocket (pertinent : ciblé par destinataire).
 • Mais les **compteurs de likes/commentaires** sont en **polling groupé** toutes les ~7 s, **pas** en WebSocket : pousser chaque like de chaque post public à tout le monde serait ingérable sans coalescence. Une requête légère pour toute la page affichée suffit largement. »

---

## G. ADMINISTRATION

> 🗣️ « En **administrateur**, j'ai des droits de gouvernance. » `[CLIC]` ouvrir le **panel admin** (onglets Signalements/bugs · Paramètres · Infrastructure).

### G.1 Tickets de bug
1. `[CLIC]` Onglet **bugs** → filtres **ouvert / réouvert / dates**.
2. `[CLIC]` **Transférer un bug → modération**.
🛡️ « Le formulaire de signalement est **unique** : le motif "Bug technique" range automatiquement le ticket côté Administration (limite plus longue), les autres motifs côté modération. Un bug peut être **requalifié** en modération s'il s'avère être un problème de contenu. »

### G.2 Seuil d'auto-masquage
1. `[CLIC]` Onglet **Paramètres** → régler le **seuil** (ex. 5). 🗣️ « Au-delà de X signalements, un post est **masqué automatiquement**. 0 = désactivé. »
🛡️ « Le compteur est dans le report-service, mais la **visibilité reste chez post-service**, via un appel **serveur-à-serveur hors gateway** authentifié par un secret interne — même esprit que les notifications. La décision de masquer n'est **jamais** prise sur la foi du navigateur. Et le seuil est un **singleton** seedé de façon idempotente : on n'écrase jamais une valeur déjà réglée. »

### G.3 Infrastructure / Docker
1. `[CLIC]` Onglet **Infrastructure** → état des **conteneurs / services**.
🗣️ « Surveillance de l'état des services, séparée de la modération de contenu. »
🛡️ « **Conteneurisation complète** (docker-compose, un Dockerfile par service). En production, seul le **reverse-proxy TLS** publie des ports : les bases et les services restent sur un **réseau interne**, injoignables de l'extérieur. »

### G.4 Création d'un compte par l'admin
1. `[CLIC]` Créer un compte (pseudo **`zervixt`**) → l'admin **génère un mot de passe**.
🛡️ « L'admin crée le compte avec un **mot de passe temporaire mailé** et une **vérification d'e-mail obligatoire** : cliquer le lien vérifie l'adresse, ouvre la session, et affiche une **modale bloquante de changement de mot de passe**. Si le username demandé est déjà pris, il est suffixé automatiquement et marqué "à régulariser". »

---

## Annexe — Q&A (Alex)

| Question | Réponse |
|---|---|
| Le visiteur peut-il voir un post privé ? | Non : lecture publique + barrière serveur → seuls les posts publics sortent. |
| Le NSFW protège-t-il vraiment les mineurs ? | Oui : majorité **calculée serveur** (date de naissance), jamais sur la foi du front. |
| Pourquoi pas du WebSocket pour les likes ? | À grande échelle, pousser chaque like à chaque connexion est ingérable sans coalescence ; un polling groupé toutes les ~7 s suffit. |
| Le bandeau "a posté" fuite-t-il les posts privés ? | Non : gating compte-public à l'émission **et** refetch via l'endpoint à barrière serveur. |
| Que se passe-t-il si le service de notif tombe ? | Émission best-effort → l'événement est perdu (assumé), **jamais** de blocage de l'action métier. |
| Comment évite-t-on de brute-forcer un TOTP ? | Rate-limit par IP **+** challenge détruit au 5ᵉ code faux. |
| Le seuil d'auto-masquage dépend-il du front ? | Non : décision serveur, appel inter-service signé ; le front n'a aucune autorité. |
