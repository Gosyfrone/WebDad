# PROSIT JWT / Docker

Groupe :

STOFFEL Maxime - Scribe

LUU Philippe - Secrétaire

RIVET Alexandre - Animateur

TOUZE Romain - Gestionnaire du temps

# Sommaire

Sommaire  
1. Contexte  
2. Mots inconnus / Notions à maîtriser  
3. Problématique  
4. Plan d’action (8h)  
5. Réalisation

## 1. Contexte

Yanis a réalisé un premier prototype MERN de gestion de posts. Ce prototype repose sur une architecture en microservices avec un client React, un service de gestion des posts en Node.js, une base MongoDB pour les posts et une couche Nginx placée devant les services. Cette première version permet déjà de comprendre le découpage global de l’application, mais elle ne répond pas encore aux besoins de sécurité attendus dans une application exposant des API.

Maia demande donc à Yanis d’intégrer la sécurité dans la prochaine itération. L’objectif principal est de faire en sorte que seuls les utilisateurs authentifiés et autorisés puissent accéder aux API. Il ne suffit plus de créer des routes fonctionnelles pour gérer les posts. Il faut maintenant vérifier l’identité de l’utilisateur, contrôler ses droits et éviter que n’importe quel client puisse appeler directement les endpoints sensibles.

Yanis propose au départ une solution où le client envoie l’identifiant et le mot de passe dans la requête, soit dans les en-têtes, soit dans le corps. La requête est ensuite interceptée par Nginx, utilisé comme API Gateway. Dans son idée, Nginx vérifierait les identifiants dans une base avant de router la requête vers le service demandé. Cette idée montre qu’il a compris le rôle central d’une API Gateway comme point d’entrée unique, mais elle présente plusieurs limites importantes.

La première limite est liée à la transmission répétée des identifiants. Envoyer un mot de passe à chaque requête augmente les risques, même si l’application utilise HTTPS. Le mot de passe doit être utilisé uniquement lors de la phase de connexion, puis remplacé par un mécanisme plus adapté, comme un jeton d’accès. La deuxième limite concerne le rôle de l’API Gateway. Une Gateway peut contrôler l’accès, vérifier la présence d’un jeton, bloquer des requêtes ou router vers les services, mais elle ne doit pas devenir un service de gestion des comptes utilisateurs. La gestion des utilisateurs, des mots de passe et des rôles doit être isolée dans un service d’authentification ou confiée à un service tiers spécialisé.

Dans cette nouvelle version, l’architecture doit donc évoluer. Le client React envoie ses identifiants uniquement au service d’authentification. Si les identifiants sont corrects, ce service génère un jeton JWT. Le client conserve ce jeton et l’utilise ensuite dans l’en-tête `Authorization` pour appeler les API protégées. Nginx joue le rôle d’API Gateway en centralisant les accès et en redirigeant les requêtes vers les bons services. Le service de posts vérifie le jeton avant d’exécuter les actions demandées.

En parallèle, Yanis rencontre aussi un problème de déploiement. Le prototype contient maintenant plusieurs composants : le client React, Nginx, le service de posts, le service d’authentification, la base MongoDB des posts et la base utilisée pour les comptes utilisateurs. Démarrer tous ces éléments manuellement dans plusieurs terminaux devient difficile à maintenir. Il faut donc mettre en place une solution multi-conteneurs avec Docker Compose afin de lancer l’ensemble de l’application avec une seule commande et de faciliter les tests locaux.

## 2. Mots inconnus / Notions à maîtriser

API Gateway : Point d’entrée unique placé entre le client et les services backend. Elle reçoit les requêtes, applique certaines règles de sécurité, puis les redirige vers le bon microservice. Dans ce prosit, Nginx est utilisé comme Gateway pour éviter que le client accède directement aux services internes.

Nginx : Serveur web qui peut aussi servir de reverse proxy et de load balancer. Dans cette architecture, il reçoit les requêtes HTTP du client React et les transmet vers le service d’authentification ou le service de posts selon l’URL appelée.

L’authentification : Correspond à la vérification de l’identité d’un utilisateur. Par exemple, lorsqu’un utilisateur envoie son email et son mot de passe, le système vérifie si ces informations correspondent à un compte existant.

L’autorisation : Correspond au contrôle des droits d’un utilisateur une fois qu’il est authentifié. Un utilisateur peut être connecté, mais ne pas avoir le droit de supprimer un post ou d’accéder à une route réservée à un administrateur.

