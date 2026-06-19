# Penetration Test Report — Breezy (`riveta`)

**Target (frontend):** https://breezy.philippeluu.fr/
**Target (API gateway):** https://api.breezy.philippeluu.fr
**Engagement:** riveta
**Authorization:** Owner/operator, confirmed by operator
**Window:** 2026-06-12 20:18–20:35 UTC · ~17 min active testing · budget: unlimited
**Thoroughness requested:** Full · **effective:** Full at the HTTP/application layer (no browser automation or Kali tooling available in the test environment — see Coverage Gaps)

---

## 1. Executive summary

Breezy is a Next.js (App Router, behind Caddy) single-page app backed by a **Go microservices** API gateway at `api.breezy.philippeluu.fr` (posts on MongoDB, users on PostgreSQL). Overall the application is **well-built from a security standpoint**: object-level authorization, JWT validation, injection handling, mass-assignment protection, CORS and like-integrity all held up under testing.

No Critical or High issues were confirmed. Seven issues were confirmed at **Medium (4)** and **Low (3)** severity, clustered around **authentication hardening** (no brute-force protection, an email-verification gap, account/email enumeration) and **information exposure** (an unauthenticated user directory, verbose errors, missing security headers). One **suspected stored-XSS** could not be browser-confirmed and is flagged for manual follow-up.

The highest-value fixes are cheap: add login rate-limiting, enforce email verification at the authorization layer (not just at login), and authenticate the user-directory endpoint.

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 4 |
| Low | 3 |
| Unconfirmed | 1 |

---

## 2. Scope & methodology

**In scope:** the two public hosts above; unauthenticated testing plus three self-registered test accounts (`riveta-pentest-a/b/c@example.com`). No out-of-scope exclusions.

**Approach:** black-box recon (fingerprinting, JS-bundle mining to recover the API surface, route/endpoint discovery), then authenticated testing across the OWASP API Top-10: broken object-level auth (BOLA/IDOR), broken authentication, mass assignment, injection, security misconfiguration, SSRF, and business-logic abuse. A bounded burst of 15 failed logins was used for rate-limit testing. Test data was tagged `riveta-pentest` and the test posts were deleted at the end (HTTP 204).

**Tooling note:** no browser automation (Playwright) or Kali/MKS scanner server was available in the environment, so client-side execution (XSS) and network-layer scanning were out of reach — reflected in Coverage Gaps.

**Recovered architecture / API surface:** frontend BFF routes (`/api/auth/refresh|logout`, `/api/translate`) and gateway endpoints for users, posts, comments, likes/reposts, follows, messages (E2E-keyed), notifications, bookmarks, and auth. Backend is Go/Gin (leaked via validator error format); user IDs are UUIDs, post IDs are Mongo ObjectIDs.

---

## 3. Findings by severity

### 🟠 Medium

#### RIV-001 — No rate-limiting / lockout on authentication
*OWASP API2/API4 · `POST /auth/login`*
15 consecutive failed logins for a known account returned **401 every time** — no `429`, no backoff, no lockout. Combined with the user directory (RIV-003) and email enumeration (RIV-005), this leaves credentials open to unthrottled brute-force and stuffing.
**Fix:** per-account + per-IP rate limiting with exponential backoff and temporary lockout; CAPTCHA after N failures; alert on bursts.

#### RIV-002 — Email-verification control bypassable via registration token
*OWASP API2 · `POST /auth/register` vs `/auth/login` · `GET /users/me`*
`login` rejects unverified accounts (`{code: email_not_verified}`, 401), **but the JWT returned directly by `/auth/register` authorizes protected endpoints** — `GET /users/me` returned **200** with that token. The verification gate exists only on the login path, so it is trivially bypassed by using the registration token, and accounts can be created and operated under email addresses the registrant does not control.
**Fix:** enforce `email_verified` at the gateway/authorization layer for all protected routes; don't issue a fully-privileged access token at registration (or scope it to verification only).

#### RIV-003 — Unauthenticated full user directory
*OWASP API3 · `GET /users`, `GET /users/by-username/{username}`*
`GET /users` returns every user (`id` UUID, `username`, `is_active`, created/updated/username-change timestamps, `username_pending`) **with no authentication**, 20/page and fully pageable; `by-username` adds follower/following counts. This enables complete enumeration of the user base and leaks internal state fields.
**Fix:** authenticate directory listing; strip internal/state fields; if a public profile lookup is intended, return minimal fields and rate-limit.

#### RIV-004 — Missing HTTP security headers
*OWASP API8 · all frontend responses*
No `Content-Security-Policy`, `Strict-Transport-Security`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, or `Permissions-Policy`. Result: no clickjacking protection, no HSTS, no CSP defense-in-depth, MIME-sniffing exposure.
**Fix:** set the full header set at Caddy or via Next.js `headers()` (see remediation plan for exact values).

