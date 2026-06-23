# Annexe — Roadmap par phases (Breezy / WebDad)

> Rapprochement des issues GitHub avec la consigne (Fx1–Fx23). Dates = créée → close (issues *Done*) ou créée → aujourd'hui (*In progress*). Généré automatiquement depuis le GitHub Project.

## Synthèse planning

| Phase | Issues (datées) | Période |
|-------|----------------|---------|
| Conception & infrastructure | 35 | 2026-06-01 → 2026-06-19 |
| Fonctionnalités primaires (Fx1-11) | 20 | 2026-06-01 → 2026-06-17 |
| Fonctionnalités secondaires (Fx12-23) | 20 | 2026-06-01 → 2026-06-18 |
| Fonctionnalités additionnelles | 62 | 2026-06-02 → 2026-06-23 |
| Corrections & stabilisation | 45 | 2026-06-08 → 2026-06-22 |
| Tests & qualité | 9 | 2026-06-01 → 2026-06-23 |

## Conception & infrastructure  (35 issues)

| # | Fx | Titre | Statut | Début | Fin |
|---|----|-------|--------|-------|-----|
| #1 |  | Initialiser le mono-repo et la structure des services | Done | 2026-06-01 | 2026-06-01 |
| #2 |  | Squelette du service Auth | Done | 2026-06-01 | 2026-06-02 |
| #3 |  | Squelette du service Post | Done | 2026-06-01 | 2026-06-02 |
| #4 |  | Modèle de données et connexion PostgreSQL | Done | 2026-06-01 | 2026-06-02 |
| #5 |  | Orchestrer l'ensemble avec docker-compose | Done | 2026-06-01 | 2026-06-02 |
| #6 |  | Schéma et connexion MongoDB | Done | 2026-06-01 | 2026-06-02 |
| #8 |  | Squelette du service Profil | Done | 2026-06-01 | 2026-06-04 |
| #9 |  | Provisionner les bases de données | Done | 2026-06-01 | 2026-06-17 |
| #12 |  | Gestion des variables d'environnement et des secrets | Done | 2026-06-01 | 2026-06-01 |
| #13 |  | Schéma et connexion MongoDB | Done | 2026-06-01 | 2026-06-04 |
| #17 |  | Squelette de l'API Gateway | Done | 2026-06-01 | 2026-06-07 |
| #18 |  | Dockeriser le service Profil | Done | 2026-06-01 | 2026-06-02 |
| #20 |  | Dockeriser le service Auth | Done | 2026-06-01 | 2026-06-02 |
| #21 |  | Squelette du service User | Done | 2026-06-01 | 2026-06-03 |
| #22 |  | Dockeriser le service Post | Done | 2026-06-01 | 2026-06-02 |
| #23 |  | Routage vers les micro-services | Done | 2026-06-01 | 2026-06-03 |
| #24 |  | Modèle de données et connexion PostgreSQL | Done | 2026-06-01 | 2026-06-03 |
| #26 |  | Propagation et validation du JWT au niveau gateway | Done | 2026-06-01 | 2026-06-19 |
| #28 |  | Dockeriser l'API Gateway | Done | 2026-06-01 | 2026-06-02 |
| #29 |  | Dockeriser le service User | Done | 2026-06-01 | 2026-06-02 |
| #30 |  | Squelette du frontend | Done | 2026-06-01 | 2026-06-02 |
| #31 |  | Client API et gestion du token | Done | 2026-06-01 | 2026-06-08 |
| #36 |  | Dockeriser le frontend | Done | 2026-06-01 | 2026-06-02 |
| #44 |  | Rate limiting | Done | 2026-06-01 | 2026-06-19 |
| #46 |  | Documentation des API (Swagger/OpenAPI) par service. | Done | 2026-06-01 | 2026-06-10 |
| #48 |  | Pipeline CI/CD | Done | 2026-06-01 | 2026-06-05 |
| #49 |  | Monitoring et logs centralisés | Done | 2026-06-01 | 2026-06-15 |
| #54 |  | Docker hot reload | Done | 2026-06-02 | 2026-06-02 |
| #55 |  | Page de monitoring des dockers | Done | 2026-06-02 | 2026-06-19 |
| #71 |  | Pouvoir stocker des photos ou vidéos en BDD (MinIO ou S3) | Done | 2026-06-03 | 2026-06-09 |
| #97 |  | Regarder pourquoi ça mets longtemps à la 1ère connexion | Done | 2026-06-05 | 2026-06-08 |
| #108 |  | Avoir des Data Seeds | Done | 2026-06-05 | 2026-06-19 |
| #124 |  | Security fondamentaux | Done | 2026-06-08 | 2026-06-19 |
| #185 |  | Limite de stockage par vidéo (5Mb ou moins) et compression | Done | 2026-06-11 | 2026-06-16 |
| #187 |  | Déploiement automatique | Done | 2026-06-11 | 2026-06-12 |

