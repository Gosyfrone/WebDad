# PROMPTING — token-efficient workflow & prompt templates

> **NOT auto-loaded.** This file is read on demand only:
> - by the human as a cheat-sheet (Part A),
> - by Claude when explicitly asked *"from PROMPTING.md, write me the prompt for: …"* (Part B).
>
> Do **not** copy its content into CLAUDE.md or memory — that would defeat the purpose (it would
> be paid on every message). One ad-hoc `Read` when needed is the whole point.

---

# Partie A — Cheat-sheet (pour toi, humain)

## Les 3 réflexes qui rapportent le plus
1. **`/clear` entre deux tâches non liées.** La conversation entière est renvoyée à chaque message →
   traîner l'historique d'une tâche finie te le fait payer en boucle. C'est le plus gros gain quotidien, gratuit.
2. **CLAUDE.md mince** (< ~250 lignes). Le reste (ARCHITECTURE / DECISIONS / STATUS / ce fichier) n'est
   payé **que quand Claude l'ouvre**. Ne laisse pas CLAUDE.md regonfler.
3. **"via graphify" / "réponds sans lire"** sur les questions ; **Haiku/Sonnet** sur le trivial.

## Choisir le modèle (`/model`)
| Tâche | Modèle |
|---|---|
| Conception d'archi, code de sécurité/délicat, debug subtil | **Opus** |
| Rebase, renommage, reformat, génération d'un service bien spécifié | **Sonnet** |
| Question simple, exécution mécanique pure | **Haiku** |
Règle : **Opus quand "il faut réfléchir", Sonnet/Haiku quand "il faut juste exécuter".**
Astuce : lance sur Sonnet, **escalade sur Opus seulement le passage dur** (ex. un conflit de rebase ambigu).

## Prompting qui économise
- Borne le **scope** : "modifie seulement `X`", donne le chemin/ligne si tu le connais.
- "**Réponds, ne code pas**" / "**ne lis pas, réponds de mémoire**" quand tu veux juste un avis.
- "**Verdict en N lignes**" → pas de pavé.
- **Groupe** tes demandes (un aller-retour = tout le contexte rechargé).
- **Coupe tôt** (Échap) si Claude part dans la mauvaise direction.
- **Valide l'archi avant de coder** (règle ⛔) : un code à jeter coûte double.

## Le workflow "compilateur de prompt" (pour les features)
1. Tu décris en vrac ce que tu veux → "à partir de `PROMPTING.md`, rédige-moi le prompt pour : …".
2. Claude génère un prompt scopé/phasé/modèle-choisi (Partie B).
3. Tu **`/clear`**.
4. Tu colles le prompt dans un contexte vierge → l'exécution démarre propre et minimale.
- À réserver aux **features / tâches multi-étapes**. Pour un truc trivial, c'est plus cher que la tâche.

## Graphify (rappel)
- **`graphify-out/` est gitignore** → un clone neuf ne l'a PAS. Chacun (re)génère en local : `graphify .`
  au clone, puis `graphify update .` (gratuit, AST) en fin de session de dev.
