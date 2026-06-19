# WebDad — Application distribuée en microservices

[![codecov](https://codecov.io/gh/Gosyfrone/WebDad/graph/badge.svg)](https://codecov.io/gh/Gosyfrone/WebDad)

Application web distribuée construite en architecture microservices dans le cadre du projet FISA INFO A3.
Le système se compose d'un frontend, d'une API Gateway et de quatre services backend indépendants,
chacun avec sa propre base de données.

## Prérequis

- **Docker** et **Docker Compose** v2 (méthode recommandée — toute la stack en une commande)
- **Go** 1.22 ou supérieur (pour lancer un service backend hors Docker)
- **Node.js** 18 ou supérieur (pour lancer le frontend hors Docker)
- **npm** 9 ou supérieur (frontend)
- **Git**

## Démarrage rapide (Docker — recommandé)

Toute la stack (4 services + gateway + frontend + 4 bases de données) démarre en une commande.

1. Cloner le dépôt :

   ```bash
   git clone <url-du-repo>
   cd WebDad
   ```

2. Créer les fichiers d'environnement à partir des modèles :

   ```bash
   make env
   ```

   Cette cible copie chaque `.env.example` en `.env` (racine + chaque service) sans
   écraser un fichier existant. **Édite ensuite les secrets** : au minimum un vrai
   `JWT_SECRET` dans le `.env` racine, et les mots de passe des bases.

3. Démarrer l'environnement :

   ```bash
   make up         # = docker compose up -d (vérifie d'abord que les .env existent)
   ```

   `make up` sert les **images de production figées** : le code est compilé/buildé au
   moment de la construction de l'image. Un changement de code n'apparaît qu'après
   `make build`. Pour développer avec rechargement automatique, voir
   [Mode développement Docker (hot-reload)](#mode-développement-docker-hot-reload).

4. Vérifier que tout répond :

   ```bash
   make ps                                  # tous les conteneurs doivent être "healthy"
   for p in 8080 8081 8082 8083 8084; do curl -s localhost:$p/health; echo; done
   ```

   - Gateway → [http://localhost:8080](http://localhost:8080)
   - Frontend → [http://localhost:3000](http://localhost:3000)

5. Arrêter :

   ```bash
   make down       # arrêt simple
   make reset      # arrêt + suppression des images locales ET des données (volumes)
   ```

> **Comment fonctionne la config `.env` :** le `.env` racine ne contient que les
> variables **transverses** (`JWT_SECRET`, `JWT_EXPIRY`, `NEXT_PUBLIC_API_URL`).
> La config propre à un service vit dans `<service>/.env` et est chargée par
> `docker compose` via `env_file:`. Les hôtes de bases (`DB_HOST`, `MONGO_HOST`)
> sont surchargés automatiquement par le compose pour viser les conteneurs.
> Les variables d'un `env_file` ne sont pas interpolables (`${...}`) dans le
> `docker-compose.yml` : seul le `.env` racine l'est.

## Mode développement Docker (hot-reload)

Même stack que `make up`, mais avec **rechargement automatique** : le code source est monté
dans les conteneurs (bind mount) et un *watcher* recompile/recharge à chaque sauvegarde.
C'est la façon recommandée pour développer sans quitter Docker.

- **Frontend** : `next dev` (HMR React).
- **Services Go** : [`air`](https://github.com/air-verse/air) recompile le binaire à chaque
  modification d'un fichier `.go`.

Prérequis : avoir créé les `.env` (`make env`, voir étape 2 ci-dessus).

```bash
make dev        # build les stages « dev », lance toute la stack en arrière-plan, puis rend la main
```

- Frontend → [http://localhost:3000](http://localhost:3000) — modifie un fichier, la page se recharge.
- Backend → modifie un `.go`, `air` recompile le service concerné.
- La stack tourne **en détaché** (arrière-plan). Pour suivre les logs utiles (front + Go,
  sans le bruit des bases) :

  ```bash
  make dev-logs
  ```

  (ou par service : `make logs-front`, `make logs-post`, `make logs-gateway`, …)

- Pour arrêter/nettoyer la stack de dev :

  ```bash
  make dev-down
  ```

**Sous le capot** : un overlay `docker-compose.dev.yml` se superpose à `docker-compose.yml`
(`make dev` = `docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d`).
Les Dockerfiles Go ont un stage `dev` (avec `air`) placé **avant** le runtime, donc `make up`
et `make build` produisent toujours les images de prod ; le mode dev est purement opt-in.

## Développement hors Docker (service par service)

Pour itérer rapidement sur un seul service sans rebuild d'image :

1. Démarrer uniquement les bases de données :

   ```bash
   make db-only
   ```

2. Installer les dépendances Go (une fois par service) :

   ```bash
   (cd auth-service   && go mod tidy)
   (cd user-service   && go mod tidy)
   (cd profil-service && go mod tidy)
   (cd post-service   && go mod tidy)
   (cd api-gateway    && go mod tidy)
   ```

3. Lancer un service backend (lit son `<service>/.env`, avec `DB_HOST=localhost`) :

   ```bash
   cd <service-name>
   make run     # ou : go run main.go
   ```

4. Lancer le frontend (Next.js) :

   ```bash
   cd frontend
   npm install
   cp .env.local.example .env.local
   npm run dev
   ```

   Le frontend démarre sur [http://localhost:3000](http://localhost:3000).

## Services

| Service        | Port | Stack                          | Base de données | Rôle                                                          |
| -------------- | ---- | ------------------------------ | --------------- | ------------------------------------------------------------- |
| frontend       | 3000 | Next.js 14 + Tailwind + shadcn | —               | Interface utilisateur (User, Moderator, Administrator)        |
| api-gateway    | 8080 | Go 1.22 + Gin                  | —               | Point d'entrée unique, routage, middleware d'authentification |
| auth-service   | 8081 | Go 1.22 + Gin                  | PostgreSQL      | Authentification, JWT, inscription / connexion                |
| user-service   | 8082 | Go 1.22 + Gin                  | PostgreSQL      | Gestion CRUD des utilisateurs et des rôles                    |
| profil-service | 8083 | Go 1.22 + Gin                  | MongoDB         | Profils utilisateurs (bio, avatar, préférences)               |
| post-service   | 8084 | Go 1.22 + Gin                  | MongoDB         | Création, lecture et gestion des posts                        |

## Conventions de commits

Le projet utilise [Conventional Commits](https://www.conventionalcommits.org/) via **commitlint**.

Format : `<type>(<scope>): <description>`

Types autorisés :

| Type       | Usage                                                          |
| ---------- | -------------------------------------------------------------- |
| `feat`     | Nouvelle fonctionnalité                                        |
| `fix`      | Correction de bug                                              |
| `docs`     | Modification de la documentation                               |
| `style`    | Formatage, point-virgules, etc. (pas de changement de logique) |
| `refactor` | Refactorisation sans changement fonctionnel                    |
| `test`     | Ajout ou modification de tests                                 |
| `chore`    | Tâches de maintenance (build, dépendances, etc.)               |
| `perf`     | Amélioration des performances                                  |

Exemples :

```
feat(auth): ajouter la route POST /register
fix(gateway): corriger le proxy vers user-service
docs(readme): documenter les variables d'environnement
chore(deps): mettre à jour express en 4.19
```

## Architecture

L'application suit une architecture en **4 couches**.

### 1. Couche Client
Trois rôles utilisateurs distincts : **User**, **Moderator**, **Administrator**.
Chaque rôle dispose de permissions et d'écrans adaptés.

### 2. Couche Web
**Frontend** Next.js 14 (App Router, TypeScript, Tailwind, shadcn/ui) qui consomme
exclusivement l'API Gateway. Toutes les requêtes passent par un point d'entrée unique.

### 3. Couche Services
- **API Gateway** : point d'entrée HTTP unique, valide le JWT et route vers le bon service backend.
- **Auth Service** : inscription, connexion, génération et validation des JWT.
- **User Service** : CRUD des utilisateurs, gestion des rôles.
- **Profil Service** : informations de profil détaillées des utilisateurs.
- **Post Service** : création, lecture, modération des posts.

Chaque service backend est indépendant : son propre **`go.mod`**, son propre `Dockerfile`
et sa propre base de données. La communication entre services passe par HTTP via l'API Gateway.

### 4. Couche Données
- **PostgreSQL** pour les données relationnelles (Auth, User).
- **MongoDB** pour les données documentaires (Profil, Post).

## Scripts disponibles

À la racine (via le `Makefile`, pilote tout l'environnement Docker) :

```bash
make help      # Liste toutes les commandes
make env       # Crée les .env manquants depuis les .env.example
make up        # Démarre toute la stack en images de prod (docker compose up -d)
make dev       # Démarre en mode dev (détaché) : hot-reload front (next dev) + Go (air)
make dev-logs  # Suit les logs front + Go (sans le bruit des bases de données)
make dev-down  # Arrête la stack de dev
make down      # Arrête les conteneurs
make build     # Rebuild toutes les images (--no-cache)
make ps        # Statut des conteneurs
make logs      # Suit les logs de tous les services
make db-only   # Démarre uniquement les 4 bases de données
make reset     # Arrêt + suppression images locales ET données (volumes)

# Logs d'un service : make logs-auth | logs-user | logs-profil | logs-post | logs-gateway | logs-front
# Shell conteneur   : make sh-auth | sh-user | sh-profil | sh-post | sh-gateway
# CLI base de données: make psql-auth | psql-user | mongo-profil-cli | mongo-post-cli
```

Dans chaque service **backend** (via le `Makefile` du service) :

```bash
make run     # Démarre le service (go run main.go)
make build   # Compile le binaire dans bin/<service-name>
make lint    # Lance golangci-lint
make test    # Lance go test ./...
make tidy    # Met à jour go.mod / go.sum
```

Dans le **frontend** :

```bash
npm run dev    # Démarre Next.js sur le port 3000 (hot reload)
npm run build  # Build de production
npm start      # Sert le build de production sur le port 3000
npm run lint   # Lance ESLint (config Next.js)
```

## Sécurité

- Authentification par **JWT** (JSON Web Tokens).
- Routes protégées par middleware côté API Gateway.
- Variables sensibles stockées dans des fichiers `.env` (jamais commités).
- CORS configuré sur chaque service.

## Équipe

Zaid, Perujan, Candis, Théo
