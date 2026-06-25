# Argumentaire — Remporter l'appel d'offre Breezy

> **Contexte :** plusieurs équipes livrent « Breezy ». Le client (le jury) choisit **un** prestataire.
> Ce document défend **tous nos choix** pour gagner l'appel d'offre. À utiliser pour préparer la défense orale et les questions-réponses.
> **Argument d'autorité immédiat :** notre Breezy est **en production, en ligne, maintenant** → `https://breezy.philippeluu.fr` (front + API + données réelles). On ne présente pas une maquette : on présente un **produit déployé**.

---

## 1. Le pitch qui gagne (30 s)

🗣️ « Vous avez plusieurs Breezy devant vous. Le nôtre se distingue sur trois promesses tenues, pas annoncées :

1. **Une messagerie réellement chiffrée de bout en bout**, où même nous, les administrateurs, **ne pouvons pas lire vos messages privés**. Très peu d'équipes peuvent le démontrer.
2. **Une sécurité qui ne fait jamais confiance au navigateur** : visibilité, âge, modération, tout est tranché **côté serveur**. Même un client modifié ne contourne rien.
3. **Un produit en production, conteneurisé et industrialisé** : CI, documentation d'API publiée, durcissement post-test d'intrusion, conformité RGPD. C'est livrable à un vrai client **dès aujourd'hui**.

Et tout ça avec **les trois rôles fonctionnels** et la **totalité des fonctionnalités primaires** plus de nombreuses secondaires. »

---

## 2. Pourquoi nous — tableau des différenciateurs

| Axe | Un « Breezy » standard | **Notre Breezy** |
|---|---|---|
| Messagerie | « privée » = stockée en clair côté serveur | **E2EE admin-proof** (X25519 + XChaCha20-Poly1305), serveur **aveugle**, sauvegarde de clé **zero-knowledge** |
| Sécurité front | la visibilité dépend du client | **zéro confiance au front** : visibilité, âge/NSFW, modération **tranchés serveur** |
| Architecture | un monolithe découpé en dossiers | **vrais microservices** (1 donnée = 1 service, base par service, schéma embarqué) |
| JWT | vérifié à la gateway seulement | **revalidé dans chaque service** (HS256 épinglé, `alg:none` rejeté) |
| Modération | un champ « hidden » dans posts | **report-service dédié**, tickets agrégés, 3 rôles, corbeille restaurable |
| Déploiement | tourne en local | **en production**, conteneurisé, CI + doc OpenAPI publiée |
| Honnêteté | promet l'E2EE partout | **périmètre de sécurité explicite et borné** (communautés = compromis assumé) |
| Conformité | absente | **RGPD** (purge auto des comptes bannis > 5 ans), durcissement **post-pentest** |

---

## 3. Défense axe par axe (alignée sur la grille du jury)

### 3.1 Architecture microservices cohérente

🗣️ « Notre découpage n'est pas cosmétique. La règle est **une donnée = un service** : le `username` et le graphe social vivent dans **user**, le décoratif (nom affiché, bio, visibilité) dans **profil**, le contenu dans **post**, etc. Les vues agrégées sont **composées par l'appelant**. »

🛡️ Preuves :
- **Tout passe par l'API Gateway** (point d'entrée unique : routage, CORS, relais WebSocket, **streaming des médias**). Seule exception **assumée et documentée** : l'émission de notifications serveur-à-serveur, **hors gateway**, authentifiée par secret interne — parce qu'elle doit être *best-effort* et ne jamais bloquer le parcours utilisateur.
- **Chaque service possède son schéma** (`EnsureSchema` idempotent au boot) — pas de scripts d'init montés, pas de couplage par la base.
- Un signalement ne relève d'**aucun** service métier → on a créé un **report-service dédié** plutôt que de polluer post/message/profil. **C'est ça, penser microservices.**

*Pourquoi pas un monolithe ?* Scalabilité indépendante, isolation des pannes, déploiement par service, équipes parallèles. Le coût (latence inter-service, cohérence) est maîtrisé par la composition côté appelant.

