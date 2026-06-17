# PROSIT SÉCURITÉ

Groupe :

STOFFEL Maxime - Animateur

LUU Philippe - Gestionnaire du temps

RIVET Alexandre - Scribe

TOUZE Romain - Secrétaire

# Sommaire

* Sommaire
* 1. Contexte
* 2. Mots inconnus / Notions à maîtriser
* 3. Problématique
* 4. Plan d’action (8h)
* 5. Réalisation
* 6. Conclusion

## 1. Contexte

Yanis a déjà mis en place une partie de la sécurité côté back-end avec la génération et la vérification de jetons JWT. Cependant, une application sécurisée ne dépend pas uniquement du serveur. Le front-end doit aussi gérer correctement l’état de connexion, protéger certaines pages et éviter d’exposer inutilement des informations sensibles.

Dans ce prosit, Yanis se demande comment protéger ses pages React et comment garantir qu’un utilisateur accède seulement à ses propres informations. Il doit donc trouver une manière propre de stocker l’état d’authentification, de partager cet état entre plusieurs composants et d’envoyer le JWT au back-end uniquement lorsque c’est nécessaire.

React Context devient alors une solution intéressante. Il permet de partager des données globales, comme l’utilisateur connecté ou le jeton, sans devoir les transmettre manuellement avec des props dans toute l’application. Le Provider est l’élément central de cette solution, car il rend le contexte disponible pour les composants enfants. Sans Provider, un hook personnalisé comme `useAuth` ne peut pas accéder aux données d’authentification.

Yanis doit aussi réfléchir au stockage des JWT. Le `localStorage` est simple à utiliser mais peut être risqué en cas d’attaque XSS. Les cookies `HttpOnly`, `Secure` et `SameSite` sont souvent plus adaptés pour protéger certains jetons, mais ils demandent une configuration plus sérieuse côté serveur. Enfin, comme le front-end et le back-end peuvent être sur des origines différentes, il faut aussi gérer les problèmes de CORS ou passer par Nginx pour centraliser le routage.

## 2. Mots inconnus / Notions à maîtriser

React Context : mécanisme de React permettant de partager des données globales entre plusieurs composants sans passer par les props à chaque niveau.

Provider : composant qui fournit les données d’un contexte à tous ses composants enfants. Dans ce prosit, `AuthProvider` fournit l’utilisateur, le token et les fonctions de connexion ou déconnexion.

Access store : espace centralisé permettant de conserver les informations liées à l’accès de l’utilisateur, comme son état de connexion, son token et ses actions de connexion ou de déconnexion. Dans notre cas, ce rôle est assuré par le contexte d’authentification fourni par `AuthProvider`.

Hook personnalisé : fonction React réutilisable commençant généralement par `use`. Par exemple, `useAuth` permet d’accéder facilement au contexte d’authentification.

JWT : JSON Web Token. C’est un jeton signé contenant des informations sur l’utilisateur, comme son identifiant ou son rôle. Il permet au back-end de vérifier l’identité de l’utilisateur.

Access token : jeton à durée de vie courte utilisé pour accéder aux routes protégées.

Refresh token : jeton à durée de vie plus longue servant à renouveler un access token sans redemander le mot de passe.

LocalStorage : stockage du navigateur accessible en JavaScript. Il est pratique mais exposé si une faille XSS existe.

Cookie HttpOnly : cookie non lisible par JavaScript, ce qui limite le risque de vol de jeton par script malveillant.

XSS : attaque consistant à injecter du JavaScript malveillant dans une page web.

CSRF : attaque où un site malveillant force le navigateur d’un utilisateur connecté à envoyer une requête non voulue.

CORS : mécanisme de sécurité du navigateur qui contrôle si un front-end peut appeler une API située sur une autre origine.

Origine : combinaison du protocole, du domaine et du port. Par exemple, `http://localhost:5173` et `http://localhost:3000` sont deux origines différentes.

Route protégée : page accessible seulement si l’utilisateur est connecté ou possède les droits nécessaires.

Authorization Bearer : format d’en-tête HTTP utilisé pour envoyer un JWT à une API, sous la forme `Authorization: Bearer <token>`.

HTTPS : version sécurisée de HTTP qui chiffre les échanges entre le navigateur et le serveur.

API Gateway : point d’entrée unique qui reçoit les requêtes du front-end et les redirige vers les bons services.

Nginx : serveur pouvant jouer le rôle de reverse proxy, d’API Gateway et de load balancer.

## 3. Problématique

Comment sécuriser une application front-end React avec React Context, un Provider, des hooks personnalisés et des JWT, afin de protéger les pages, gérer correctement l’état de connexion, garantir l’accès aux données propres à chaque utilisateur et éviter les problèmes de CORS avec le back-end ?

