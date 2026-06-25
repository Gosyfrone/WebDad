# Discours de démonstration — Soutenance Breezy

> Script oral pour la démo live. Partie : **Messagerie chiffrée (E2EE)** + **Modération / Administration**.
> Objectif : montrer ET défendre les choix d'architecture. Phrases prêtes à dire ; les *[actions]* indiquent ce que tu fais à l'écran.

---

## 0. Transition d'ouverture (10 s)

« Je vais vous présenter deux blocs qui montrent à la fois notre exigence de **sécurité** et le fait que **les trois rôles sont fonctionnels** : d'abord la **messagerie chiffrée de bout en bout**, puis le **centre de modération et d'administration**. »

---

## 1. MESSAGERIE — Le serveur est aveugle

### 1.1 La passphrase et tout le dispositif de sécurité

*[Tu ouvres Messages. L'écran de passphrase / PassphraseGate apparaît, la vue est floutée tant que l'identité n'est pas déverrouillée.]*

« Avant même d'afficher quoi que ce soit, l'application me demande ma **passphrase**. C'est le cœur de notre dispositif de sécurité, et je vais expliquer pourquoi.

Notre messagerie est **chiffrée de bout en bout**. Concrètement, **tout le chiffrement se passe dans le navigateur** : le serveur de messagerie ne voit jamais que des **enveloppes** — du `ciphertext` et un `nonce` — et des **métadonnées** : qui parle à qui, quand, lu ou non lu. **Jamais de texte en clair.** On dit que le serveur est *aveugle*, et c'est volontaire : même un administrateur de la plateforme **ne peut pas lire un message privé**. C'est *admin-proof*.

Techniquement : chaque utilisateur a une **paire de clés X25519**, dont la clé privée vit **dans le navigateur**, dans IndexedDB. Chaque conversation a une **clé de contenu symétrique** qui est **scellée individuellement pour chaque membre** — une *sealed box*. Le serveur transporte ces enveloppes mais **n'a aucune clé pour les ouvrir**. Le chiffrement symétrique des messages utilise **XChaCha20-Poly1305**. »

*[Tu tapes la passphrase, la vue se déverrouille.]*

« La passphrase, elle, sert à un problème concret : la clé privée était **par appareil**, donc illisible si je me connecte depuis un deuxième téléphone. On a donc ajouté une **sauvegarde de clé chiffrée par passphrase**, en *zero-knowledge*.

Ma clé privée est **scellée côté client** sous une clé dérivée de ma passphrase par **Argon2id** — la fonction de référence pour le hachage de mots de passe — avec des paramètres au **minimum recommandé par l'OWASP** (19 Mio de mémoire). On a calibré ça volontairement : des paramètres plus lourds figeaient le téléphone 15 à 25 secondes, donc on a fait un **compromis mobile assumé**, avec auto-upgrade des anciennes sauvegardes. Et ce calcul tourne dans un **Web Worker** pour ne pas bloquer l'interface.

Le point clé : le serveur stocke cette sauvegarde comme un **blob opaque** — sel, nonce, clé privée emballée. **Il ne voit ni ma passphrase, ni ma clé privée.** Donc je peux retrouver mes conversations sur un nouvel appareil, sans jamais casser le modèle *admin-proof*.

Et l'honnêteté de ce choix va jusqu'au bout : si j'oublie ma passphrase, la sauvegarde est **irrécupérable**. C'est le prix du zero-knowledge — on a refusé tout *escrow* (séquestre de clé) qui aurait redonné au serveur un moyen de lire. »

> **Si on te demande « et si l'utilisateur oublie sa passphrase ? »** → « C'est assumé : pas de récupération possible, sinon le serveur détiendrait un moyen de déchiffrer et on perdrait la garantie admin-proof. Un appareil sans clé ni sauvegarde régénère une nouvelle identité ; l'historique antérieur reste illisible sur cet appareil — comme Signal. »

### 1.2 Signaler un message (E2EE intact)

*[Tu ouvres une conversation, clic droit / menu sur un message → Signaler.]*

« Je signale ce message. Et là il y a une vraie question d'architecture : **comment modérer un contenu que le serveur ne sait pas lire ?**

Notre réponse, c'est le modèle de Signal ou WhatsApp : **c'est le destinataire qui divulgue volontairement** la copie en clair au moment du signalement. Le serveur **ne déchiffre jamais de lui-même** — il reçoit le contenu uniquement parce que **moi, récepteur légitime, je choisis de le joindre** à mon signalement. L'E2EE reste intact, et la modération reste possible. »

*[Tu confirmes le signalement.]*

« On reviendra sur ce signalement tout à l'heure, côté modérateur. »

### 1.3 Groupes et communautés — un modèle hybride assumé

*[Tu montres un groupe, puis tu invites quelqu'un.]*

« Pour les **groupes**, on garde le même niveau de sécurité que les DM : ils sont **admin-proof**. Quand j'invite quelqu'un *[action]*, c'est **un membre qui détient déjà la clé** qui l'**emballe pour la clé publique** du nouvel arrivant. Le serveur ne fait que transporter. N'importe quel membre peut inviter, la gestion reste à l'owner, et c'est plafonné à 32 membres. »

*[Tu passes aux communautés — annuaire public / Découvrir les communautés.]*

« Les **communautés**, c'est un **compromis assumé**, et c'est important de l'expliquer parce que c'est un choix réfléchi, pas un oubli.

Une communauté, c'est un espace **semi-public** type *canal Telegram* : nom en clair, **annuaire public**, on peut la rejoindre librement *[tu rejoins / montres l'annuaire]*, avec un nombre de **lecteurs illimité**. Or l'**E2EE strict est mathématiquement impossible** avec des viewers inconnus et illimités : il faudrait qu'un détenteur de clé soit **en ligne** pour sceller la clé à chaque nouvel arrivant. Donc pour les communautés, et **uniquement** pour elles, **la clé est côté serveur** : elles sont *admin-readable* **par conception**.

C'est une décision **annoncée et bornée** : DM et groupes = *admin-proof* ; communautés = lisibles, parce que le modèle social l'exige. On préfère un compromis honnête et documenté à une fausse promesse de sécurité. »

> **Phrase de défense forte :** « On n'a pas voulu vendre du chiffrement de bout en bout là où il est techniquement impossible. Le périmètre exact de ce qui est protégé est clair, c'est ça qui fait la valeur sécurité. »

---

## 2. MODÉRATION & ADMINISTRATION — Les 3 rôles

*[Tu te connectes en modérateur / tu ouvres le centre de modération.]*

« Je passe maintenant côté **modérateur**. D'abord un mot d'architecture : un signalement porte sur un contenu qui appartient à un **autre** service — un post, un message, un profil. Il **n'appartient donc à aucun** d'eux. On a créé un **report-service dédié**, ce qui renforce la cohérence microservices : chaque service garde une responsabilité unique. »

### 2.1 Les tickets et les filtres

*[Tu montres la liste des tickets et les filtres.]*

« Voici les **tickets de signalement**. Premier choix de conception : on **agrège**. Si dix personnes signalent le même contenu, on ne crée pas dix tickets — un **index unique partiel** dans Mongo empile les signalements dans **un seul ticket parent**, avec le compteur et les motifs dénormalisés.

Je peux filtrer *[tu démontres]* par **statut** — ouvert, réouvert —, par **date**, et par **volume minimum de signalements**. Ce dernier filtre est utile pour prioriser : on traite d'abord ce qui est massivement signalé.

Le cycle de vie d'un ticket est pensé pour résister au spam :
- **Un seul signalement par personne et par contenu** — impossible de gonfler un compteur tout seul.
- Un ticket fermé se **rouvre automatiquement** au-delà d'un seuil, mais comme chaque personne ne compte qu'une fois, il faut **deux personnes distinctes** pour le rouvrir. C'est notre garde-fou anti-spam.
- Et un statut terminal **"approuvé"** : si je juge le contenu conforme, le ticket est verrouillé et **tout nouveau signalement est refusé** — ça évite le harcèlement par signalement répété. »

*[Tu ouvres le ticket de message signalé tout à l'heure.]*

« Et voici le signalement de message de tout à l'heure. Le serveur n'a pas déchiffré : c'est la **copie divulguée par le destinataire** que je consulte. »

### 2.2 Supprimer un tweet → corbeille → restaurer

*[Tu supprimes un post depuis la modération.]*

« Je retire ce post. Important : ce n'est **pas une suppression définitive**, c'est un **soft-delete**. Le post part dans une **corbeille — "Tweets supprimés"**. »

*[Tu ouvres l'onglet "Tweets supprimés", tu montres le post, tu le restaures.]*

« Et c'est un vrai choix produit, pas une facilité technique. On garde le contenu pour une raison concrète : **en cas de poursuites judiciaires ou de contestation**, on doit pouvoir **restaurer** un contenu retiré à tort, ou le **conserver comme preuve**. Je peux le **restaurer** d'un clic *[action]*, ou le **purger définitivement** si nécessaire. La réversibilité est volontaire.

À noter : on a **réutilisé** un mécanisme de soft-delete qui existait déjà, plutôt que de le réimplémenter — moins de code, plus de cohérence. »

### 2.3 Marquer comme NSFW

*[Tu marques un post comme contenu sensible.]*

« Je peux aussi marquer un contenu comme **sensible — NSFW**. Le post reste **public**, mais il est **flouté** à l'affichage pour ceux qui ne doivent pas le voir.

Et la règle de sécurité ici est serveur, jamais front : la **majorité est calculée par le serveur** à partir de la date de naissance — jamais stockée, recalculée à la lecture. Donc **le front ne peut pas se faire passer pour majeur**. Le floutage est un filtre d'affichage façon X, mais **la politique d'âge, elle, est autoritative côté serveur**. »

### 2.4 Bannir un utilisateur

*[Tu bannis l'utilisateur depuis le TicketDetail / encart "profil de risque".]*

« Sur la cible du ticket, j'ai un encart de **profil de risque** et l'action **Bannir**. Je bannis ce compte.

Techniquement, le bannissement passe `is_active` à faux et **révoque immédiatement toutes les sessions** de l'utilisateur — comme un changement de mot de passe. »

*[Si possible : tu te déconnectes / ouvres une fenêtre privée et tentes de te reconnecter avec le compte banni.]*

« Et voilà ce que voit l'utilisateur banni à la reconnexion : le serveur renvoie un **403** et le message **"Compte désactivé, veuillez contacter le support de Breezy."**. Son profil affiche aussi qu'il a été banni et n'est plus accessible.

Point sur les rôles : un **modérateur ne peut pas bannir un administrateur** ni un autre modérateur — il y a un *gating* explicite. La hiérarchie des rôles est respectée côté serveur. »

### 2.5 Attribuer une certification → les badges

*[Tu passes en administrateur si besoin, tu ouvres un profil et attribues une certification.]*

« Dernier point : l'**attribution de certifications**, réservée aux modérateurs et administrateurs. C'est une route dédiée avec un **enum fermé** — on ne peut attribuer que des valeurs valides, pas injecter n'importe quoi.

On a **deux certifications attribuables** *[tu montres le menu]* :
- **Politique** — un sceau **doré**, pour les personnalités politiques.
- **Personnalité publique / influenceur** — un sceau **bleu**, façon vérification X.

*[Tu attribues, puis tu montres le badge sur le profil + le hover qui affiche le libellé.]*

Et il y a un **troisième badge, le badge Staff** — le logo Breezy — qui, lui, est **automatique** : dès qu'un compte est modérateur ou administrateur, il l'affiche. On ne peut pas l'attribuer à la main, ce qui évite l'usurpation d'un statut officiel : seul un vrai membre du staff porte le badge staff.

Ces badges apparaissent partout où l'identité est affichée — profil, posts, survol — parce que dans notre app **l'identité est toujours cliquable et vérifiable**. »

---

## 3. Conclusion de la partie (15 s)

« Donc pour résumer cette partie : une messagerie **réellement chiffrée de bout en bout** avec un serveur aveugle et une sauvegarde de clé zero-knowledge ; un compromis **assumé et borné** sur les communautés ; et un centre de modération qui fait vivre **les trois rôles** — l'utilisateur signale, le modérateur traite et retire, l'administrateur règle les seuils, bannit et certifie — le tout en restant **compatible avec le chiffrement** et avec une **réversibilité** pensée pour le juridique. »

---

## Annexe — Réponses aux questions pièges

| Question | Réponse courte |
|---|---|
| Un admin peut-il lire un DM ? | **Non.** Clé scellée par membre, serveur aveugle. (Communautés = oui, par conception annoncée.) |
| L'état "lu" ne casse pas l'E2EE ? | Non : ce sont des **timestamps / métadonnées**, jamais le contenu. |
| Passphrase oubliée ? | Irrécupérable — prix du zero-knowledge, pas d'escrow pour garder l'admin-proof. |
| Modérer un message chiffré ? | Le **destinataire divulgue** volontairement ; le serveur ne déchiffre jamais, pose juste un tombstone par id. |
| Un seul compte peut-il rouvrir un ticket en spammant ? | Non : 1 signalement / personne / contenu + seuil = **2 personnes distinctes**. |
| Les 3 rôles sont-ils fonctionnels ? | User signale ; Modérateur traite tickets/warns/retraits, bannit un user ; Admin règle le seuil, transfère les bugs, bannit, certifie — avec gating mod ≠ admin. |
| Pourquoi un report-service séparé ? | Un signalement n'appartient à aucun service métier → service propre, cohérence microservices. |
| Le NSFW protège-t-il vraiment les mineurs ? | La majorité est **calculée serveur** depuis la date de naissance, jamais sur la foi du front. |
| Suppression d'un post = vraie suppression ? | Soft-delete réversible (corbeille) pour le juridique ; purge définitive possible. Pour les **messages**, la suppression "pour tous" efface réellement le ciphertext (le serveur était déjà aveugle → tombstone honnête). |
