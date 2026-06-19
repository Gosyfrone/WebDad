# Breezy — Executive Summary (`riveta`)

**Assessed:** breezy.philippeluu.fr + api.breezy.philippeluu.fr · 2026-06-12 · authorized by owner

## Bottom line
Breezy is a **well-engineered application**. The core defenses an attacker would target first — access control between users, login token integrity, injection, and data-tampering protections — all **held up**. No critical or high-risk vulnerabilities were confirmed.

The issues found are about **hardening the front door and reducing what anonymous visitors can see**, not broken core security.

## What was found
- **4 medium-risk** and **3 low-risk** issues. **0 critical, 0 high.**
- The themes:
  1. **Login has no brute-force protection** — an attacker can guess passwords without being slowed down or locked out.
  2. **Email verification can be skipped** — signing up hands out a working access pass even before the email is verified, so the verification step doesn't really gate access.
  3. **Anyone can list every user** — the full member directory (usernames, IDs, signup dates) is readable without logging in, and it's easy to tell whether a given email is registered.
  4. **Standard browser-protection headers are missing** (clickjacking, HTTPS-pinning, content-sniffing defenses).
- One **possible "stored cross-site scripting"** issue (malicious post content) could **not be confirmed** here because the test setup had no browser to watch it run. It should be checked manually — if real, it would be high-risk.

## What this means for the business
None of these let an attacker immediately take over accounts or data today. But together they make **account-guessing attacks** and **user privacy exposure** easier than they should be. The fixes are small, well-understood, and low-effort.

## Top 3 actions
1. **Add rate-limiting and lockout to login** (and registration).
2. **Enforce email verification everywhere**, not just on the login screen.
3. **Require login to browse the user directory** and stop revealing whether an email is registered.

Plus: add the standard security headers, and manually verify the suspected post-content XSS.

*Full technical detail in `report.md`; step-by-step fixes in `remediation-plan.md`.*