## 4. Plan d’action (8h)

1. Analyse des besoins de sécurité côté front-end et des risques liés aux JWT (0.75h)

2. Étude de React Context, du Provider et des hooks personnalisés (1h)

3. Conception de l’architecture d’authentification front-end (1h)

4. Mise en place du `AuthProvider` et du hook `useAuth` (1.5h)

5. Création des routes protégées et contrôle d’accès aux pages (1h)

6. Gestion du stockage des jetons et sécurisation des appels API (1h)

7. Configuration des échanges front-end / back-end et gestion du CORS avec Express ou Nginx (1h)


## 5. Réalisation

### 5.1 Analyse des besoins de sécurité côté front-end et des risques liés aux JWT (0.75h)

La première étape consiste à comprendre ce que le front-end peut réellement sécuriser. Le back-end reste la partie la plus importante pour la sécurité, car un utilisateur peut toujours modifier le JavaScript dans son navigateur ou appeler directement l’API avec un outil comme Postman. Le front-end sert surtout à protéger l’expérience utilisateur, à éviter l’accès visuel à certaines pages et à envoyer correctement le JWT au serveur.

Pour garantir qu’un utilisateur accède uniquement à ses propres informations, le front-end ne doit pas décider seul. Il envoie le jeton au back-end, puis le serveur utilise l’identifiant contenu dans le JWT pour récupérer les bonnes données. Par exemple, une page `/profile` peut être protégée côté React, mais c’est le serveur qui doit vérifier que les données retournées correspondent bien à l’utilisateur connecté.

Le stockage du jeton est aussi un point important. Dans un prototype, on peut utiliser le `localStorage` car il est simple à comprendre. Cependant, cette solution est sensible aux attaques XSS. Une solution plus sécurisée consiste à garder l’access token en mémoire et à utiliser un refresh token dans un cookie `HttpOnly`, `Secure` et `SameSite`.

```text
Utilisateur connecté
   ↓
JWT reçu depuis le service d’authentification
   ↓
Front-end stocke temporairement le jeton
   ↓
Requête API avec Authorization: Bearer <token>
   ↓
Back-end vérifie le jeton et filtre les données
```

Cette analyse montre que le front-end améliore la sécurité et l’expérience utilisateur, mais qu’il ne remplace jamais les vérifications côté serveur.

### 5.2 Étude de React Context, du Provider et des hooks personnalisés (1h)

React Context permet de partager un état global dans l’application. Dans le cas de l’authentification, plusieurs composants ont besoin des mêmes informations. La page de connexion doit enregistrer le token, la barre de navigation doit savoir si l’utilisateur est connecté, et les routes protégées doivent refuser l’accès si aucun utilisateur n’est présent.

Sans Context, il faudrait transmettre l’utilisateur et le token avec des props dans beaucoup de composants. Cela rendrait le code plus lourd. Avec un Provider, on peut encapsuler l’application et rendre ces informations disponibles partout où elles sont nécessaires.

```jsx
import { createContext, useContext, useState } from "react";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [token, setToken] = useState(null);
  const [user, setUser] = useState(null);

  function login(newToken, userData) {
    setToken(newToken);
    setUser(userData);
  }

  function logout() {
    setToken(null);
    setUser(null);
  }

  return (
    <AuthContext.Provider value={{ token, user, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);

  if (!context) {
    throw new Error("useAuth doit être utilisé dans un AuthProvider");
  }

  return context;
}
```
Ce code crée un contexte d’authentification qui permet de partager les informations de connexion dans toute l’application.

Il stocke deux données principales : le token et l’user.
La fonction login sert à enregistrer le token et les informations de l’utilisateur quand il se connecte.

La fonction logout remet ces valeurs à null, donc elle déconnecte l’utilisateur.

Le AuthProvider englobe l’application et rend ces données disponibles à tous les composants enfants.

Enfin, le hook useAuth permet d’accéder facilement au contexte depuis n’importe quel composant. Il vérifie aussi qu’on utilise bien ce hook à l’intérieur du Provider.

Le message d’erreur dans `useAuth` est utile car il permet de détecter rapidement un oubli du Provider. Cela correspond directement au titre du sujet : sans Provider, le contexte d’authentification ne peut pas fonctionner correctement.

### 5.3 Conception de l’architecture d’authentification front-end (1h)

Pour garder une application propre, nous avons séparé les responsabilités. Le contexte gère l’état global d’authentification. Les pages gèrent l’affichage. Les services API s’occupent des requêtes HTTP. Les routes protégées contrôlent l’accès aux pages privées.

