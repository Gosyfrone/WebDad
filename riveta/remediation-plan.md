# Remediation Plan — Breezy (`riveta`)

Ticket-style, ordered by priority (impact ÷ effort). IDs map to `report.md`.

---

## P1 — Do first (cheap, high leverage)

### TICKET RIV-001 · Add authentication rate-limiting & lockout
**Severity:** Medium · **Effort:** S
- [ ] Add per-IP **and** per-account rate limiting to `POST /auth/login` (e.g. 5–10 attempts / 15 min) with exponential backoff.
- [ ] Temporary account lockout / step-up (CAPTCHA) after repeated failures.
- [ ] Apply the same throttling to `POST /auth/register` and password-reset.
- [ ] Emit a security log/alert on failed-login bursts.
**Done when:** a burst of failed logins starts returning `429` / lockout instead of unlimited `401`s.

### TICKET RIV-002 · Enforce email verification at the authorization layer
**Severity:** Medium · **Effort:** S–M
- [ ] Check `email_verified` in the gateway/JWT-validation middleware for **all** protected endpoints, not only at login.
- [ ] Either don't issue a usable access token from `/auth/register`, or issue a **verification-scoped** token that only permits the verify flow.
- [ ] Add a regression test: registration token must NOT access `/users/me` etc. until verified.
**Done when:** the registration token returns `403/401` on protected routes until the email is verified.

### TICKET RIV-003 · Authenticate the user directory & trim fields
**Severity:** Medium · **Effort:** S
- [ ] Require a valid session for `GET /users`.
- [ ] Remove internal fields (`username_pending`, `username_changed_at`, `is_active`, raw timestamps) from any publicly reachable response.
- [ ] If public profile lookup by username is a product requirement, keep only minimal fields and rate-limit it.
**Done when:** `GET /users` without a token returns `401`.

### TICKET RIV-004 · Add HTTP security headers
**Severity:** Medium · **Effort:** S
Set globally (Caddy site block or Next.js `headers()`):
```
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
Content-Security-Policy: default-src 'self'; img-src 'self' data:; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self' https://api.breezy.philippeluu.fr; frame-ancestors 'none'
```
- [ ] Tune CSP against the real asset/origin list (fonts, media, api host) and test in report-only mode first.
**Done when:** all six headers are present on responses and CSP is enforcing.

---

## P2 — Next

### TICKET RIV-005 · Stop registration email enumeration
**Severity:** Low · **Effort:** S
- [ ] Return a generic "check your email" response whether or not the address exists; handle the "already registered" case via an out-of-band email, not a `409` body.
- [ ] Combine with RIV-001 throttling.

### TICKET RIV-007 · Harden errors & input validation
**Severity:** Low · **Effort:** S
- [ ] Validate path identifiers (UUID/ObjectID) and return `400`/`404` instead of `500 "erreur interne"` (seen on `/users/{non-uuid}`, `/users/check-email`).
- [ ] Return generic error bodies; don't surface Go struct field names from the validator.
- [ ] Remove the `X-Powered-By: Next.js` header.

### TICKET RIV-006 · Review unauthenticated timeline exposure
**Severity:** Low · **Effort:** S (decision) / M (if changing)
- [ ] Confirm the public feed is intended.
- [ ] Avoid returning internal author **UUIDs** to anonymous clients — expose opaque handles instead; ensure non-public posts are gated.

---

## P3 — Verify (not yet a confirmed bug)

### TICKET RIV-U1 · Confirm or rule out stored XSS in post content
**Severity:** suspected (High if confirmed) · **Effort:** S to verify
- [ ] Post `<img src=x onerror=alert(document.domain)>` and view the feed/post in a real browser.
- [ ] `grep` the frontend for `dangerouslySetInnerHTML` and any markdown/HTML rendering of post `content`.
- [ ] If it executes: sanitise/escape on output (and ideally on input), and CSP (RIV-004) provides defense-in-depth.

---

## Closing the assessment's blind spots (optional, for a follow-up engagement)
- Provide an **admin account** and re-test function-level authorization on `/admin`.
- Connect a **browser automation** environment to verify XSS and exercise the E2E messaging flow.
- Run a **network/TLS scan** (nmap, testssl) against both hosts.
