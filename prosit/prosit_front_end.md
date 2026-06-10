# PROSIT FRONT END

Groupe :

STOFFEL Maxime – Secrétaire

LUU Philippe – Scribe

RIVET Alexandre – Gestionnaire du temps

TOUZE Romain - Animateur

# Sommaire

* Sommaire
* 1. Contexte
* 2. Mots inconnus / Notions à maîtriser
* 3. Problématique
* 4. Plan d’action (8h)
* 5. Réalisation

## 1. Contexte

Après avoir réalisé le développement du back-end de l’application de lecture de bandes dessinées, la prochaine étape consiste à construire l’interface utilisateur permettant d’exploiter les fonctionnalités proposées par l’API. L’objectif est de proposer une application web dans laquelle les utilisateurs peuvent consulter une bibliothèque de bandes dessinées numériques, accéder à une boutique de produits dérivés et interagir au sein d’un mini-réseau social.

Yanis est chargé de concevoir ce front-end en utilisant une architecture moderne reposant sur React et Tailwind CSS. Avant de commencer le développement, il réalise une phase de modélisation afin de définir les différentes pages de l’application ainsi que leur organisation. Cette étape passe notamment par la réalisation de wireframes permettant de représenter la structure générale de l’interface.

Lors du développement des premiers composants, des problèmes apparaissent sur l’affichage des bandes dessinées. Certaines données JSON provenant de l’API ne sont pas encore disponibles au moment où le composant tente de les afficher. Cela provoque des erreurs de rendu et met en évidence un problème de gestion du cycle de vie des composants.

Yanis s’interroge alors sur l’utilisation des outils choisis et sur la manière de gérer correctement le chargement des données dans React afin de produire une application fiable et évolutive.

## 2. Mots inconnus / Notions à maîtriser

* Front end : Correspond à la partie visible d’une application avec laquelle l’utilisateur interagit directement. Contrairement au back-end qui traite les données et la logique métier, le front-end se concentre sur l’affichage et l’expérience utilisateur.
* React : Bibliothèque JavaScript permettant de construire une interface sous forme de composants réutilisables. Chaque composant possède sa propre logique et son propre affichage.
* Tailwind CSS : Bibliothèque de styles basée sur des classes utilitaires permettant de construire rapidement une interface sans écrire de grandes feuilles CSS.
* Un composant : Représente une partie indépendante de l’interface pouvant être utilisée plusieurs fois dans l’application.
* Le cycle de vie d’un composant : Correspond aux différentes étapes qu’il traverse : création, affichage, mise à jour puis suppression.
* Le hook useEffect : Utilisé dans React pour exécuter certaines actions après l’affichage du composant. Il est souvent utilisé pour récupérer des données depuis une API.
* Le hook useState : Permet de stocker des données dans un composant et de déclencher une mise à jour de l’affichage lorsqu’elles changent.
* JSON (JavaScript Object Notation) :  Format d’échange de données largement utilisé entre le front-end et le back-end.
* Redux : Bibliothèque permettant de centraliser l’état global d’une application afin de simplifier le partage des données entre plusieurs composants.
* Les wireframes :  Maquettes simplifiées utilisées pour organiser visuellement une interface avant son développement.

## 3. Problématique

Comment concevoir une application front-end avec React permettant d’afficher correctement des données provenant d’une API tout en assurant une bonne gestion du cycle de vie des composants, du chargement des données JSON et de l’évolution future de l’application ?

## 4. Plan d’action (8h)

* Analyse des besoins fonctionnels et conception des wireframes (1h)
* Étude de l’architecture Front-end et organisation des composants (1h)
* Modélisation des données JSON et des flux de communication (1h)
* Développement des composants React et intégration Tailwind CSS (2h)
* Gestion du cycle de vie des composants et récupération des données API (1h30)
* Mise en place de la gestion d’état globale avec Redux (0.5h)
* Validation du fonctionnement et optimisation du rendu (1h)

## 5. Réalisation

### 5.1 Analyse des besoins fonctionnels et conception des wireframes (1h)

Avant de commencer le développement du front-end, nous avons analysé les besoins exprimés dans le sujet afin d’identifier les fonctionnalités attendues et les parcours utilisateurs principaux. L’application devait permettre de consulter une bibliothèque de bandes dessinées numériques, d’accéder à une boutique et d’utiliser un mini-réseau social.

Pour préparer le développement, nous avons réalisé des wireframes simples afin de représenter l’organisation des pages avant de coder l’interface. Par exemple, la page bibliothèque peut être structurée avec une barre de navigation en haut, une zone de filtres à gauche et une grille de bandes dessinées au centre.

Exemple de structure prévue :

```text
------------------------------------------------
| Navbar : Accueil | Bibliothèque | Boutique |
------------------------------------------------
| Filtres        | Liste des bandes dessinées |
| Genre          | [BD] [BD] [BD]             |
| Auteur         | [BD] [BD] [BD]             |
------------------------------------------------
```

Cette étape nous a permis d’anticiper les composants nécessaires comme la barre de navigation, la carte d’une bande dessinée, la liste des BD et les filtres de recherche.

### 5.2 Étude de l’architecture Front-end et organisation des composants (1h)

Une fois les besoins définis, nous avons étudié l’architecture du front-end afin d’obtenir une application claire et maintenable. Nous avons choisi de découper le projet en plusieurs composants React, chacun ayant une responsabilité précise.

Par exemple, l’application peut être organisée avec un composant principal App, un composant Layout pour la structure commune, puis des pages comme LibraryPage, ShopPage et CommunityPage.

Exemple d’organisation :

```text
src/
 ├── App.jsx
 ├── components/
 │    ├── Navbar.jsx
 │    ├── ComicCard.jsx
 │    └── ComicList.jsx
 ├── pages/
 │    ├── LibraryPage.jsx
 │    ├── ShopPage.jsx
 │    └── CommunityPage.jsx
 └── services/
      └── api.js
```