```text
src/
 ├── App.jsx
 ├── main.jsx
 ├── context/
 │    └── AuthProvider.jsx
 ├── routes/
 │    └── ProtectedRoute.jsx
 ├── pages/
 │    ├── LoginPage.jsx
 │    ├── ProfilePage.jsx
 │    └── PostsPage.jsx
 ├── services/
 │    └── apiClient.js
 └── components/
      └── Navbar.jsx
```

L’application doit ensuite être entourée par le Provider dans le fichier d’entrée. Cela permet à tous les composants placés dans `App` d’utiliser le hook `useAuth`.

```jsx
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { AuthProvider } from "./context/AuthProvider";

ReactDOM.createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <AuthProvider>
      <App />
    </AuthProvider>
  </React.StrictMode>
);
```

Cette organisation permet de créer des composants réutilisables. Le contexte joue ici le rôle d’access store, car il centralise les informations nécessaires à l’accès de l’utilisateur, comme `user`, `token`, `login` et `logout`. Par exemple, la Navbar peut afficher un bouton de connexion ou de déconnexion selon l’état global.

```jsx
function Navbar() {
  const { user, logout } = useAuth();

  return (
    <nav>
      {user ? (
        <button onClick={logout}>Déconnexion</button>
      ) : (
        <a href="/login">Connexion</a>
      )}
    </nav>
  );
}
```

### 5.4 Mise en place du AuthProvider et du hook useAuth (1.5h)

La réalisation principale du prosit consiste à créer un `AuthProvider` complet. Ce composant doit gérer la connexion, la déconnexion, la récupération de l’utilisateur courant et éventuellement la persistance du jeton après un rechargement de page.

Dans une première version, le Provider peut initialiser son état à partir du `localStorage`. Cela permet de conserver la connexion après un rafraîchissement. Même si cette solution n’est pas la plus sécurisée pour une application critique, elle est compréhensible dans un prototype et permet de bien comprendre le rôle du contexte.

```jsx
import { useEffect, useState } from "react";
import { AuthContext } from "./AuthContext";
import { loginRequest, getMeRequest } from "../services/authApi";

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem("accessToken"));
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function loadUser() {
      if (!token) {
        setLoading(false);
        return;
      }

      try {
        const currentUser = await getMeRequest(token);
        setUser(currentUser);
      } catch (error) {
        localStorage.removeItem("accessToken");
        setToken(null);
        setUser(null);
      } finally {
        setLoading(false);
      }
    }

    loadUser();
  }, [token]);

  async function login(email, password) {
    const data = await loginRequest(email, password);
    localStorage.setItem("accessToken", data.token);
    setToken(data.token);
    setUser(data.user);
  }

  function logout() {
    localStorage.removeItem("accessToken");
    setToken(null);
    setUser(null);
  }

  return (
    <AuthContext.Provider value={{ token, user, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
```

Dans cet exemple, `loading` est important car l’application doit parfois vérifier si un jeton déjà présent est encore valide. Pendant cette vérification, il ne faut pas rediriger trop rapidement l’utilisateur vers la page de connexion.

La page de connexion peut ensuite utiliser le hook `useAuth` pour appeler la fonction `login`.

```jsx
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/useAuth";

function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  async function handleSubmit(event) {
    event.preventDefault();

    try {
      await login(email, password);
      navigate("/profile");
    } catch (err) {
      setError("Email ou mot de passe incorrect");
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="email"
        value={email}
        onChange={(event) => setEmail(event.target.value)}
        placeholder="Email"
      />

      <input
        type="password"
        value={password}
        onChange={(event) => setPassword(event.target.value)}
        placeholder="Mot de passe"
      />

      {error && <p>{error}</p>}

      <button type="submit">Se connecter</button>
    </form>
  );
}

export default LoginPage;
```

### 5.5 Mise en place des routes protégées et contrôle d’accès aux pages (1h)

Une route protégée permet d’empêcher un utilisateur non connecté d’accéder à certaines pages. Par exemple, la page profil ou la page de gestion des posts ne doivent pas être accessibles sans authentification.

Avec React Router, on peut créer un composant `ProtectedRoute`. Il vérifie si un utilisateur est connecté. Si ce n’est pas le cas, il redirige vers la page de connexion.

```jsx
import { Navigate } from "react-router-dom";
import { useAuth } from "../context/AuthProvider";

function ProtectedRoute({ children }) {
  const { user } = useAuth();

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  return children;
}

export default ProtectedRoute;
```

Son utilisation dans les routes reste simple.

```jsx
<Route
  path="/profile"
  element={
    <ProtectedRoute>
      <ProfilePage />
    </ProtectedRoute>
  }
/>
```