### 3.2 Sécurité — le cœur de notre offre

**Authentification & sessions**
🛡️ « JWT **HS256, algorithme épinglé** (`alg:none` rejeté), **revalidé — signature ET expiration — dans chaque service**, pas seulement à la gateway. L'identité vient **toujours du jeton validé**, jamais du corps de requête ou de l'URL → protection contre l'usurpation d'objet (IDOR/BOLA). »
🛡️ « **Access token 15 min** en Bearer (immunité CSRF native), **refresh token 24 h** en cookie **httpOnly + SameSite + Secure, à rotation** (révoqué à chaque usage). Un changement de mot de passe, un reset, un changement d'e-mail ou un **ban révoque toutes les sessions**. »

**Chiffrement de bout en bout (l'argument massue)**
🛡️ « La messagerie privée est **E2EE** : tout le chiffrement vit dans le navigateur, le serveur ne voit que des **enveloppes** (ciphertext/nonce) et des **métadonnées**. **Même un administrateur ne peut pas lire un DM.** Paire **X25519** par utilisateur, clé de conversation **scellée par membre**, contenu en **XChaCha20-Poly1305**. La sauvegarde multi-appareils est chiffrée sous une passphrase via **Argon2id** (minimum OWASP), en **zero-knowledge** : le serveur ne voit ni la passphrase ni la clé privée. »

**Défense en profondeur**
🛡️ « **Rate-limiting** sur les points sensibles (connexion, 2FA, inscription, mot de passe oublié, renvoi d'e-mail, traduction). **En-têtes de sécurité** : HSTS, CSP, anti-clickjacking, anti-sniffing, Referrer-Policy, Permissions-Policy. **Isolation réseau** en prod : seul le reverse-proxy TLS publie des ports, les bases et services restent sur un réseau interne. **MFA TOTP** disponible. »

**Crédibilité supplémentaire** : on a réalisé un **test d'intrusion** et **corrigé** (durcissement documenté). On ne dit pas « c'est sécurisé », on **le prouve par la démarche**.

### 3.3 Confiance zéro côté front (server-authoritative)

🗣️ « La sécurité ne repose **jamais** sur le client. Trois exemples qui font la différence : »
- **Visibilité des posts** : la barrière (public/privé, follow, blocage, audience) est appliquée **par post-service**, pas par masquage CSS.
- **Majorité / NSFW** : l'âge est **recalculé serveur** depuis la date de naissance (jamais stocké) ; `nsfw_visible = majeur ET activé`. Un mineur ne peut **pas** se faire passer pour majeur en bidouillant le front.
- **Auto-masquage à seuil** : le compteur est dans report-service mais la **visibilité reste chez post-service**, via un appel serveur-à-serveur signé. Jamais sur la foi du navigateur.

### 3.4 Conteneurisation complète & industrialisation

🛡️ « **Full Docker** (docker-compose) : un Dockerfile par service, hot-reload en dev (`air` + `next dev`), images figées en prod. **CI GitHub Actions** : workflows Go et frontend, build Docker, **contrôle de dérive de la doc Swagger** (un PR qui change une route sans régénérer la doc est bloqué). **Documentation d'API publiée** (Redoc) et auto-déployée. **Conformité RGPD** : purge automatique des comptes bannis depuis plus de 5 ans. »

🗣️ « Concrètement, pour le client : c'est **maintenable, reproductible et auditable**. On livre un produit, pas un prototype. »

### 3.5 Toutes les fonctionnalités primaires + secondaires bien choisies

🗣️ « **100 % des fonctionnalités primaires** sont là et fonctionnelles : compte + validation, auth sécurisée, posts 280 caractères, posts sur profil, fil des abonnements, like, commentaire, réponse à 2 niveaux, follow/unfollow, profil, liste des posts. »

🗣️ « Et des secondaires **choisies pour leur valeur**, pas pour faire nombre : messagerie E2EE, signalement + modération, images **et vidéos**, hashtags + recherche + tendances, notifications temps réel, **12 langues avec RTL**, **traduction automatique** des posts, thème personnalisé, sondages, GIFs, signets en collections, communautés. »

### 3.6 Les trois rôles — réellement fonctionnels

🗣️ « **User** : signale du contenu, gère sa vie privée. **Modérateur** : traite les tickets, avertit, retire (soft-delete), marque NSFW, bannit un utilisateur, certifie. **Administrateur** : règle le seuil d'auto-masquage, transfère les bugs, crée des comptes, monitore l'infra, bannit. Avec un **gating strict** : un modérateur ne peut **pas** bannir un administrateur. »

### 3.7 Internationalisation & accessibilité

🛡️ « **12 langues**, préférence **par compte**, **RTL** complet pour l'arabe (l'interface se retourne), **traduction automatique** des posts selon la langue source. Thème clair/sombre/personnalisé, **responsive mobile-first**. » → un produit **prêt à l'international**, pas franco-français.

### 3.8 Temps réel

🛡️ « WebSocket relayé par la gateway : bandeau **« a posté »**, **notifications agrégées** style Instagram, **« en train d'écrire »**, statut **en ligne**, reçus **Envoyé/Ouvert**. Émission de notifications **best-effort hors gateway** → jamais de blocage du parcours. »

---

## 4. Nos choix produits malins (et les alternatives qu'on a écartées)

> Montre la **maturité d'ingénierie** : on a tranché, avec des raisons.

- **Modèle de messagerie hybride** : DM/groupes **admin-proof** (clé scellée par membre) ; communautés **admin-readable** (clé serveur). *Écarté :* E2EE strict pour les communautés — **mathématiquement impossible** avec des viewers inconnus illimités sans détenteur de clé en ligne. On préfère un compromis **honnête et borné** à une fausse promesse.
- **Argon2id 19 Mio / t=2** plutôt que 64 Mio / t=3 : compromis **mobile** (la dérivation JS est synchrone et figeait le téléphone 15-25 s), tout en restant au **minimum OWASP**, avec auto-upgrade des anciennes sauvegardes.
- **État « lu » = donnée serveur** (timestamps) plutôt que localStorage par appareil : **cohérent multi-appareils**, et **sans casser l'E2EE** (ce sont des métadonnées, jamais le contenu).
- **Modération d'un message chiffré par divulgation du destinataire** (façon WhatsApp/Signal) plutôt que déchiffrement serveur : on **modère sans casser l'E2EE**.
- **Suppression = tombstone honnête** : le serveur étant aveugle, effacer le ciphertext est une **vraie** suppression « pour tous ».
- **Réutiliser l'existant** (soft-delete, ban, rôles) pour la modération plutôt que réimplémenter : moins de code, plus de cohérence.
- **Migrations rétrocompatibles au boot** : une base peuplée existe ; tout champ contraint est **backfillé idempotemment** au démarrage → zéro régression sur des données réelles.

---

## 5. L'honnêteté comme argument (ce qui inspire confiance à un client)

🗣️ « On vous dit **exactement** ce qui est protégé et ce qui ne l'est pas. Les **communautés ne sont pas E2EE** — c'est un choix annoncé, parce qu'un canal semi-public à audience illimitée ne peut pas l'être. Un prestataire qui vous promet du chiffrement de bout en bout *partout* vous ment ou ne l'a pas implémenté. **Nous traçons le périmètre exact** — et c'est précisément ce qui rend notre promesse crédible. »

Limites assumées (et pourquoi ce n'est pas un défaut) :
- **Reset admin de passphrase** différé : la seule sémantique crypto-honnête serait « wipe backup → nouvelle identité » ; pas d'escrow, **pour préserver l'admin-proof**.
- **Forward secrecy / rotation de clé** (groupes) : feuille de route — on connaît le sujet et on l'a priorisé.
- **Tests d'intégration sur bases réelles** : à renforcer (les tests unitaires sont déjà nombreux).

→ Un soumissionnaire qui **connaît ses limites et les a priorisées** est plus fiable que celui qui prétend n'en avoir aucune.

---

## 6. Viabilité — gagner un appel d'offre, c'est prouver que ça vit

🗣️ « Un client n'achète pas qu'une techno, il achète un **produit qui dure**. Notre modèle économique est pensé en trois temps : »
1. **Amorçage par levée de fonds** → l'objectif initial est l'**audience**, pas le revenu.
2. **Croissance de l'audience** → rétention, viralité (invitations, partages), qualité d'expérience.
3. **Monétisation publicitaire** : posts sponsorisés (CPM ≈ 8 €/mille, CPC ≈ 0,25 €/clic), tendances sponsorisées, **formules annonceurs** en self-service (Starter 99 €, Pro 490 €, Business 1 900 €/mois) + abonnement **Breezy+** (4,99 €/mois, sans pub, badge, fonctions étendues).

→ On présente un **modèle de revenus**, pas juste une app.

---

## 7. Roadmap priorisée (vision)

| Chantier | Pourquoi | Effort | Priorité |
|---|---|---|---|
| Durcissement XSS/CSP bloquante + audit récurrent | Réduire encore l'exposition de l'access token | 8 j | Haute |
| Anti-automation (rate-limit étendu, bannissement IP type Fail2Ban) | Limiter l'abus | 4 j | Haute |
| Tests d'intégration sur bases réelles en CI | Fiabiliser auth/post/message/notif/média | 10 j | Haute |
| Forward secrecy / rotation de clé (groupes) | Renforcer l'E2EE | — | Moyenne |
| Moteur de recherche full-text | Recherche plus puissante | — | Moyenne |

→ On sait **où on va** et dans quel ordre. C'est un **partenaire**, pas un livreur ponctuel.

---

## 8. Réponses aux attaques du jury (entraînement)

| Attaque probable | Notre riposte |
|---|---|
| « Vos communautés ne sont pas chiffrées, contrairement aux DM. » | « Exact, et c'est **assumé** : l'E2EE strict est impossible avec des viewers illimités. On a tracé le périmètre, là où d'autres promettent l'impossible. » |
| « Un admin peut tout lire, non ? » | « **Non** pour les DM et groupes : serveur aveugle, clé scellée par membre. Démonstration possible. » |
| « Microservices = juste des dossiers ? » | « Non : **base par service, schéma embarqué, tout via la gateway, report-service dédié**. On peut montrer le compose. » |
| « C'est sécurisé ? Prouvez-le. » | « JWT revalidé partout, sessions à rotation, E2EE, en-têtes + rate-limiting, **et un pentest réalisé puis corrigé**. » |
| « Ça tourne vraiment ? » | « **En production, en ligne** : `breezy.philippeluu.fr`. Doc d'API publiée. CI qui bloque la dérive. » |
| « Et la conformité légale ? » | « **RGPD** : purge auto des comptes bannis > 5 ans. Modération avec corbeille restaurable **en cas de poursuites**. NSFW + majorité serveur. » |
| « Les 3 rôles marchent ? » | « Démo live : User signale, Modérateur traite/bannit/certifie, Admin règle/monitore — avec gating mod ≠ admin. » |

---

## 9. Clôture (à dire en dernier)

🗣️ « En résumé : face aux autres Breezy, le nôtre est le seul à combiner une **messagerie réellement admin-proof**, une **sécurité qui ne fait jamais confiance au front**, une **architecture microservices authentique**, et un **produit en production, industrialisé et conforme** — avec les trois rôles et toutes les fonctionnalités primaires. On ne vous demande pas de nous croire sur parole : **c'est en ligne, c'est documenté, et on vous le montre maintenant.** »

---

*Document compagnon du `Scenario_Demo_Soutenance.md` (parcours de démo) et du `Discours_Demo_Soutenance.md` (script détaillé Messages + Modération).*