## Fonctionnalités primaires (Fx1-11)  (20 issues)

| # | Fx | Titre | Statut | Début | Fin |
|---|----|-------|--------|-------|-----|
| #7 | Fx1 | Endpoint d'inscription | Done | 2026-06-01 | 2026-06-02 |
| #10 | Fx2 | Connexion et génération du JWT | Done | 2026-06-01 | 2026-06-03 |
| #11 | Fx3 | CRUD des publications | Done | 2026-06-01 | 2026-06-03 |
| #14 | Fx2 | Middleware JWT et autorisation par rôle | Done | 2026-06-01 | 2026-06-16 |
| #15 | Fx4/10 | Consulter et éditer son profil | Done | 2026-06-01 | 2026-06-04 |
| #16 | Fx5 | Fil des publications | Done | 2026-06-01 | 2026-06-07 |
| #25 | Fx1 | CRUD de son propre compte | Done | 2026-06-01 | 2026-06-03 |
| #32 | Fx1/2 | Pages d'authentification | Done | 2026-06-01 | 2026-06-03 |
| #33 | Fx3/5 | Fil et création de publications | Done | 2026-06-01 | 2026-06-02 |
| #34 | Fx4/10/11 | Page profil | Done | 2026-06-01 | 2026-06-02 |
| #35 | Fx-rôles | Affichage conditionnel selon le rôle | Done | 2026-06-01 | 2026-06-17 |
| #37 | Fx6 | Système de likes/réactions | Done | 2026-06-01 | 2026-06-08 |
| #39 | Fx7/8 | Commentaires | Done | 2026-06-01 | 2026-06-07 |
| #41 | Fx9 | Système d'abonnements / follow | Done | 2026-06-01 | 2026-06-08 |
| #42 | Fx2 | Refresh token | Done | 2026-06-01 | 2026-06-03 |
| #43 | Fx5 | Pagination / scroll infini | Done | 2026-06-01 | 2026-06-08 |
| #68 | Fx9 | Voir les followers et les abonnements | Done | 2026-06-03 | 2026-06-07 |
| #82 | Fx4/10 | Consulter et éditer son profil | Done | 2026-06-04 | 2026-06-07 |
| #151 | Fx9 | Mettre à jour nombre d'abonné quand on follow/unfollow un utilisateur | Done | 2026-06-09 | 2026-06-10 |
| #154 | Fx11 | Pages de likes et de réponses sur le profil | Done | 2026-06-09 | 2026-06-16 |

## Fonctionnalités secondaires (Fx12-23)  (20 issues)

| # | Fx | Titre | Statut | Début | Fin |
|---|----|-------|--------|-------|-----|
| #19 | Fx21 | Modération des contenus (rôle Moderator) | Done | 2026-06-01 | 2026-06-16 |
| #27 | Fx21 | Administration des comptes (rôle Admin) | Done | 2026-06-01 | 2026-06-10 |
| #38 | Fx20 | Signalement d'un contenu | Done | 2026-06-01 | 2026-06-16 |
| #40 | Fx18 | Upload d'images | Done | 2026-06-01 | 2026-06-10 |
| #45 | Fx14/15/16 | Notifications temps réel | Done | 2026-06-01 | 2026-06-09 |
| #63 | Fx23 | Bouton de switch dark / clear mode | Done | 2026-06-02 | 2026-06-03 |
| #78 | Fx23 | Bouton dark mode clear mode pour PC | Done | 2026-06-04 | 2026-06-05 |
| #81 | Fx23 | Refonte dark mode | Done | 2026-06-04 | 2026-06-05 |
| #85 | Fx17 | Messages privées et chiffrement | Done | 2026-06-05 | 2026-06-08 |
| #87 | Fx17/18 | Medias (photos, videos, stickers) | Done | 2026-06-05 | 2026-06-10 |
| #91 | Fx20 | Signalement des breeze | Done | 2026-06-05 | 2026-06-18 |
| #92 | Fx19 | Lancement automatique des vidéos au passage dans le feed | Done | 2026-06-05 | 2026-06-16 |
| #115 | Fx22 | Langue EN | Done | 2026-06-08 | 2026-06-08 |
| #129 | Fx14 | Mentions | Done | 2026-06-08 | 2026-06-09 |
| #131 | Fx17 | Front End message privés et groupe | Done | 2026-06-08 | 2026-06-08 |
| #156 | Fx12/13 | Les hashtags + recherche et sidebar , et top tendances | Done | 2026-06-09 | 2026-06-12 |
| #172 | Fx23 | Thème personnalisé | Done | 2026-06-10 | 2026-06-12 |
| #174 | Fx16 | Ajout de notification groupé quand lors d'abonnements | Done | 2026-06-10 | 2026-06-12 |
| #205 | Fx22 | Multi-langue | Done | 2026-06-12 | 2026-06-15 |
| #211 | Fx13 | FIltre par like dans la recherche dans les publications de la loupe | Done | 2026-06-12 | 2026-06-18 |