### 🟡 Low

#### RIV-005 — Account/email enumeration via registration
*OWASP API2 · `POST /auth/register`*
Existing email → **409** `{"error":"email déjà utilisé"}`; new email → **201**. A clean scriptable oracle (login itself is safe — generic error). Feeds targeted phishing and RIV-001.
**Fix:** generic acknowledgement regardless of existence (verify/login email sent out-of-band); rate-limit registration.

#### RIV-006 — Unauthenticated timeline exposure incl. author UUIDs
*OWASP API3 · `GET /posts`, `/posts/{id}`, `/posts/{id}/comments`*
Full timeline, content, counts and **author UUIDs** returned unauthenticated. A public feed is likely intentional, but exposing stable author UUIDs to anonymous clients enables correlation.
**Fix:** confirm public-feed intent; use opaque handles instead of internal UUIDs for unauthenticated responses.

#### RIV-007 — Verbose errors / internal-detail disclosure
*OWASP API8 · multiple*
Malformed IDs → `500 {"error":"erreur interne"}` (should be 400/404); validation errors leak Go struct field names (`RegisterRequest.Password … min tag`); `X-Powered-By: Next.js` present.
**Fix:** validate identifiers and return 400/404; generic error bodies; strip `X-Powered-By`; map validation errors to safe messages.

---

## 4. Unconfirmed / requires manual follow-up

#### RIV-U1 — Suspected stored XSS in post content *(would be High if confirmed)*
`<img src=x onerror=alert(document.domain)>` and `<svg onload=alert(1)>` are **stored verbatim** server-side (no HTML sanitisation on post content). Execution was **not** browser-verified — no browser automation was available, and React escapes text by default, so exploitability hinges on whether any view renders post content via `dangerouslySetInnerHTML` or a rich-text/markdown renderer.
**Next step:** post the payload and open the feed/post in a real browser; grep the frontend for `dangerouslySetInnerHTML` and any HTML/markdown rendering of `content`.

---

## 5. Tested and NOT vulnerable (assurance)

- **CVE-2025-29927** Next.js middleware auth bypass — all `x-middleware-subrequest` variants still 307 to `/login`.
- **JWT** — `alg:none`/`None`/`NONE`, signature stripping, garbage signature, and role-tamper-with-original-sig all rejected (401); HS256 weak-secret forge against a 24-word common-secret list all rejected.
- **Mass assignment** — ignored at registration (`role`/`is_admin`/`id`/`email_verified`) and at `PATCH /users/me` (`role`/`id`/`email`/`follower_count`).
- **BOLA/IDOR** — cross-user post delete → 403, edit → 404, profile modify → 404, message keys & conversations scoped to caller.
- **Injection** — typed binding rejects NoSQL operator objects (400); `by-username` SQLi treated as literal; search endpoints stable.
- **SSRF** — `POST /api/translate` ignores injected `url`.
- **Business logic** — repeated self-like idempotent (`likes_count` stays 1).
- **CORS** — arbitrary `Origin` not reflected. **Password policy** — minimum length enforced.

---

## 6. Coverage gaps

1. **Client-side execution (XSS)** not browser-verified — no Playwright/browser MCP in environment (RIV-U1).
2. **Admin/function-level authorization** (`/admin`, middleware-gated `/messages`) not exercised — no admin role obtainable, no admin creds supplied.
3. **E2E messaging** (`/messages/keys`, conversations) only partially exercised — requires key provisioning between accounts.
4. **Network layer** (port/TLS-cipher/service scanning) not performed — no Kali/MKS server connected; only the two public HTTPS hosts assessed.

To close 1–4: connect a browser MCP + Kali server and provide (or allow creation of) an admin account, then re-run authenticated and network phases.

---

## 7. Cleanup / test artifacts

Test **posts deleted** during the engagement (HTTP 204): `6a2c6ce6376d78ebc1f50b01`, `6a2c6d72376d78ebc1f50b02`.
Test **accounts** could not self-delete (`DELETE /users/me` → 403) — **please remove manually**:
`riveta-pentest-a@example.com` (`ae11a0bb…`), `riveta-pentest-b@example.com` (`855b1cbc…`), `riveta-pentest-c@example.com` (`fe296cf5…`), `riveta-pentest-nodup-9281@example.com`.

---

## Appendix — evidence pointers

Raw recon and per-phase scripts/outputs: `outputs/riveta/recon/` (`inventory.json`, `jwt_test.*`, `idor_test.sh`, `phase2.sh`, `phase3.sh`). Orchestration log (NDJSON): `outputs/riveta/activity/pentester-orchestrator.log`.