Un service tiers d’authentification : Service externe ou séparé qui gère la connexion, les comptes utilisateurs, les mots de passe, les rôles et parfois les jetons. Des solutions comme Auth0, Keycloak ou Firebase Authentication permettent d’éviter de développer soi-même toute la partie sensible de l’authentification. Dans un projet pédagogique, on peut aussi créer un microservice d’authentification interne afin de comprendre les mécanismes.

JWT : signifie JSON Web Token. C’est un jeton signé contenant des informations sur l’utilisateur. Il est composé de trois parties : un header, un payload et une signature. Le header indique l’algorithme utilisé, le payload contient les informations utiles comme l’identifiant et le rôle de l’utilisateur, et la signature permet de vérifier que le jeton n’a pas été modifié.

Access token : est un jeton utilisé pour accéder à une ressource protégée. Il a généralement une durée de vie courte. Cela limite les risques si le jeton est volé.

Refresh token : Jeton avec une durée de vie plus longue. Il sert à demander un nouveau jeton d’accès sans obliger l’utilisateur à se reconnecter avec son mot de passe. Il doit être stocké avec beaucoup de précaution.

Header `Authorization` : Une en-tête HTTP utilisé pour transmettre les informations d’authentification. Avec un JWT, on utilise généralement le format `Authorization: Bearer <token>`.

Bearer : Signifie que le porteur du jeton peut accéder à la ressource si le jeton est valide. Cela implique qu’un JWT doit être protégé côté client, car toute personne qui possède le jeton peut potentiellement l’utiliser.

Hachage de mot de passe : Consiste à transformer un mot de passe en une valeur non réversible. On ne stocke jamais un mot de passe en clair dans une base de données. En Node.js, la bibliothèque `bcrypt` est souvent utilisée pour hacher et vérifier les mots de passe.

Variable d’environnement : est une valeur de configuration externe au code source. Elle permet de stocker des informations comme l’URL de MongoDB, le port du serveur ou la clé secrète utilisée pour signer les JWT. Cette approche évite d’écrire des informations sensibles directement dans le code.

Docker : Technologie de conteneurisation. Elle permet d’emballer une application avec ses dépendances dans un conteneur afin de l’exécuter de manière reproductible sur différentes machines.

Docker Compose : Outil permettant de définir et de lancer plusieurs conteneurs avec un fichier `docker-compose.yml`. Il est adapté au contexte du prosit car l’application est composée de plusieurs services qui doivent fonctionner ensemble.

Un reverse proxy : Reçoit les requêtes du client et les transmet à un serveur interne. Le client ne connaît pas directement l’adresse des services internes, ce qui améliore l’organisation et la sécurité de l’architecture.

Un middleware Express : Fonction exécutée pendant le traitement d’une requête. Dans ce prosit, un middleware peut vérifier le JWT avant de laisser passer la requête vers une route protégée.

Le contrôle d’accès côté backend : Consiste à refuser une requête si l’utilisateur n’est pas authentifié ou s’il ne possède pas les droits nécessaires. Ce contrôle est indispensable car le frontend peut être modifié ou contourné par un utilisateur.

Le contrôle d’accès côté frontend : Consiste à adapter l’interface selon l’état de connexion et le rôle de l’utilisateur. Par exemple, le bouton de suppression d’un post peut être masqué pour un utilisateur non administrateur. Ce contrôle améliore l’expérience utilisateur mais ne remplace jamais la sécurité backend.

## 3. Problématique

Comment sécuriser une application MERN organisée en microservices en utilisant une API Gateway Nginx, un service d’authentification basé sur JWT et un déploiement multi-conteneurs Docker, afin de garantir que seuls les utilisateurs authentifiés et autorisés puissent accéder aux API backend ?

## 4. Plan d’action (8h)

1. Analyse des contraintes techniques et des limites de la première idée (0.5h)
2. Etude du fonctionnement d'un service authentification et des JWT (1h)
3. Conception de l'architecture cible (1h)
4. Réalisation du service d'authentification (1.5h)
5. Vérification du JWT et contrôle d’accès dans le service de posts (1h)
6. Proposition de contrôle d’accès côté frontend React (0.75h)
7. Configuration de Nginx comme API Gateway (1h)
8. Mise en place du déploiement multi-conteneurs avec Docker Compose (1.25h)

## 5. Réalisation