## Fonctionnalités additionnelles  (77 issues)

| # | Fx | Titre | Statut | Début | Fin |
|---|----|-------|--------|-------|-----|
| #60 |  | Mobile view | Done | 2026-06-02 | 2026-06-02 |
| #72 |  | Refonte graphique UI | Done | 2026-06-03 | 2026-06-04 |
| #74 |  | Date de naissance et interdiction nsfw | Done | 2026-06-03 | 2026-06-19 |
| #75 |  | Post publicitaires | Backlog | 2026-06-03 | — |
| #84 |  | Pages d'erreurs | Done | 2026-06-04 | 2026-06-05 |
| #88 |  | Page d'aides avec logo (besoin d'aide) + didacticiel à la 1ère venue | Backlog | 2026-06-05 | — |
| #89 |  | Pages légales (mentions légales, confidentialités CGU) | Done | 2026-06-05 | 2026-06-08 |
| #90 |  | Hover curseur sur le profil affiche une partie du profil | Done | 2026-06-05 | 2026-06-10 |
| #93 |  | Sondage | Done | 2026-06-05 | 2026-06-15 |
| #94 |  | AI-Grok-Like | Backlog | 2026-06-05 | — |
| #95 |  | Mot de passe oublié | Done | 2026-06-05 | 2026-06-05 |
| #96 |  | Mot de passe oublié | Done | 2026-06-05 | 2026-06-16 |
| #98 |  | Animation sur les réactions + impressions | Done | 2026-06-05 | 2026-06-08 |
| #99 |  | Auto refresh + si on quitte le détail d'un post ça revient au post et pas en hau | Done | 2026-06-05 | 2026-06-15 |
| #100 |  | Posts publicitaires obligatoire impossible à skip | Backlog | 2026-06-05 | — |
| #101 |  | Logo de certifications | Backlog | 2026-06-05 | — |
| #102 |  | Loading screen breezy | In progress | 2026-06-05 | 2026-06-23 |
| #103 |  | Shorts | Backlog | 2026-06-05 | — |
| #104 |  | Traduction automatique des posts | Done | 2026-06-05 | 2026-06-08 |
| #105 |  | MFA | Done | 2026-06-05 | 2026-06-17 |
| #106 |  | Auth with Google and Microsoft | Done | 2026-06-05 | 2026-06-12 |
| #107 |  | Confidentialité / Visilbité des posts | Done | 2026-06-05 | 2026-06-10 |
| #109 |  | Forfait premium (role) | Backlog | 2026-06-05 | — |
| #111 |  | Filtrage de mots interdits + refonte bouton thème | Done | 2026-06-05 | 2026-06-10 |
| #116 |  | Signets | Done | 2026-06-08 | 2026-06-09 |
| #119 |  | Retweet | Done | 2026-06-08 | 2026-06-09 |
| #120 |  | Epingler un Tweet + Animation sur les réactions | Done | 2026-06-08 | 2026-06-08 |
| #123 |  | Ajout d'un bouton Paramètre | Done | 2026-06-08 | 2026-06-08 |
| #127 |  | Localisation du profil et localisation des tweets (si activé) lié à GMaps | Backlog | 2026-06-08 | — |
| #128 |  | Like, repost et emoji commentaire | Done | 2026-06-08 | 2026-06-08 |
| #132 |  | Historique des recherches | Done | 2026-06-08 | 2026-06-10 |
| #139 |  | Bugs | Done | 2026-06-09 | 2026-06-09 |
| #144 |  | Passphrase de récupération des messages | Done | 2026-06-09 | 2026-06-12 |
| #145 |  | Vérification du compte Gmail | Done | 2026-06-09 | 2026-06-16 |
| #146 |  | MFA à activer | Done | 2026-06-09 | 2026-06-18 |
| #153 |  | Son (1 son pour les post, 1 pour DM, 1 pour notif) voir Felipe | Done | 2026-06-09 | 2026-06-19 |
| #155 |  | Vocaux (et pouvoir partager les vocaux dans une autre conv et dans le chat) | In progress | 2026-06-09 | 2026-06-23 |
| #158 |  | Refaire le lecteur video en moderne | Done | 2026-06-09 | 2026-06-16 |
| #159 |  | GIFs | Done | 2026-06-09 | 2026-06-10 |
| #163 |  | Edit les messages privés (il faut l'ancien message au dessus et le nouveau comme | Done | 2026-06-09 | 2026-06-10 |
| #164 |  | Retweet et citer d'une vidéo avec la vidéo + GIFs | Done | 2026-06-09 | 2026-06-10 |
| #165 |  | Vérif delete tweet en BDD pour savoir si on doit restaurer et RGPD aussi | Done | 2026-06-09 | 2026-06-15 |
| #166 |  | Email de vérification du compte | Done | 2026-06-09 | 2026-06-11 |
| #168 |  | Integration  YouTube TikTok snap avec preview | Backlog | 2026-06-10 | — |
| #170 |  | Login par username | Done | 2026-06-10 | 2026-06-11 |
| #177 |  | Faire une deuxième vue quand on clique sur un post | Done | 2026-06-10 | 2026-06-16 |
| #196 |  | Vue visiteur (message privés non disponibles et vue des profil non dispo) | Done | 2026-06-12 | 2026-06-12 |
| #197 |  | Ajouter paramètre Nationnalité dans le profil | Done | 2026-06-12 | 2026-06-15 |
| #207 |  | Admin doit pouvoir créer un compte | Done | 2026-06-12 | 2026-06-12 |
| #213 |  | Story appareils photo, snaps | Backlog | 2026-06-12 | — |
| #214 |  | Rajout de "X est en train d'écrire" dans les messages | Done | 2026-06-12 | 2026-06-15 |
| #215 |  | Rajout de "Ouvert"  + Suppression + en train d'écrire + Ajouter les partages de  | Done | 2026-06-12 | 2026-06-15 |
| #216 |  | Mettre sur le profil "En ligne" ou "Dernière connexion il a X... temps" | Done | 2026-06-12 | 2026-06-16 |
| #217 |  | Suppression message dans les messages privés | Done | 2026-06-12 | 2026-06-15 |
| #218 |  | Bouton stat à supprimer dans les créations de posts | Done | 2026-06-12 | 2026-06-15 |
| #219 |  | Ajouter les partages de posts vers les messages privés | Done | 2026-06-12 | 2026-06-15 |
| #225 |  | Jeux dans les conversations | Backlog | 2026-06-12 | — |
| #226 |  | API live sur les match de cdm dans explorer | Backlog | 2026-06-12 | — |
| #233 |  | Monitoring avec graphes de l'affluence sur les dockers (traffic réseau, CPU) | Backlog | 2026-06-15 | — |
| #234 |  | Pouvoir changer le mdp & email dans les paramètres | Done | 2026-06-15 | 2026-06-15 |
| #237 |  | Régler la confidentialité des messages privés (je peux régler les messages privé | Backlog | 2026-06-15 | — |
| #242 |  | Choisir la vIsibilité des réponses de posts | Done | 2026-06-15 | 2026-06-15 |
| #252 |  | Mettre en place un store | Done | 2026-06-16 | 2026-06-18 |
| #256 |  | Pouvoir bloquer un utilisateur | Done | 2026-06-16 | 2026-06-19 |
| #257 |  | Rendre cliquable la PP dans les messages privés | Backlog | 2026-06-16 | — |
| #258 |  | Pouvoir partager un post via le bouton du post | Done | 2026-06-16 | 2026-06-18 |
| #268 |  | Enlever le bouton stat dans les posts | Done | 2026-06-16 | 2026-06-17 |
| #271 |  | Pouvoir liker une réponse d'un post + notification like reponse + auto accept pr | Done | 2026-06-17 | 2026-06-17 |
| #273 |  | Refonte sidebar | Done | 2026-06-17 | 2026-06-17 |
| #275 |  | Faire fonctionner l'activité quand on ferme l'app | Done | 2026-06-17 | 2026-06-18 |
| #278 |  | Connexion via d'autres services (Facebook, Apple, LinkedIn ?) | Done | 2026-06-17 | 2026-06-18 |
| #286 |  | Améliorer la compression d'image + enregistrer | Done | 2026-06-18 | 2026-06-19 |
| #293 |  | BUG : modifier activité quand on ferme navigateur | Done | 2026-06-18 | 2026-06-18 |
| #296 |  | BUG : Changer activité sur tous les navigateurs | Done | 2026-06-18 | 2026-06-18 |
| #304 |  | Bug : espace blanc dans le à posté | Done | 2026-06-19 | 2026-06-19 |
| #310 |  | API GIF | Done | 2026-06-19 | 2026-06-19 |
| #318 |  | Fix bug UI | Done | 2026-06-19 | 2026-06-21 |

