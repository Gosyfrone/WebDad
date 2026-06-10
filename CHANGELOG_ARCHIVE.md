# CHANGELOG ARCHIVE — WebDad / Breezy

> Historical logs, resolved issues, completed-work history and obsolete debugging notes moved out of
> the auto-loaded context. The **session-by-session** work log lives in `CHANGELOG.md`. This file holds
> the *resolved issues* and *debugging history* that used to sit in CLAUDE.md §6.

---

## Resolved issues (formerly CLAUDE.md §6 "✅ RÉSOLU")

### ✅ Mongo `profiles` validator drift
An old `mongo-profil` volume kept a `$jsonSchema` still requiring `user_id` + **`username`** (before
`display_name` moved to profil-service) → the register's `POST /profils` (no longer sending `username`)
failed → no profile created → empty `display_name` search. **Fix:** `ensureCollections` now applies
`collMod` on existing collections (idempotent validator resync) — the service *maintains* its schema at
every boot, not only at creation. Verified e2e (validator back to `user_id`+`created_at`, insert without
username OK, search OK).
*Residual limit (not a bug):* login-only accounts (never through register) have no profile — a profile is
created only at register (`POST /profils`) or via the edit popup, never at login (decision: no lazy profile
provisioning). Such accounts aren't findable by `display_name` until they have a profile.

### ✅ Schema harmonization
All 4 original services (auth/post/user/profil) own their schema (embedded + `EnsureSchema` at boot). No
more mounted `init-db` scripts; `scripts/init-db/` emptied; `profil-init.js` removed. Later, message,
notification and media services followed the same autonomous pattern.

### ✅ profil-service implemented (repository/service)
Read (`GET /profils/:userId`, `GET /profils/me` — read-only, 404 if absent), edit (`PATCH /profils/me`,
`birth_date` set-once), creation = `POST /profils` only (display_name required), admin `DELETE`.

---

## Debugging history (resolved, formerly inline in §6 / changelog)

- **`make up` = frozen prod images.** The frontend image bakes `npm run build` (`output: standalone` →
  `node server.js`) and Go images bake a static binary, all frozen at build. `docker compose up` does NOT
  rebuild on source change → use `make dev` for hot-reload, or `make build` for fresh prod images. ⚠️ stale
  `webdad-*` images may be **dev** (air) images from a prior `make dev`; a prod `up` without `--build`
  relaunches them and air crashes (`.air.toml` absent).
- **Next `Cannot find module './NNN.js'`** (multiple occurrences) — incoherent `frontend/.next` cache after a
  local `npm run build` mixed with the Docker `next dev` bind-mount. Fix: delete `frontend/.next`, restart the
  frontend container.
- **Post pin "Épinglage impossible" (404 on `/pin`)** — the running post-service process hadn't loaded the new
  `PATCH /posts/:id/pin` route. Fix: `docker compose restart post-service`. Preventive: `init.go` now resyncs
  Mongo validators via `collMod` so old volumes accept `pinned_at`.
- **Messages back-navigation corrupted after `?dm=`** — cleanup via raw `window.history.replaceState` overwrote
  Next App Router internal state. Fix: clean via `router.replace('/messages', { scroll:false })`.
- **Community role not live-updated** — `connectRealtime` only handled `message` events, ignoring
  `member_role_changed`. Fix: expose non-message events; `messages-view` refetches on role/title change.
- **Trailing-slash 307 loop** — `POST /users` (collection endpoint) 307'd to `/users/` the service didn't know.
  Fix: `RedirectTrailingSlash=false` + 2 routes per service (bare prefix + `/*path`).
- **Go 1.22 → 1.25 bump** — 1.22 EOL (no more stdlib security patches → govulncheck CVEs). Bumped Dockerfiles to
  `golang:1.25-alpine`, `go 1.25.0` in the 5 `go.mod`, CI `setup-go: 1.25`, + `x/net@v0.55.0`,
  `golang-jwt/jwt/v5@v5.3.1` → govulncheck 0 vuln.
- **CLAUDE.md weight reduction (113 KB → ~64 KB)** — moved the "Last updated" history to `CHANGELOG.md`,
  condensed §3 status columns. (This very refactor continues that effort: see CHANGELOG.md.)

---

## Session work log

The full session-by-session log (feature merges, rebases, e2e verification notes) is in
**`CHANGELOG.md`**. Older entries are pushed here over time as `CHANGELOG.md` is trimmed to the latest
2–3 entries.
