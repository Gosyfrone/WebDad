# Discours de démonstration — ROMAIN

> Parties : **Authentification** · **Recherche / Hashtags / Explorer** · **Profil** · **Visibilité & compte privé** · **Signets** · **Partage de profil**.
> Format : `[CLIC]` = action écran · 🗣️ = ce qu'on dit · 🛡️ = le choix technique défendu.
> Principe transverse à marmarteler : **une donnée = un service**, et **la sécurité ne dépend jamais du front**.

---

## A. AUTHENTIFICATION

### A.1 Connexions sociales (Google / GitHub)
🗣️ « On peut s'inscrire de façon classique, ou via **Google** ou **GitHub**. »
🛡️ « Côté technique, on a un **registre générique de providers** : ajouter une connexion sociale, c'est de la **configuration**, pas du code. Google passe par **OpenID Connect** (on vérifie l'`id_token` côté serveur — signature, émetteur, audience), GitHub par **OAuth2 classique**. Point de sécurité important : **le front ne nous fournit jamais le token** — il relaie seulement le `code`, et on fait l'échange **côté serveur**. On a aussi un anti-CSRF par `state` généré et revérifié. »
🛡️ *(si on demande pourquoi)* « Ne jamais faire confiance à un token fourni par le navigateur : sinon on usurpe un compte avec un simple e-mail concordant. »

### A.2 Création de compte + CGU obligatoires
1. `[CLIC]` Lancer une **inscription**.
2. `[CLIC]` Montrer qu'on **doit cocher l'acceptation des CGU** pour valider. 🗣️ « L'acceptation des CGU est **obligatoire** (on n'oblige plus à scroller les mentions, mais à les **accepter**). »
   🛡️ « Et c'est vrai même pour une inscription **OAuth** : un nouveau compte social n'existe — credentials, profil, JWT — **qu'après** avoir saisi son username, sa date de naissance et **accepté les CGU**. Tant que ce n'est pas fait, on ne stocke qu'un jeton d'onboarding temporaire. Impossible de créer un compte sans consentement. »

### A.3 Âge minimum 13 ans
1. `[CLIC]` Saisir une date de naissance **< 13 ans** → refus.
🗣️ « Âge minimum **13 ans**, sinon inscription impossible. »
🛡️ « À ne pas confondre avec le seuil **18 ans** du contenu sensible. Et surtout : la majorité est **recalculée par le serveur** à partir de la date de naissance, **jamais stockée comme un booléen** — donc le front ne peut pas mentir sur l'âge. »

### A.4 Validation e-mail
🗣️ « À l'inscription, un **e-mail de validation** est envoyé. Tant qu'il n'est pas validé, **la création de contenu est bloquée** (un middleware garde les posts, commentaires, messages). »
🛡️ « Le lien de validation est un **jeton opaque, à usage unique, haché en base, valable 24 h**. Cliquer le lien **prouve la possession de la boîte mail** → ça vaut une connexion : on valide l'e-mail **et** on ouvre la session. C'est notre mail-service dédié qui envoie, isolé du reste pour ne pas mélanger les secrets SMTP avec l'authentification. »

---

## B. RECHERCHE · HASHTAGS · EXPLORER

### B.1 Hashtag cliquable
1. `[CLIC]` Cliquer un **#hashtag** dans un post → résultats filtrés.
🛡️ « Les hashtags sont **extraits côté serveur** à la publication et stockés **dénormalisés sur le post** (en minuscules, sans dièse). Du coup c'est **indexable directement dans Mongo**, sans service de recherche dédié ni base à part. »

### B.2 Recherche
1. `[CLIC]` Rechercher **« breez »**.
2. `[CLIC]` Filtrer par **Publications**, puis par **Comptes**, puis les deux.
🗣️ « Recherche unifiée : comptes par username/nom affiché, et hashtags. »
🛡️ « La recherche de **comptes** vit dans **user-service** (c'est de l'identité), les **hashtags** dans **post-service**. Et un point clé : les **tendances et résultats respectent la barrière de visibilité** — un profil privé ne fuite **pas** via un compteur de hashtag, parce qu'on filtre **avant** de compter. »

### B.3 Page Explorer
1. `[CLIC]` Ouvrir **Explorer**.
🗣️ « Page de découverte : top tendances, suggestions de profils non suivis, publications. »
🛡️ « Le filtrage "posts avec hashtags" se fait **côté serveur**, pas en téléchargeant des pages de feed pour les trier sur le client — ça préserve la barrière de visibilité et la bande passante. »

---

## C. PROFIL