## Corrections & stabilisation  (46 issues)

| # | Fx | Titre | Statut | Début | Fin |
|---|----|-------|--------|-------|-----|
| #130 |  | Bug : header en mobile il y a un espace entre le haut du téléphone et le header  | Done | 2026-06-08 | 2026-06-09 |
| #137 |  | Bug : pop up éditer le profil non optimisé pour tous les téléphones | Done | 2026-06-09 | 2026-06-11 |
| #142 |  | Bug Message épinglé + bug de traduction | Done | 2026-06-09 | 2026-06-09 |
| #147 |  | Bug de traduction : bug par rapport a detect langue bug si là ou il détecte savo | Done | 2026-06-09 | 2026-06-09 |
| #148 |  | header en mobile il y a un espace entre le haut du téléphone et le header (voir  | Done | 2026-06-09 | 2026-06-11 |
| #149 |  | Epingler : n'est pas propre à chaque utilisateur, un autre utilisateur peut voir | Done | 2026-06-09 | 2026-06-09 |
| #162 |  | Bug : Calendrier possible de mettre une date futur sur iphone (vérifier sur tout | Done | 2026-06-09 | 2026-06-11 |
| #176 |  | BUG : Changement dans le profil pas dynamique dans le feed | Done | 2026-06-10 | 2026-06-16 |
| #190 |  | Vérif des mails logo qui charge pas car en localhost, à changer car le site est  | Done | 2026-06-11 | 2026-06-16 |
| #192 |  | Bug : Liaison des services (en production) | Done | 2026-06-11 | 2026-06-16 |
| #193 |  | Bug : notif message pv sur telephone | Done | 2026-06-11 | 2026-06-18 |
| #194 |  | Bug : barre d'envoi des messages en message pv nous déplace hors de l'écran | Done | 2026-06-11 | 2026-06-18 |
| #195 |  | Bugs: ajouter param de visibilité (privé/public) | Done | 2026-06-12 | 2026-06-12 |
| #198 |  | Autoriser les tirets et les points dans le username et bloquer les caractères sp | Done | 2026-06-12 | 2026-06-17 |
| #199 |  | Bug : recherche de compte (loupe transparente) | Done | 2026-06-12 | 2026-06-12 |
| #200 |  | Bug : Compte créé par Google non recherchable | Done | 2026-06-12 | 2026-06-12 |
| #201 |  | Bug : Quand je fais un thème personnalisé, je peux mettre toujours appuyer sur c | Done | 2026-06-12 | 2026-06-18 |
| #202 |  | Sur mobile, apparation de la croix après un click | Done | 2026-06-12 | 2026-06-15 |
| #206 |  | Bug :  le traducteur auto détecte que l'anglais + Multi langues | Done | 2026-06-12 | 2026-06-15 |
| #210 |  | Messagerie inactive depuis profil | Done | 2026-06-12 | 2026-06-18 |
| #212 |  | Header pas bien collé en haut en mobile en visiteur | Done | 2026-06-12 | 2026-06-12 |
| #220 |  | Fix responsive register | Done | 2026-06-12 | 2026-06-15 |
| #221 |  | BUG : Dans les messages Privés le composant pour écrire les messages | Done | 2026-06-12 | 2026-06-18 |
| #222 |  | Enregistrer edition du profil marche pas en online | Done | 2026-06-12 | 2026-06-15 |
| #223 |  | Bugs : Sur le feed le texte "Ca breeze" va au dessus du header | Done | 2026-06-12 | 2026-06-15 |
| #224 |  | Bugs : Dernier Tendance n'a pas de place | Done | 2026-06-12 | 2026-06-15 |
| #235 |  | Bug : Déverouillage des DM après passphrase trop long voir si on peut optimiser | Backlog | 2026-06-15 | — |
| #236 |  | Affichage des drapeaux sur Google Chrome | Done | 2026-06-15 | 2026-06-16 |
| #243 |  | Bug : Attribution role administrateur fonctionne mais délai d'attribution ? | Done | 2026-06-15 | 2026-06-18 |
| #246 |  | BUG : Filtre buggé dans Explorer sur telephone | Done | 2026-06-15 | 2026-06-18 |
| #249 |  | BUG : peut mettre 0 minute dans un sondage + image sur les questions + rectifica | Done | 2026-06-15 | 2026-06-17 |
| #251 |  | BUG : Connexion 404 en dev + overlay pop up dans feed | Done | 2026-06-16 | 2026-06-16 |
| #255 |  | BUG : quand on s'abonne à un compte privé affichage | Done | 2026-06-16 | 2026-06-17 |
| #259 |  | Bug : reco automatique meme après 24h (session non expiré) | Done | 2026-06-16 | 2026-06-17 |
| #260 |  | BUG  : Menu pas accessible certains moments + quand on actualise message ça affi | Done | 2026-06-16 | 2026-06-17 |
| #261 |  | BUG :  Refonte UX/UI pour créer groupe message privé | Done | 2026-06-16 | 2026-06-17 |
| #262 |  | Bug : Je recherche un profil et j'appuie sur fleche de retour dans l'UI et ça me | Done | 2026-06-16 | 2026-06-18 |
| #265 |  | BUG: Plein de bug chromium based à résoudre | Done | 2026-06-16 | 2026-06-18 |
| #266 |  | BUG : Scroll qui n'a rien à faire là | Done | 2026-06-16 | 2026-06-16 |
| #274 |  | BUG : CGU and 5min token | Done | 2026-06-17 | 2026-06-19 |
| #283 |  | Fleche de retour sur les pages admin et modo + supprimer le compte hors champ de | Done | 2026-06-17 | 2026-06-17 |
| #287 |  | Bug : Remove spotify + facebook | Done | 2026-06-18 | 2026-06-19 |
| #288 |  | Bug : notifications travers le header de l'onglet Notifications | Done | 2026-06-18 | 2026-06-19 |
| #305 |  | BUG: "Qui suivre" lors d'une recherche | Done | 2026-06-19 | 2026-06-19 |
| #312 |  | Bug : UI de post bug | Done | 2026-06-19 | 2026-06-19 |
| #320 |  | Fix : Pages légales (mentions, cgu, confidentialité) + pop up accepter les condi | Done | 2026-06-22 | 2026-06-22 |