Cette organisation permet de séparer la navigation, les pages et les composants métiers. Elle facilite aussi les évolutions, car un composant peut être modifié sans impacter toute l’application.

### 5.3 Modélisation des données JSON et flux de communication (1h)

Le front-end récupère ses données depuis l’API développée précédemment. Les données sont reçues au format JSON, puis utilisées par React pour afficher les bandes dessinées.

Nous avons donc défini une structure simple pour représenter une bande dessinée. Chaque objet contient les informations nécessaires à l’affichage, comme l’identifiant, le titre, l’auteur, l’image de couverture et le genre.

Exemple de donnée JSON :

```json
{
  "id": 1,
  "title": "One Piece",
  "author": "Eiichiro Oda",
  "genre": "Aventure",
  "cover": "/images/one-piece.jpg"
}
```

Ces données sont ensuite récupérées via une fonction placée dans un fichier séparé afin de ne pas mélanger la logique d’appel API avec l’affichage.

Exemple :

```js
export async function getComics() {
  const response = await fetch("/api/comics");
  return response.json();
}
```

Cette séparation permet d’avoir un code plus lisible et de réutiliser facilement les appels API dans plusieurs composants.

### 5.4 Développement des composants React et intégration Tailwind CSS (2h)

Le développement du prototype a été réalisé avec React. Chaque élément important de l’interface a été transformé en composant indépendant. Par exemple, l’affichage d’une bande dessinée peut être réalisé dans un composant ComicCard.

Exemple de composant :

```jsx
function ComicCard({ comic }) {
  return (
    <div className="p-4 rounded shadow bg-white">
      <img src={comic.cover} alt={comic.title} className="w-full rounded" />
      <h2 className="text-lg font-bold mt-2">{comic.title}</h2>
      <p className="text-sm text-gray-600">{comic.author}</p>
    </div>
  );
}

export default ComicCard;
```

Dans cet exemple, les données de la BD sont reçues avec les props. Tailwind CSS est utilisé directement dans les classes afin de gérer les espacements, les bordures, les ombres et la taille du texte.

Cette approche permet d’obtenir un composant simple, réutilisable et facilement modifiable.

### 5.5 Gestion du cycle de vie et récupération des données API (1h30)

Le principal problème rencontré concernait le chargement des données JSON. Lors du premier rendu, certaines données n’étaient pas encore disponibles, ce qui pouvait provoquer des erreurs d’affichage.

Pour résoudre ce problème, nous avons utilisé useState pour stocker les données et useEffect pour déclencher l’appel API après le montage du composant.

Exemple :

```jsx
import { useEffect, useState } from "react";
import { getComics } from "../services/api";

function LibraryPage() {
  const [comics, setComics] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getComics()
      .then((data) => setComics(data))
      .catch((error) => console.error(error))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return <p>Chargement des bandes dessinées...</p>;
  }

  return (
    <div>
      {comics.map((comic) => (
        <ComicCard key={comic.id} comic={comic} />
      ))}
    </div>
  );
}
```

Le tableau vide dans useEffect indique que l’appel API ne doit être exécuté qu’une seule fois au montage du composant. L’état loading permet d’éviter d’afficher la liste avant que les données soient chargées.

Cette solution répond directement au problème d’initialisation rencontré dans le sujet.

### 5.6 Mise en place de la gestion d’état globale avec Redux (0.5h)

Après avoir stabilisé le chargement des données, nous avons étudié l’utilisation de Redux. Dans un premier temps, useState suffit pour gérer les données d’une page. Cependant, Redux devient utile lorsque plusieurs composants ou plusieurs pages doivent accéder aux mêmes données.

Par exemple, si la bibliothèque, le profil utilisateur et la boutique doivent accéder aux informations de l’utilisateur connecté, il est plus propre de centraliser ces données.

Exemple simplifié de store Redux :

```js
import { configureStore } from "@reduxjs/toolkit";
import comicsReducer from "./comicsSlice";

export const store = configureStore({
  reducer: {
    comics: comicsReducer
  }
});
```

Exemple d’accès aux données dans un composant :

```jsx
import { useSelector } from "react-redux";

function ComicCounter() {
  const comics = useSelector((state) => state.comics.items);

  return <p>Nombre de BD : {comics.length}</p>;
}
```

Nous avons donc conclu que Redux n’était pas obligatoire pour un prototype simple, mais qu’il serait pertinent si l’application devait évoluer avec davantage de données partagées.

### 5.7 Validation du fonctionnement et optimisation du rendu (1h)

La dernière étape a consisté à vérifier que l’application fonctionnait correctement. Nous avons contrôlé que les données JSON étaient bien récupérées, que les composants ne généraient plus d’erreurs au rendu et que la navigation restait fluide.

Nous avons aussi ajouté quelques sécurités dans les composants pour éviter les erreurs si une donnée est manquante.

Exemple :

```jsx
<h2>{comic.title || "Titre indisponible"}</h2>
<p>{comic.author || "Auteur inconnu"}</p>
```

Ce type de vérification permet d’éviter qu’une donnée absente dans le JSON bloque l’affichage de toute la page.

Nous avons également veillé à utiliser une clé unique lors de l’affichage des listes avec map, car cela aide React à mettre à jour correctement les éléments affichés.

Exemple :

```jsx
{comics.map((comic) => (
  <ComicCard key={comic.id} comic={comic} />
))}
```

Les tests réalisés montrent que l’architecture React associée à Tailwind permet de répondre aux besoins du sujet. L’application peut afficher les données de manière fiable et reste organisée pour accueillir de nouvelles fonctionnalités.