### C.1 Les 3 onglets
1. `[CLIC]` Aller sur **son profil** → onglets **Publications / Réponses / J'aime**.
🛡️ « L'identité d'un utilisateur est **scindée entre deux services** : **user-service** possède le username, le rôle et le **graphe social** ; **profil-service** possède le décoratif — nom affiché, bio, avatar, bannière, nationalité, visibilité. La vue que vous voyez est **agrégée par le front**, sans transaction inter-base. C'est le principe "une donnée = un service". »

### C.2 Épingler un post
1. `[CLIC]` **Épingler** un post en haut du profil.
🛡️ « Un seul épinglé par profil. Détail malin : le profil trie dessus, mais les **feeds renvoient une copie sans l'info d'épinglage** → l'épingle de quelqu'un d'autre ne vient jamais personnaliser le feed global. »

### C.3 Éditer le profil (+ cooldown)
1. `[CLIC]` **Éditer** : changer **bannière** et **photo de profil**, montrer qu'un **GIF animé** est accepté (montrer `@faker`).
2. `[CLIC]` Renseigner **nationalité**, **sexe**, **localisation**.
🗣️ « Le **username** et le **nom affiché** ont un **délai d'attente** entre deux changements — anti-abus. »
🛡️ « La **nationalité** est stockée en **code ISO** (pas un libellé traduit) : robuste à la langue et à l'indisponibilité de l'API catalogue. La date de naissance est **fixée une seule fois**. »

### C.4 Abonnements / Abonnés
1. `[CLIC]` Montrer **Abonnements / Abonnés** + compteurs.
🛡️ « Le suivi est une **table d'arêtes** dans user-service ; une demande à un profil privé est dans une **table séparée** — une demande n'est **pas** une relation. L'acceptation est une **transaction atomique** (on supprime la demande et on crée l'arête en même temps) → jamais "accepté mais pas suivant". »

### C.5 Partage de profil
1. `[CLIC]` **Partager** un profil (DM / natif / copier le lien).
🛡️ « Le partage est **100 % front, sans backend** : un partage, c'est un lien. S'il part en message privé, il voyage **dans le message chiffré E2EE** existant. Savoir **où ne pas ajouter de service** fait partie de l'architecture. »

---

## D. VISIBILITÉ & COMPTE PRIVÉ *(dans Paramètres)*

### D.1 Activité en ligne
🗣️ « On peut afficher ou masquer son **statut en ligne**. »
1. `[CLIC]` Montrer un profil **en ligne**, puis **`@zervixt`** dont l'activité est **désactivée**.
🛡️ « L'activité vit dans **profil-service**, à côté de la visibilité : c'est une **décoration publique avec interrupteur de confidentialité**. Si c'est masqué, le serveur **omet** la dernière connexion ; un profil privé ne la révèle qu'au propriétaire ou aux abonnés acceptés. Et le **refresh de session ne compte pas** comme une nouvelle connexion. »

### D.2 Compte privé
1. `[CLIC]` Passer en **privé** → depuis un autre compte, montrer la **demande d'abonnement** à accepter.
🛡️ « Le basculement public→privé **auto-accepte** les demandes en attente. Et la barrière "qui voit mes posts" est appliquée **côté serveur** par post-service à chaque lecture, jamais par le front. »

---

## E. SIGNETS

1. `[CLIC]` Ouvrir **Signets**, retrouver le post mis en signet plus tôt.
2. `[CLIC]` Montrer les **collections**.
🛡️ « Les signets sont dans **post-service** (un signet référence un post, comme un like). On a un **modèle "burst" façon Instagram** : un clic rapide classe automatiquement dans la dernière collection, sinon on ouvre un sélecteur — **le serveur ne devine jamais** où classer. Et la fenêtre de temps est **côté serveur**, donc cohérente multi-appareils et impossible à tricher en changeant l'horloge du navigateur. »

---

## Annexe — Q&A (Romain)

| Question | Réponse |
|---|---|
| Un client peut-il forger son identité ? | Non : le front ne fournit jamais le token OAuth (échange serveur), et la gateway **strippe** tout en-tête d'identité entrant. |
| Peut-on créer un compte sans accepter les CGU ? | Non, même en OAuth : pas de compte/JWT avant consentement. |
| Les tendances révèlent-elles un compte privé ? | Non : on applique la barrière de visibilité **avant** de compter. |
| Pourquoi le blocage n'est pas dans profil-service ? | C'est une relation entre **deux identités** → graphe social = user-service ; profil ne porte que du décoratif. |
| Pourquoi pas de service de recherche dédié ? | Hashtags dénormalisés + index Mongo suffisent à notre échelle ; un moteur full-text serait du sur-dimensionnement (perspective). |
| Le partage n'a pas de backend ? | Non : un lien ; les 3 voies (DM E2EE / natif / copie) sont 100 % client. |
