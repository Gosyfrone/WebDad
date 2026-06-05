# PROSIT API

Groupe :
STOFFEL Maxime – Scribe
LUU Philippe – Secrétaire
RIVET Alexandre – Animateur
TOUZE Romain - Gestionnaire du temps

# Sommaire

* Sommaire
* 1. Contexte
* 2. Mots inconnus / Notions à maîtriser
* 3. Problématique
* 4. Plan d’action (8h)
* 5. Réalisation


## 1. Contexte

Une entreprise spécialisée dans l’édition de bandes dessinées dispose d’une application web permettant aux utilisateurs de lire des BD en ligne, commander des éditions papier et interagir via un mini-réseau social.

L’application actuelle repose sur une architecture monolithique en 3 couches. Toutes les fonctionnalités sont regroupées dans une seule application comprenant l’interface utilisateur, la logique métier et l’accès aux données.

Cette architecture devient progressivement difficile à maintenir lorsque les fonctionnalités augmentent. Les déploiements sont plus longs, les mises à jour sont plus risquées et il devient difficile d’adapter uniquement certaines parties du système.

Pour répondre à ces limites, l’entreprise souhaite migrer progressivement vers une architecture microservices (MSA). Le mini-réseau social est choisi comme premier domaine de migration car il représente un risque métier plus faible.

Yanis, récemment recruté comme développeur Full Stack, reçoit comme mission de produire un prototype d’API REST destiné à gérer des publications (posts).

Le prototype doit utiliser :

* Node.js pour le serveur
* Express pour créer l’API
* MongoDB pour la persistance
* Mongoose comme ODM JavaScript

Le prototype doit également respecter plusieurs contraintes :

* niveau de maturité REST 2
* déploiement portable
* compatibilité Docker
* préparation au déploiement AWS EKS
* intégration dans une architecture microservices
* prise en charge de la montée en charge via Nginx

## 2. Mots inconnus / Notions à maîtriser

* API REST : Interface permettant à plusieurs applications de communiquer via des requêtes HTTP en manipulant des ressources.
* CRUD : Ensemble des opérations principales sur les données : Create (créer), Read (lire), Update (modifier), Delete (supprimer).
* Node.js : Environnement permettant d’exécuter du JavaScript côté serveur.
* Express : Framework Node.js servant à créer plus facilement des API et applications web.
* MongoDB : Base de données NoSQL qui stocke les données sous forme de documents.
* Mongoose : ODM (Object Document Mapper) permettant de manipuler MongoDB avec des objets JavaScript.
* ODM : Outil qui fait le lien entre le code de l’application et les données stockées dans une base NoSQL.
* Architecture monolithique : Architecture où toutes les fonctionnalités sont regroupées dans une seule application.
* Microservices (MSA) : Architecture où les fonctionnalités sont séparées en plusieurs services indépendants.
* API Gateway : Point d’entrée unique permettant d’accéder aux différents microservices.
* Authentification : Processus qui vérifie l’identité d’un utilisateur.
* Autorisation : Processus qui détermine les droits d’accès d’un utilisateur.
* Scalabilité : Capacité d’un système à supporter une augmentation de charge.
* Disponibilité : Capacité du système à rester accessible.
* Fiabilité : Capacité du système à fonctionner correctement sans erreur.
* Performance : Rapidité de traitement des requêtes.
* Évolutivité : Capacité du système à être amélioré sans devoir être entièrement reconstruit.
* Connectivité : Capacité d’un service à communiquer avec les autres composants.
* Docker : Technologie permettant d’exécuter une application dans un conteneur.
* Conteneurisation : Méthode consistant à emballer une application avec ses dépendances.
* Nginx : Logiciel utilisé comme load balancer ou serveur proxy.
* Load Balancer : Outil qui répartit les requêtes entre plusieurs instances d’un service.
* AWS EKS : Service cloud d’AWS permettant d’orchestrer des conteneurs Kubernetes.

## 3. Problématique

Comment concevoir une API REST de gestion de posts avec Node.js, Express et MongoDB qui respecte le niveau de maturité 2 REST tout en garantissant la portabilité, la scalabilité et l’intégration dans une architecture microservices déployable dans des environnements conteneurisés ?

## 4. Plan d’action (8h)

* Analyse des contraintes techniques et du sujet (0.5h)
* Étude des architectures possibles et choix de l’architecture cible (1h)
* Conception du modèle de données avec MongoDB et Mongoose (1h)
* Développement de l’API REST CRUD (2h)
* Mise en place des bonnes pratiques de déploiement Node.js (1h)
* Conteneurisation avec Docker (1h)
* Validation des propriétés non fonctionnelles (1h30)

## 5. Réalisation

### 5.1 Analyse des contraintes techniques et du sujet (0.5h)

Avant de commencer le développement du prototype, nous avons étudié les contraintes du sujet afin d’identifier les exigences techniques attendues. L’objectif n’était pas uniquement de produire une API fonctionnelle mais de proposer une solution compatible avec la stratégie de migration vers une architecture microservices.

Le premier élément identifié concernait l’indépendance du service. Le sujet précise que l’entreprise souhaite faire évoluer progressivement une application monolithique vers plusieurs services autonomes. Le microservice développé devait donc pouvoir être déployé sans dépendre directement des autres composants.