### 5.1 Analyse des contraintes techniques et des limites de la première idée (0.5h)

La première étape consiste à analyser la proposition initiale de Yanis. Son idée est de faire passer les identifiants dans chaque requête et de laisser Nginx vérifier ces informations dans une base avant de router vers les services. Cette solution n’est pas adaptée car elle mélange plusieurs responsabilités.

L’identifiant et le mot de passe ne doivent pas être envoyés à chaque appel API. Le mot de passe est une donnée très sensible. Même avec HTTPS, il est préférable de limiter son utilisation à la phase de connexion. Une fois l’utilisateur connecté, le système doit lui fournir un jeton temporaire permettant d’accéder aux ressources protégées.

Le deuxième problème est le rôle donné à Nginx. Nginx est très utile comme point d’entrée, reverse proxy, load balancer ou API Gateway simple. En revanche, il ne doit pas devenir le composant qui stocke les comptes utilisateurs et vérifie les mots de passe. La gestion des comptes correspond à une logique métier de sécurité qui doit être placée dans un service dédié ou dans un fournisseur d’identité externe.

La correction de l’architecture consiste donc à séparer l’authentification du routage. Nginx reçoit les requêtes et les redirige. Le service d’authentification vérifie les identifiants et génère les jetons. Le service de posts vérifie le jeton avant d’autoriser l’accès aux routes protégées.

```text
Client React
    ↓
Nginx / API Gateway
    ↓
Service Authentification  →  MongoDB Auth
    ↓
Génération du JWT
    ↓
Client React utilise le JWT pour appeler le service Posts
    ↓
Service Posts  →  MongoDB Posts
```

Cette séparation rend l’application plus propre et plus proche d’une architecture microservices. Chaque composant possède une responsabilité claire.

### 5.2 Étude du fonctionnement d’un service d’authentification et des JWT (1h)

Un service d’authentification sert à gérer les comptes utilisateurs et la connexion. Lorsqu’un utilisateur s’inscrit, le service reçoit son email, son mot de passe et éventuellement son rôle. Avant l’enregistrement, le mot de passe est haché avec `bcrypt`. La base ne contient donc pas le mot de passe en clair, mais une empreinte sécurisée.

Lorsqu’un utilisateur se connecte, le service recherche le compte associé à l’email. Il compare ensuite le mot de passe envoyé avec le hash stocké en base. Si la comparaison est correcte, le service génère un JWT signé avec une clé secrète. Ce jeton contient uniquement les informations nécessaires, comme l’identifiant de l’utilisateur et son rôle.

Un JWT n’est pas chiffré par défaut. Son contenu peut être lu si quelqu’un récupère le jeton. Il ne faut donc pas y placer de mot de passe ou d’information confidentielle. La sécurité repose surtout sur la signature, qui permet au backend de vérifier que le jeton n’a pas été modifié.

Un exemple de payload JWT peut être le suivant :

```json
{
  "sub": "65f1a8b3d9",
  "email": "yanis@example.com",
  "role": "admin",
  "iat": 1710000000,
  "exp": 1710003600
}
```

La propriété `sub` représente l’identifiant du sujet, donc l’utilisateur. La propriété `role` permet de faire du contrôle d’accès. La propriété `exp` indique la date d’expiration du jeton. Cette expiration est importante car un jeton ne doit pas rester valable indéfiniment.

Après connexion, le client envoie le jeton dans chaque requête protégée avec l’en-tête suivant :

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

Le backend récupère cet en-tête, extrait le jeton, vérifie sa signature et contrôle son expiration. Si le jeton est valide, la requête continue. Sinon, le serveur renvoie une réponse `401 Unauthorized`.

### 5.3 Conception de l’architecture cible (1h)

L’architecture retenue sépare les responsabilités entre plusieurs composants. Le client React est responsable de l’interface utilisateur. Il permet à l’utilisateur de se connecter, stocke le jeton reçu et l’envoie lorsqu’il appelle une API protégée.

Nginx est placé devant les services. Il sert d’API Gateway simple. Son rôle est de recevoir toutes les requêtes entrantes puis de les transmettre vers le bon service. Par exemple, une requête commençant par `/auth` est envoyée au service d’authentification, alors qu’une requête commençant par `/posts` est envoyée au service de posts.

Le service d’authentification gère les utilisateurs. Il expose une route d’inscription et une route de connexion. Il utilise une base MongoDB dédiée aux comptes utilisateurs. Cette séparation permet de ne pas mélanger les données de sécurité avec les données métier des posts.

