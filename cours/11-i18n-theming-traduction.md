# 11 — i18n, thème & traduction

> Features : internationalisation 12 langues · préférence de langue par compte · traduction automatique des posts ·
> thème clair/sombre/personnalisé · responsive mobile-first.

## Vue d'ensemble

Le bloc « expérience utilisateur ». Trois choix structurants : **i18n maison sans dépendance**, **préférence de langue
= donnée de compte** (user-service), et **traduction via une route BFF** (clés serveur). Le thème personnalisé est une
**roue chromatique maison** (interdiction d'ajouter une dépendance npm). Tout est front sauf la préférence de langue.

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **user** (PG) | `users.preferred_locale` (nullable, contraint aux locales supportées) | la préférence est une donnée de compte, pas une clé navigateur |
| **BFF Next** | route `/api/translate` (clés serveur), `/api/countries` | les clés des fournisseurs restent côté serveur |
| **Frontend** | `lib/i18n.ts`, `LanguageProvider`, thème, roue chromatique, responsive | tout le reste est front |

## Comment c'est codé

### i18n (12 langues)
- **Maison, zéro dépendance** (`lib/i18n.ts` + dictionnaires statiques + `LanguageProvider` + `useT()`). 12 langues : **FR/EN/ZH/ES/PT/RU/JA/KO/AR/HI/DE/IT** ; **FR** est la référence et le repli final. Chaque chaîne UI passe par `useT()`, clés `namespace.key` ; toute nouvelle clé doit être ajoutée aux 12 locales.
- Les **pages légales** restent rédigées en FR/EN et retombent sur EN pour les autres locales (ne pas présenter une traduction automatique comme juridiquement fiable).

### Préférence de langue (donnée de compte)
- `users.preferred_locale` **nullable** (NULL = utiliser `navigator.language`), transporté par `GET/PATCH /users/me` (pas de nouvelle route). **Le logout ne l'efface pas** : une reconnexion au même compte la restaure ; un autre compte charge la sienne. Le localStorage ne sert qu'aux **visiteurs anonymes**.

### Traduction automatique des posts
- **Via la route BFF `/api/translate`** (LibreTranslate + repli Google), clés **côté serveur** ; post-service ne stocke **aucune** traduction dérivée.
- **Garde cliente prudente** (`shouldAttemptTranslation`) : détection de script + marqueurs latins évitent les faux positifs (un mot étranger isolé, une faute, du texte mixte ne doivent pas auto-traduire un post français). La cible = **toujours la langue du compte**. Le résultat est jeté si `detectedSourceLanguage === targetLanguage` (véto local même-langue). Han (chinois) et Kana (japonais) distincts → ZH↔JA fonctionne. Matrice 12×12 couverte, cache.

### Thème clair / sombre / personnalisé
- **Clair/sombre** (next-themes) + accent de marque (dégradé violet→indigo→cyan). **Surfaces tokenisées** (vars CSS dans `globals.css`, valeurs claires = état actuel, déclinaison sombre) → le sombre vit à un seul endroit.
- **Thème personnalisé** = 3ᵉ dimension (couleurs choisies par l'utilisateur), 3 cibles indépendantes : **background** (repeint l'espace de lecture, surfaces glass), **text**, **primary** (avec `--primary-foreground` auto-contrasté WCAG + les boutons à dégradé de marque). Overrides en styles inline sur `<html>` → gagnent sur la feuille de style quel que soit le mode.
- **Roue chromatique maison** (`ColorWheel`, HSV conic-gradient) : `node_modules` root-owned → `npm i react-colorful` impossible, et une roue in-repo est défense-friendly. Maths de couleur **pures/testées** (`lib/color.ts`). **Persiste les hex sources ET la map de vars CSS pré-calculée** → un petit script `<head>` inline applique la map avant le 1ᵉʳ paint (pas de FOUC, pas de maths dans le script bloquant).

### Responsive
- **Mobile-first, pivot `lg` (1024)** : <lg `MobileHeader` (drawer Sheet) + `MobileTabBar` + `ComposeFab` ; ≥lg sidebar ; ≥xl colonne droite. **Identité cliquable → profil partout** (`UserListItem`), avec aperçu au survol côté web.
- **Fallback avatar = première lettre Unicode réelle** (`initialOf`), globalement (saute espaces/chiffres/`_`/`-`).

## Décisions clés (et alternatives écartées)

- **i18n maison sans dépendance** : `node_modules` root-owned + maîtrise/défense ; FR référence et repli.
- **Préférence de langue = donnée de compte** (user-service) plutôt qu'une clé navigateur globale : suit le compte entre appareils, NULL = navigateur.
- **Traduction via BFF, clés serveur** : ne jamais exposer la clé d'API ; post-service ne stocke aucune traduction (pas de dérivé en base).
- **Garde cliente prudente** : éviter d'auto-traduire à tort un texte déjà dans la langue cible (véto local), tout en laissant l'auto-détection upstream faire autorité.
- **Roue chromatique maison + maths pures testées** : contrainte npm + défense ; persister la map pré-calculée évite le FOUC.
- **Pages légales non auto-traduites** : EN comme repli juridique fiable.

## Points de défense / Q&A anticipées

- *« Pourquoi pas i18next ? »* → `node_modules` root-owned (pas d'ajout npm) + une lib maison est plus simple à défendre et suffit à l'échelle.
- *« La clé de traduction est-elle exposée ? »* → non, elle reste dans le BFF (route serveur).
- *« Comment évite-t-on les fausses traductions ? »* → garde cliente (script + marqueurs) + véto même-langue sur le `detectedSourceLanguage`.

## Limites assumées / perspectives

- Traduction : dépendance à des fournisseurs externes (LibreTranslate/Google) — cache atténue le coût.
- Une utility `bg-*`/`border-*` peut surcharger les classes `@layer components` (piège connu du theming).