Ce système protège l’interface, mais il ne doit pas être confondu avec une vraie sécurité serveur. Même si la page est cachée, un utilisateur pourrait appeler l’API directement. Le back-end doit donc toujours vérifier le JWT et utiliser l’identifiant de l’utilisateur contenu dans le jeton pour retourner uniquement ses propres données.

```js
app.get("/api/profile", authenticateToken, async (req, res) => {
  const profile = await getProfileByUserId(req.user.id);
  res.json(profile);
});
```

### 5.6 Gestion du stockage des jetons et sécurisation des appels API (1h)

Lorsque l’utilisateur se connecte, le front-end reçoit un JWT. Ce jeton doit ensuite être envoyé dans les requêtes vers les routes protégées. Dans une version simple, on peut le stocker dans le contexte React et éventuellement dans le `localStorage` pour garder la session après un rechargement.

```jsx
function login(newToken, userData) {
  localStorage.setItem("token", newToken);
  setToken(newToken);
  setUser(userData);
}

function logout() {
  localStorage.removeItem("token");
  setToken(null);
  setUser(null);
}
```

Pour éviter de répéter le même code dans toutes les pages, il est préférable de créer un client API centralisé. Ce fichier ajoute automatiquement le JWT dans l’en-tête `Authorization`.

```js
export async function apiFetch(url, options = {}) {
  const token = localStorage.getItem("token");

  return fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      ...options.headers
    }
  });
}
```

Ensuite, une page peut appeler une route protégée plus simplement.

```js
const response = await apiFetch("/api/profile");
const profile = await response.json();
```

Il faut aussi faire attention à ne pas stocker d’informations sensibles dans le JWT. Un JWT est signé, mais son contenu peut être lu. Il ne faut donc jamais mettre de mot de passe ou de donnée confidentielle dans le payload.

### 5.7 Gestion du CORS et routage avec Nginx ou Express (1h)

Le CORS apparaît lorsque le front-end et le back-end ne sont pas sur la même origine. Par exemple, si React tourne sur `http://localhost:5173` et l’API sur `http://localhost:3000`, le navigateur bloque certaines requêtes si le serveur ne les autorise pas explicitement.

Une première solution consiste à configurer CORS côté Express.

```js
const cors = require("cors");

app.use(cors({
  origin: "http://localhost:5173",
  methods: ["GET", "POST", "PUT", "DELETE"],
  allowedHeaders: ["Content-Type", "Authorization"],
  credentials: true
}));
```

Cette configuration indique au navigateur que le front-end a le droit d’appeler l’API. Il faut éviter d’utiliser `origin: "*"` avec des credentials, car ce n’est pas adapté à une application sécurisée.

Une autre solution consiste à passer par Nginx comme API Gateway. Le front-end appelle un seul point d’entrée, et Nginx redirige ensuite vers les bons services. Cela simplifie le routage et limite les problèmes d’origines différentes.

```nginx
upstream backend_service {
  server backend1:3000;
  server backend2:3000;
}

server {
  listen 80;

  location /auth/ {
    proxy_pass http://auth-service:4000/;
  }

  location /api/ {
    proxy_pass http://backend_service/;
    proxy_set_header Authorization $http_authorization;
  }
}
```

Dans cet exemple, Nginx ne fait pas seulement du routage vers `/auth` ou `/api`. Il peut aussi répartir les requêtes entre plusieurs instances du back-end grâce à l’`upstream`, ce qui correspond au principe de load balancing. La ligne qui transmet l’en-tête `Authorization` reste importante, car le back-end doit recevoir le JWT pour vérifier l’utilisateur.


## 6. Conclusion

Ce prosit permet de comprendre que la sécurité d’une application distribuée ne se limite pas au back-end. Le front-end doit gérer proprement l’état d’authentification, protéger les pages privées et envoyer le JWT de manière contrôlée.

React Context et le Provider permettent de centraliser l’utilisateur connecté, le jeton et les fonctions de connexion ou de déconnexion. Le hook personnalisé `useAuth` simplifie l’accès à ces données dans les composants et rend le code plus réutilisable.

Les routes protégées améliorent l’expérience utilisateur en empêchant l’accès visuel aux pages privées. Cependant, la vraie vérification doit rester côté back-end, car le front-end peut être contourné.

Le stockage des JWT doit être choisi avec prudence. Le `localStorage` est simple pour un prototype, mais il est sensible aux attaques XSS. Une solution plus sécurisée consiste à utiliser un access token court et un refresh token protégé dans un cookie `HttpOnly`, `Secure` et `SameSite`.

Enfin, la gestion du CORS peut être faite avec Express ou simplifiée avec Nginx comme API Gateway. Cette approche permet de garder un point d’entrée clair, de transmettre correctement l’en-tête `Authorization` et de préparer l’application à une architecture plus complète.