Le service de posts gère les publications. Il ne connaît pas les mots de passe des utilisateurs. Il se contente de vérifier le JWT reçu et d’appliquer les règles d’autorisation. Par exemple, un utilisateur connecté peut lire les posts, mais seul un administrateur peut supprimer un post.

```text
Client React
   |
   |  /auth/register ou /auth/login
   v
Nginx Gateway ----------------> Auth Service ----> MongoDB Auth
   |
   |  /posts avec Authorization: Bearer <token>
   v
Posts Service ----------------> MongoDB Posts
```

Cette architecture répond au sujet car elle utilise l’API Gateway pour centraliser les accès sans lui donner la responsabilité de stocker les identifiants. Elle permet aussi d’ajouter plus tard un service tiers d’authentification comme Keycloak ou Auth0 sans modifier complètement le service de posts.

### 5.4 Réalisation du service d’authentification (1.5h)

Le service d’authentification est développé avec Node.js et Express. Il utilise MongoDB pour stocker les utilisateurs, Mongoose pour modéliser les données, bcrypt pour hacher les mots de passe et jsonwebtoken pour générer les JWT.

L’installation des dépendances peut se faire avec la commande suivante :

```bash
npm init -y
npm install express mongoose bcrypt jsonwebtoken dotenv
```

Le modèle utilisateur contient l’email, le mot de passe haché et le rôle. Le rôle peut servir ensuite à contrôler l’accès à certaines routes.

```js
const mongoose = require("mongoose")

const UserSchema = new mongoose.Schema({
  email: {
    type: String,
    required: true,
    unique: true
  },
  password: {
    type: String,
    required: true
  },
  role: {
    type: String,
    enum: ["user", "admin"],
    default: "user"
  }
})

module.exports = mongoose.model("User", UserSchema)
```

La route d’inscription hache le mot de passe avant de l’enregistrer. Cette étape est indispensable car une fuite de base de données ne doit pas exposer directement les mots de passe des utilisateurs.

```js
const express = require("express")
const bcrypt = require("bcrypt")
const jwt = require("jsonwebtoken")
const User = require("./models/User")

const router = express.Router()

router.post("/register", async (req, res) => {
  const { email, password } = req.body

  const existingUser = await User.findOne({ email })
  if (existingUser) {
    return res.status(409).json({ message: "Utilisateur déjà existant" })
  }

  const hashedPassword = await bcrypt.hash(password, 10)

  const user = await User.create({
    email,
    password: hashedPassword
  })

  res.status(201).json({
    id: user._id,
    email: user.email,
    role: user.role
  })
})
```

La route de connexion vérifie le mot de passe et génère un JWT si les identifiants sont valides.

```js
router.post("/login", async (req, res) => {
  const { email, password } = req.body

  const user = await User.findOne({ email })
  if (!user) {
    return res.status(401).json({ message: "Identifiants invalides" })
  }

  const isPasswordValid = await bcrypt.compare(password, user.password)
  if (!isPasswordValid) {
    return res.status(401).json({ message: "Identifiants invalides" })
  }

  const token = jwt.sign(
    {
      sub: user._id.toString(),
      email: user.email,
      role: user.role
    },
    process.env.JWT_SECRET,
    { expiresIn: "1h" }
  )

  res.status(200).json({ token })
})

module.exports = router
```

La clé `JWT_SECRET` ne doit pas être écrite directement dans le code. Elle doit être fournie avec une variable d’environnement. Cela évite de publier une clé secrète dans un dépôt Git.

```env
PORT=4000
MONGO_URL=mongodb://auth-db:27017/auth
JWT_SECRET=une_cle_longue_et_secrete
```

### 5.5 Vérification du JWT et contrôle d’accès dans le service de posts (1h)

Une fois le service d’authentification en place, le service de posts doit refuser les requêtes non authentifiées. Pour cela, on ajoute un middleware Express chargé de vérifier la présence et la validité du JWT.

Le middleware lit l’en-tête `Authorization`. S’il est absent, la réponse doit être `401 Unauthorized`. S’il est présent mais invalide ou expiré, la réponse doit également être `401 Unauthorized`. Si le jeton est valide, le middleware ajoute les informations de l’utilisateur dans `req.user` pour que les routes suivantes puissent les utiliser.

