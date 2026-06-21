# 08 — Médias & GIFs

> Features : media-service + MinIO (stockage opaque) · upload image/vidéo · pièces jointes chiffrées ·
> GIFs GIPHY · lightbox / viewer photo · autoplay vidéo façon Twitter.

## Vue d'ensemble

Le **6ᵉ microservice métier**, et la réponse « conteneurisation » la plus canonique de la soutenance : **media-service +
MinIO**. Le service est **content-agnostic** (octets opaques sous un id aléatoire) et **tout le téléchargement passe par
la gateway**, jamais par des URLs MinIO présignées — c'est le principe cardinal défendu à l'oral.

## Services concernés & pourquoi

| Service | Rôle | Pourquoi ici |
|---|---|---|
| **media** (MinIO) | upload (clair + chiffré), download streamé, proxy GIPHY | stockage transverse opaque, autonome (bucket propre) |
| **post** (MongoDB) | référence `media []MediaRef` (refs relatives `/media/<id>`) | reste agnostique du contenu |
| **message** (MongoDB) | id/nonce/mime dans l'enveloppe chiffrée | E2EE de bout en bout (cf. fiche 06) |
| **api-gateway** | streame le download (Range/seek) | « tout passe par la gateway » |
| **Frontend** | composer média, picker GIF, lightbox, `FeedVideo` | UX |

## Comment c'est codé

### media-service + MinIO
- **Content-agnostic** : octets opaques sous un id aléatoire, `owner_id` dans les user-metadata MinIO → **pas de DB annexe**.
- **Upload clair `POST /media`** (JWT, **sniff magic-byte** du MIME, caps) vs **`POST /media/encrypted`** (blob E2EE opaque, pas de sniff, cap `MEDIA_MAX_BLOB_BYTES`). **`GET /media/:id` est public** (id non devinable), streamé avec **Range/seek + cache immutable + ETag**.
- **Download par la gateway, PAS d'URL présignée** : une URL présignée exposerait MinIO (port public, CORS, host non résolu par le navigateur) et violerait le principe. Le proxy stdlib streame déjà → la vidéo n'est pas un problème mémoire.
- **Cap d'upload uniforme 5 Mo** (image/vidéo/blob), aligné sur le plus petit cap d'un service de référence (photo X = 5 Mo). **Administrateurs non plafonnés** : `Upload`/`UploadEncrypted` bypassent les checks de taille si `claims.Role == "admin"`. **Limite appliquée serveur** (§6) ; le front ne fait qu'une garde UX (`exceedsMediaLimit`, admin-aware) à la sélection.
- **En-têtes de durcissement** : `X-Content-Type-Options: nosniff` + `Content-Disposition: attachment` (blobs) au download.

### Câblage posts / messages
- **Posts** : post-service porte `media []MediaRef` (refs **relatives** `/media/<id>`), `content` optionnel si ≥1 média.
- **Messages** : fichier chiffré côté client → blob opaque uploadé → id/nonce/mime dans l'**enveloppe chiffrée** → serveur aveugle, **zéro changement de schéma**.

### GIFs GIPHY
- **Proxy GIPHY porté par media-service via la gateway** (`/gifs/search`) : la clé `GIPHY_API_KEY` reste **côté serveur**. Rattaché au media-service (pas une route BFF) pour garder Swagger + routage API au même endroit que les médias, sans nouveau microservice ni DB.
- **Le GIF choisi est capturé en MinIO** (`POST /gifs/capture`) et stocké `/media/<id>` — **jamais l'URL giphy externe**. *Pourquoi* : (1) règle « everything through the gateway » ; (2) le hotlink `media.giphy.com` renvoyait des 403/404 non déterministes en prod ; (3) la prod sert `CSP default-src 'self'` → en enforce, tout `img-src` externe serait bloqué. La capture rend le média same-origin et immutable-cachable.
- **Anti-SSRF par allowlist d'hôtes** (`*.giphy.com` connus), pas de fetch d'URL arbitraire ; taille bornée, type revalidé magic-bytes, owner = utilisateur capturant.

### Viewer & autoplay
- **Deux comportements** : message privé → `MediaLightbox` (le `src` est l'objectURL déjà déchiffré → download reste E2EE) ; post → `PostPhotoModal` deux panneaux façon X (média + commentaires). Les deux overlays via `createPortal(document.body)` (un ancêtre `backdrop-blur`/`transform` confinerait un `position:fixed`).
- **`FeedVideo`** : autoplay muted+loop+playsInline quand ≥60 % visible (IntersectionObserver), pause hors écran. **Lazy-mount** via un 2ᵉ observer (`rootMargin:400px`) → une vidéo loin dans un feed infini n'est ni téléchargée ni bouclée. Front-only, zéro coût serveur ; autoplay impose muted (politique navigateur).

## Décisions clés (et alternatives écartées)

- **Download par la gateway, jamais d'URL MinIO présignée** : règle cardinale ; présigné exposerait MinIO (port, CORS, host).
- **media-service content-agnostic + MinIO** plutôt que blob en DB / sur disque : réponse microservices canonique, coche « conteneurisation », bien meilleure à la défense.
- **GIF capturé en MinIO**, pas hotlinké : same-origin (compatible CSP `self`), immunisé aux 403/404 du CDN giphy. *Écarté* : proxy à la volée (dépendance runtime giphy à chaque vue, cache délicat). *Résiduel assumé* : les vignettes du picker restent hotlinkées (UI éphémère).
- **Cap uniforme 5 Mo, admins non plafonnés** plutôt qu'un cap admin élevé configurable : un admin de confiance n'a pas besoin d'un nombre arbitraire ; le streaming MinIO encaisse les gros fichiers.
- **Pas de génération de variantes à la capture** : un GIF animé re-encodé perdrait l'animation.
- **Lazy-mount vidéo** : clé pour ne pas charger N vidéos d'un feed infini.

## Points de défense / Q&A anticipées

- *« Pourquoi pas d'URL présignée MinIO (plus simple) ? »* → ça exposerait MinIO et violerait « tout par la gateway » ; le proxy streame déjà la vidéo.
- *« Le GIF tiers ne casse pas la CSP ? »* → non, il est capturé en MinIO → same-origin.
- *« La vidéo en mémoire ? »* → non, streaming Range + lazy-mount.

## Limites assumées / perspectives

- Vignettes du picker GIF encore hotlinkées (à proxifier si la CSP passe en enforce).
- Ops : poser `GIPHY_API_KEY` en prod + purger les entrées mortes du `fallbackGifCatalog`.
- Nettoyage des blobs média orphelins (suppression de message) — TODO.
