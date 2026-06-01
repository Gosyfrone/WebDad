# WebDad — Application distribuée en microservices

Application web distribuée construite en architecture microservices dans le cadre du projet FISA INFO A3.
Le système se compose d'un frontend, d'une API Gateway et de quatre services backend indépendants,
chacun avec sa propre base de données.

## Prérequis

- **Node.js** 18 ou supérieur
- **npm** 9 ou supérieur
- **Docker** et **Docker Compose** (pour l'exécution conteneurisée — phase ultérieure)
- **Git**

## Getting started

1. Cloner le dépôt :

   ```bash
   git clone <url-du-repo>
   cd WebDad
   ```

2. Installer les dépendances de chaque service :

   ```bash
   cd auth-service    && npm install && cd ..
   cd user-service    && npm install && cd ..
   cd profil-service  && npm install && cd ..
   cd post-service    && npm install && cd ..
   cd api-gateway     && npm install && cd ..
   cd frontend        && npm install && cd ..
   ```

3. Copier les fichiers d'environnement :

   ```bash
   cp auth-service/.env.example    auth-service/.env
   cp user-service/.env.example    user-service/.env
   cp profil-service/.env.example  profil-service/.env
   cp post-service/.env.example    post-service/.env
   cp api-gateway/.env.example     api-gateway/.env
   cp frontend/.env.example        frontend/.env
   ```

4. Lancer un service en mode développement :

   ```bash
   cd <service-name>
   npm run dev
   ```

## Services

| Service          | Port | Base de données | Rôle                                                          |
| ---------------- | ---- | --------------- | ------------------------------------------------------------- |
| API Gateway      | 3000 | —               | Point d'entrée unique, routage, middleware d'authentification |
| Auth Service     | 3001 | PostgreSQL      | Authentification, JWT, inscription / connexion                |
| User Service     | 3002 | PostgreSQL      | Gestion CRUD des utilisateurs et des rôles                    |
| Profil Service   | 3003 | MongoDB         | Profils utilisateurs (bio, avatar, préférences)               |
| Post Service     | 3004 | MongoDB         | Création, lecture et gestion des posts                        |
| Frontend         | 5173 | —               | Interface React (User, Moderator, Administrator)              |

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
**Frontend** React qui consomme exclusivement l'API Gateway.
Toutes les requêtes passent par un point d'entrée unique.

### 3. Couche Services
- **API Gateway** : point d'entrée HTTP unique, valide le JWT et route vers le bon service backend.
- **Auth Service** : inscription, connexion, génération et validation des JWT.
- **User Service** : CRUD des utilisateurs, gestion des rôles.
- **Profil Service** : informations de profil détaillées des utilisateurs.
- **Post Service** : création, lecture, modération des posts.

Chaque service est indépendant : son propre `package.json`, son propre `Dockerfile` (à venir) et sa
propre base de données. La communication entre services passe par HTTP via l'API Gateway.

### 4. Couche Données
- **PostgreSQL** pour les données relationnelles (Auth, User).
- **MongoDB** pour les données documentaires (Profil, Post).

## Scripts disponibles

Dans chaque service backend :

```bash
npm start      # Démarre le service en mode production
npm run dev    # Démarre le service avec nodemon (hot reload)
npm run lint   # Lance ESLint
npm test       # Lance la suite de tests
```

## Sécurité

- Authentification par **JWT** (JSON Web Tokens).
- Routes protégées par middleware côté API Gateway.
- Variables sensibles stockées dans des fichiers `.env` (jamais commités).
- CORS configuré sur chaque service.

## Équipe

Zaid, Perujan, Candis, Théo