Nous avons également identifié une contrainte forte concernant la persistance des données. La base MongoDB ne devait pas être intégrée dans le même environnement d’exécution que l’API afin de conserver une séparation claire des responsabilités. Cette approche facilite les mises à jour, permet d’augmenter les performances et améliore la maintenance.
Enfin, le sujet impose une forte portabilité du prototype. Le service devait pouvoir fonctionner localement pendant le développement, être déployé dans un environnement conteneurisé en datacenter puis évoluer vers AWS EKS sans modification majeure du code. Après cette phase d’analyse, nous avons conclu qu’un découpage en microservice indépendant associé à une conteneurisation constituait l’approche la plus adaptée.


### 5.2 Étude des architectures possibles et choix de l’architecture (1h)
Avant d’implémenter le service, nous avons comparé plusieurs architectures logicielles afin de choisir celle qui répondait le mieux au contexte du projet.

La première architecture étudiée était l’architecture monolithique. Dans ce modèle, toutes les fonctionnalités de l’application sont regroupées dans une seule application. L’interface utilisateur, la logique métier et la persistance des données sont exécutées dans le même environnement.
Cette approche présente l’avantage d’être simple à développer et rapide à déployer dans les premières phases d’un projet. En revanche, lorsque l’application devient importante, chaque évolution nécessite de redéployer l’ensemble du système. Cela limite la capacité de montée en charge et augmente le risque d’impact sur les autres fonctionnalités.
Nous avons ensuite étudié l’architecture microservices. Dans cette approche, chaque domaine fonctionnel est isolé dans un service indépendant. Chaque service peut être développé, déployé et mis à jour sans modifier les autres composants.

Cette architecture apporte davantage de flexibilité et répond directement aux objectifs du sujet, notamment la scalabilité, la maintenabilité et la préparation au cloud.

Compte tenu du contexte présenté dans le prosit, nous avons retenu une architecture microservices. 

Le client n’accède jamais directement au microservice. Les requêtes transitent d’abord par un load balancer Nginx chargé de répartir la charge entre plusieurs instances du service lorsque le trafic augmente.
Les requêtes sont ensuite transmises vers une API Gateway qui agit comme point d’entrée unique. Cette Gateway centralise le routage et facilite l’ajout futur de nouveaux microservices.

Avant d’atteindre le service métier, une étape d’authentification et d’autorisation est prévue afin de contrôler les accès aux ressources.
Le microservice Posts contient uniquement la logique métier liée aux publications. La persistance est assurée séparément par MongoDB.


```text
Client
↓
Nginx
↓
API Gateway
↓
Service Authentification
↓
Microservice Posts
↓
MongoDB
```

### 5.3 Conception du modèle de données avec MongoDB et Mongoose (1h)

```js
const mongoose = require("mongoose")

const PostSchema = new mongoose.Schema({
title:{
type:String,
required:true
},
content:{
type:String,
required:true
}
})

module.exports =
mongoose.model("Post",PostSchema)
```

```js
await Post.create({
title:"Premier post",
content:"Contenu"
})
```

### 5.4 Développement de l’API REST (2h)

```bash
npm init -y
npm install express mongoose dotenv
```

```js
mongoose.connect(
process.env.MONGO_URL
)
```

```http
GET /api/v1/posts
GET /api/v1/posts/:id
POST /api/v1/posts
PUT /api/v1/posts/:id
PATCH /api/v1/posts/:id
DELETE /api/v1/posts/:id
```

```json
{
"title":"Mon post",
"content":"Bonjour"
}
```

```json
{
"_id":"123",
"title":"Mon post",
"content":"Bonjour"
}

L’API utilise également les codes de retour HTTP standards afin de communiquer correctement avec les clients.
200 OK
201 Created
204 No Content
400 Bad Request
404 Not Found
500 Internal Server Error

```

### 5.5 Mise en place de bonnes pratiques de déploiement Node.js (1h)

```env
MONGO_URL=
PORT=
```

```dockerfile
FROM node:22-alpine

WORKDIR /app

COPY package*.json ./

RUN npm ci

COPY . .

USER node

EXPOSE 3000

CMD ["node","server.js"]
```

### 5.6 Conteneurisation avec Docker (1h)

```yaml
services:

 api:

  build: .

  ports:
   - "3000:3000"

  environment:
   MONGO_URL: mongodb://mongo:27017/posts

  depends_on:
   - mongo

 mongo:

  image: mongo

volumes:

 mongo-data:
```

### 5.7 Validation des propriétés non fonctionnelles (1.5h)

Une fois le prototype terminé, nous avons vérifié qu’il répondait aux propriétés non fonctionnelles imposées.
La fiabilité est assurée par le découpage des responsabilités entre les différents composants. Une panne de la base n’implique pas l’arrêt complet du système.

Les performances sont améliorées grâce au stockage documentaire de MongoDB et à la possibilité de multiplier les instances du service.
La disponibilité est renforcée par la possibilité d’ajouter plusieurs conteneurs derrière Nginx.

La scalabilité est assurée par l’architecture microservices et la compatibilité avec Kubernetes.

La connectivité repose sur des échanges HTTP standardisés entre composants.
Enfin, l’évolutivité est facilitée puisque de nouveaux services pourront être ajoutés sans modifier le microservice Posts existant.

L’ensemble de ces choix permet d’obtenir un prototype cohérent avec les objectifs du sujet et préparé pour une évolution future vers une infrastructure cloud.