## Tests & qualité  (12 issues)

| # | Fx | Titre | Statut | Début | Fin |
|---|----|-------|--------|-------|-----|
| #47 |  | Tests automatisés | Done | 2026-06-01 | 2026-06-19 |
| #303 |  | COVERAGE : Tests frontend | Done | 2026-06-19 | 2026-06-23 |
| #321 |  | tests unitaires ~95% mail service | In progress | 2026-06-22 | 2026-06-23 |
| #322 |  | tests unitaires ~95% profil service | In progress | 2026-06-22 | 2026-06-23 |
| #323 |  | tests unitaires ~95% api gateway | Backlog | 2026-06-22 | — |
| #324 |  | tests unitaires ~95% notif service | In review | 2026-06-22 | — |
| #325 |  | tests unitaires ~95% media service | In progress | 2026-06-22 | 2026-06-23 |
| #326 |  | tests unitaires ~95% report service | In progress | 2026-06-22 | 2026-06-23 |
| #327 |  | tests unitaires ~95% post service | In progress | 2026-06-22 | 2026-06-23 |
| #328 |  | tests unitaires ~95% user service | Backlog | 2026-06-22 | — |
| #329 |  | tests unitaires ~95% front end | Done | 2026-06-22 | 2026-06-23 |
| #330 |  | tests unitaires ~95% message service | In progress | 2026-06-22 | 2026-06-23 |