- Questions archi/"où est X" → `graphify query` (sous-graphe ciblé, 5-10× moins cher qu'ouvrir des fichiers).
- `/graphify .` complet = **payant** (re-sémantique) → rare, seulement si beaucoup de docs ajoutées.
- Ne fais pas lire `GRAPH_REPORT.md` en entier pour une question ciblée.
- **Dis-moi si tu as graphify ou pas** ("j'ai graphify" / "pas de graphify") → j'évite de vérifier
  (= économie). Sans graphify, je m'appuie sur `ARCHITECTURE.md`/`DECISIONS.md` + grep ciblé.

## Réflexes anti-gaspillage
- CHANGELOG : jamais dans l'auto-load. Mémoire (`MEMORY.md`) : seulement des faits durables.
- Pas de duplication entre docs (contenu répété = payé plusieurs fois à l'ouverture).
- Docs "pour Claude" en **anglais** (~10-15% moins cher à lire).

---

# Part B — Prompt generation templates (for Claude)

When the user says *"from PROMPTING.md, write me the prompt for: <description>"*, produce a **single,
self-contained, copy-pasteable prompt** that the user will paste into a fresh (`/clear`ed) session.

## Graphify availability (decide once, cheaply)
`graphify-out/` is **gitignored** → it is NOT guaranteed to exist in a given checkout.
- If the user **states** it ("j'ai graphify" / "pas de graphify"), **trust them — do not check** (saves tokens).
- Else, a **single** `test -f graphify-out/graph.json` is acceptable; don't probe further.
- **With graphify:** the generated prompt may instruct "via graphify, …".
- **Without graphify:** the generated prompt must instead say "via ARCHITECTURE.md/DECISIONS.md +
  targeted grep (don't scan the whole repo)". Never make a prompt depend on a graph that may be absent.

## Generation rules (always apply)
1. **Self-contained.** The generated prompt must work with zero prior conversation. Reference the right
   docs (`DECISIONS.md`, `ARCHITECTURE.md`, `PROJECT_STATUS.md`) and — **only if available** — `graphify`,
   rather than assuming context.
2. **Pick the model.** Start the prompt with a `/model <sonnet|opus|haiku>` line + one-clause why.
   Conception/security → opus; well-spec'd execution → sonnet; trivial → haiku.
3. **Bound the scope** explicitly (which files/service, what NOT to touch).
4. **Respect the ⛔ rule:** for any feature, the FIRST generated prompt is a **design/proposal** prompt
   that says *"don't write code"* and lists the decisions to settle. Implementation prompts come **after**
   the user validates the architecture, **one phase per prompt**.
5. **Minimize exploration:** tell Claude to use `graphify query`/the summary docs instead of scanning;
   give known paths.
6. **Control the output:** specify the expected format and length ("structured markdown, no code",
   "5-line report"), and add **"stop and ask if a choice is ambiguous"** to avoid wasted generation.
7. **Defer doc/test side-effects** when they'd bloat a mechanical task (e.g. rebases): explicitly say
   whether to update `PROJECT_STATUS.md`/`CHANGELOG.md` and whether to run tests.
8. Keep the generated prompt **tight** — it is a spec, not an essay. Output it in a fenced block so the
   user can copy it cleanly.

## Template — feature, Phase A (design / proposal)
```
/model opus
Nouvelle feature : <description>. **Ne code rien.**
1. Via <graphify si dispo, sinon ARCHITECTURE.md/DECISIONS.md + grep ciblé>, dis comment ça s'insère
   dans l'archi existante (services, gateway, BFF, front).
2. Propose l'architecture en tranchant chaque point ci-dessous, avec un choix recommandé + 1 ligne de justif :
   - <décision 1 : nouveau microservice vs extension d'un service existant ? quelle DB ?>
   - <décision 2 : modèle de données / tables / collections, où vit l'état>
   - <décision 3 : flux d'auth / sécurité / intégration access+refresh si pertinent>
   - <décision 4 : impact gateway/BFF/front, nouvelles routes, nouvelles env vars>
3. Découpe en phases livrables (façon média 0→3).
Pose-moi une question si un choix t'appartient pas. Réponds en markdown structuré, sans code.
```

## Template — feature, Phase B (implement one validated phase)
```
/model <sonnet|opus>   # opus seulement si la phase touche de la sécu/logique délicate
Implémente la Phase <N> de <feature>. Archi déjà validée : <coller le résumé figé, 5 lignes>.
Scope : seulement <chemins/service>. Ne touche pas à <le reste>.
Respecte les conventions (service autonome + EnsureSchema, .env pour les secrets, i18n FR/EN, etc.).
À la fin : build + lint, mets à jour PROJECT_STATUS.md + une entrée CHANGELOG. Rapport court.
Stop et demande si un point d'archi non couvert apparaît.
```

## Template — rebase
```
/model sonnet
Rebase sur <origin/branch>. Résous les conflits en gardant les deux features quand c'est possible.
Ne mets PAS à jour CLAUDE.md/CHANGELOG, ne lance pas les tests.
Si un conflit est ambigu, arrête-toi et liste-le au lieu de deviner.
Rapport final 5 lignes max : commits rejoués, fichiers en conflit, ce que tu as gardé.
```
(Si un conflit est non trivial → l'utilisateur escalade ce conflit-là sur opus.)

## Template — codebase question (cheap)
```
/model <haiku|sonnet>
<Via graphify, | Via ARCHITECTURE.md/DECISIONS.md + grep ciblé,> <question>.
Réponds en N lignes, n'ouvre les fichiers source qu'en dernier recours.
```

## Notes for the generator
- If the user's description is too vague to settle the decision list, **ask 1–3 clarifying questions
  first**, then generate — a vague prompt causes expensive exploration downstream.
- Adapt the decision bullets to the actual feature; don't emit placeholder text.
- Default to the smallest model that fits; only reach for opus when reasoning genuinely matters.