```js
const jwt = require("jsonwebtoken")

function authenticateToken(req, res, next) {
  const authHeader = req.headers.authorization

  if (!authHeader || !authHeader.startsWith("Bearer ")) {
    return res.status(401).json({ message: "Token manquant" })
  }

  const token = authHeader.split(" ")[1]

  try {
    const payload = jwt.verify(token, process.env.JWT_SECRET)
    req.user = payload
    next()
  } catch (error) {
    return res.status(401).json({ message: "Token invalide ou expiré" })
  }
}

module.exports = authenticateToken
```

Le contrôle d’accès ne s’arrête pas à l’authentification. Il faut aussi vérifier les droits. Par exemple, la lecture des posts peut être ouverte à tous les utilisateurs connectés, alors que la suppression d’un post peut être réservée aux administrateurs.

```js
function requireRole(role) {
  return (req, res, next) => {
    if (!req.user || req.user.role !== role) {
      return res.status(403).json({ message: "Accès interdit" })
    }

    next()
  }
}

module.exports = requireRole
```

L’utilisation dans les routes du service de posts peut ensuite être faite de manière simple.

```js
const express = require("express")
const authenticateToken = require("./middlewares/authenticateToken")
const requireRole = require("./middlewares/requireRole")

const router = express.Router()

router.get("/", authenticateToken, async (req, res) => {
  res.status(200).json({ message: "Liste des posts" })
})

router.post("/", authenticateToken, async (req, res) => {
  res.status(201).json({ message: "Post créé" })
})

router.delete("/:id", authenticateToken, requireRole("admin"), async (req, res) => {
  res.status(204).send()
})

module.exports = router
```

Dans cette logique, `401 Unauthorized` signifie que l’utilisateur n’est pas correctement authentifié. `403 Forbidden` signifie que l’utilisateur est authentifié mais qu’il n’a pas les droits nécessaires. Cette différence est importante pour produire une API claire et maintenable.

### 5.6 Proposition de contrôle d’accès côté frontend React (0.75h)

Le frontend doit aussi être adapté à l’authentification. Lorsqu’un utilisateur se connecte, React reçoit le JWT depuis le service d’authentification. Le client peut ensuite conserver ce jeton et l’ajouter aux requêtes vers les routes protégées.

Dans une version simple de prototype, le jeton peut être stocké en mémoire ou dans le `localStorage`. Le stockage en mémoire est plus prudent contre certaines attaques XSS, mais il disparaît au rechargement de la page. Le `localStorage` est simple à utiliser, mais il est plus exposé si une faille XSS existe. Pour un projet plus avancé, on peut utiliser un cookie `HttpOnly`, `Secure` et `SameSite` afin de limiter l’accès du JavaScript au jeton.

Un exemple simple avec `fetch` peut être le suivant :

```js
const token = localStorage.getItem("token")

const response = await fetch("/posts", {
  method: "GET",
  headers: {
    Authorization: `Bearer ${token}`
  }
})
```

Le frontend peut aussi adapter l’affichage selon le rôle de l’utilisateur. Par exemple, un utilisateur avec le rôle `admin` voit le bouton de suppression, alors qu’un utilisateur classique ne le voit pas.

```jsx
function PostActions({ user, post }) {
  return (
    <div>
      <button>Voir</button>
      {user?.role === "admin" && (
        <button>Supprimer</button>
      )}
    </div>
  )
}
```

Ce contrôle côté frontend améliore l’expérience utilisateur, mais il ne doit jamais être considéré comme suffisant. Un utilisateur peut modifier le code côté navigateur ou appeler directement l’API avec un outil comme Postman. Les contrôles importants doivent donc toujours être réalisés côté backend.

### 5.7 Configuration de Nginx comme API Gateway (1h)

Nginx est utilisé comme point d’entrée unique de l’application. Le client n’a pas besoin de connaître les ports internes des services. Il appelle seulement Nginx, et Nginx route les requêtes selon leur chemin.

Une configuration simple peut router `/auth` vers le service d’authentification et `/posts` vers le service de posts.

```nginx
events {}

http {
  server {
    listen 80;

    location /auth/ {
      proxy_pass http://auth-service:4000/;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /posts/ {
      proxy_pass http://posts-service:3000/;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header Authorization $http_authorization;
    }
  }
}
```

La ligne `proxy_set_header Authorization $http_authorization;` est importante car elle transmet le jeton JWT reçu par Nginx au service de posts. Sans cela, le service backend pourrait ne pas recevoir l’en-tête nécessaire à l’authentification.

