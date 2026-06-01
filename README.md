# WebDad — Application distribuée en microservices

Application web distribuée construite en architecture microservices dans le cadre du projet FISA INFO A3.
Le système se compose d'un frontend, d'une API Gateway et de quatre services backend indépendants,
chacun avec sa propre base de données.

## Prérequis

- **Go** 1.22 ou supérieur (services backend)
- **Node.js** 18 ou supérieur (frontend)
- **npm** 9 ou supérieur (frontend)
- **Docker** et **Docker Compose** (pour l'exécution conteneurisée — phase ultérieure)
- **Git**

## Getting started

1. Cloner le dépôt :

   ```bash
   git clone <url-du-repo>
   cd WebDad
   ```

2. Installer les dépendances de chaque service backend (Go) :

   ```bash
   (cd auth-service   && go mod tidy)
   (cd user-service   && go mod tidy)
   (cd profil-service && go mod tidy)
   (cd post-service   && go mod tidy)
   (cd api-gateway    && go mod tidy)
   ```

3. Copier les fichiers d'environnement :

   ```bash
   cp auth-service/.env.example    auth-service/.env
   cp user-service/.env.example    user-service/.env
   cp profil-service/.env.example  profil-service/.env
   cp post-service/.env.example    post-service/.env
   cp api-gateway/.env.example     api-gateway/.env
   ```

4. Lancer un service backend en mode développement :

   ```bash
   cd <service-name>
   make run     # ou : go run main.go
   ```

5. Lancer le frontend (Next.js) :

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

Chaque service backend est indépendant : son propre **`go.mod`**, son propre `Dockerfile` (à venir)
et sa propre base de données. La communication entre services passe par HTTP via l'API Gateway.

### 4. Couche Données
- **PostgreSQL** pour les données relationnelles (Auth, User).
- **MongoDB** pour les données documentaires (Profil, Post).

## Scripts disponibles

Dans chaque service **backend** (via le `Makefile`) :

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