Dans cette configuration, Nginx ne vérifie pas directement le mot de passe. Il applique surtout le routage et transmet les informations nécessaires aux services. Cette solution respecte mieux la séparation des responsabilités.

### 5.8 Mise en place du déploiement multi-conteneurs avec Docker Compose (1.25h)

Le dernier besoin du prosit concerne le lancement de l’application. Avec plusieurs services, il n’est plus pratique de démarrer manuellement chaque composant dans un terminal séparé. Docker Compose permet de décrire toute l’architecture dans un seul fichier et de démarrer l’ensemble avec une commande.

Le fichier `docker-compose.yml` définit Nginx, le service d’authentification, le service de posts, la base MongoDB utilisée pour les comptes et la base MongoDB utilisée pour les posts.

```yaml
services:
  gateway:
    image: nginx:alpine
    ports:
      - "8080:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - auth-service
      - posts-service

  auth-service:
    build: ./auth-service
    environment:
      PORT: 4000
      MONGO_URL: mongodb://auth-db:27017/auth
      JWT_SECRET: ${JWT_SECRET}
    depends_on:
      - auth-db

  posts-service:
    build: ./posts-service
    environment:
      PORT: 3000
      MONGO_URL: mongodb://posts-db:27017/posts
      JWT_SECRET: ${JWT_SECRET}
    depends_on:
      - posts-db

  auth-db:
    image: mongo:7
    volumes:
      - auth-data:/data/db

  posts-db:
    image: mongo:7
    volumes:
      - posts-data:/data/db

volumes:
  auth-data:
  posts-data:
```

La variable `JWT_SECRET` doit être identique entre le service d’authentification et le service de posts si le service de posts vérifie directement la signature du JWT. Dans une architecture plus avancée, on pourrait utiliser une paire de clés publique et privée. Le service d’authentification signerait le jeton avec une clé privée, et les services backend vérifieraient le jeton avec une clé publique.

Chaque service possède son propre `Dockerfile`. Pour un service Node.js, un Dockerfile simple peut être écrit ainsi :

```dockerfile
FROM node:22-alpine

WORKDIR /app

COPY package*.json ./
RUN npm ci --omit=dev

COPY . .

USER node

EXPOSE 3000

CMD ["node", "server.js"]
```

Une fois le fichier Compose créé, l’application peut être lancée avec la commande suivante :

```bash
docker compose up --build
```

Le client ou l’outil de test peut ensuite appeler l’API Gateway sur le port `8080`. Par exemple, l’inscription passe par `/auth/register`, la connexion par `/auth/login`, puis la consultation des posts par `/posts` avec le jeton dans l’en-tête `Authorization`.

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"yanis@example.com","password":"password"}'
```

```bash
curl http://localhost:8080/posts \
  -H "Authorization: Bearer <token>"
```

Cette mise en place répond au problème de lancement rencontré par Yanis. Tous les composants peuvent être démarrés ensemble, les dépendances entre services sont déclarées et les bases de données conservent leurs données grâce aux volumes Docker.

### Conclusion

La solution proposée répond aux objectifs d’apprentissage du prosit. Le fonctionnement d’un service d’authentification est expliqué à travers la séparation entre la connexion, le hachage du mot de passe et la génération du JWT. La sécurité de l’application est gérée avec des jetons, qui remplacent l’envoi répété des identifiants.

Le contrôle d’accès backend est assuré par des middlewares Express. Le premier vérifie que le JWT est présent et valide. Le deuxième peut vérifier le rôle de l’utilisateur pour protéger certaines routes. Cette approche permet de bloquer les accès non autorisés directement au niveau de l’API.

Le contrôle d’accès frontend est aussi prévu. React peut adapter l’affichage selon l’état de connexion et le rôle de l’utilisateur. Cependant, cette partie reste complémentaire car la vraie sécurité doit être appliquée côté serveur.

L’API Gateway Nginx est utilisée correctement. Elle ne stocke pas les identifiants et ne remplace pas le service d’authentification. Elle sert de point d’entrée unique, transmet les requêtes vers les bons services et conserve une architecture claire.

Enfin, Docker Compose résout la difficulté de démarrage manuel. L’application devient plus simple à lancer, plus reproductible et plus adaptée à une architecture microservices. Cette base pourra ensuite évoluer vers une orchestration plus complète avec Kubernetes si le projet doit être déployé à plus grande échelle.
