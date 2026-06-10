#!/usr/bin/env bash
# Agrège les specs Swagger 2.0 par service en un openapi.{json,yaml} unique.
# Usage : bash scripts/swagger-aggregate.sh
set -euo pipefail

SWAG="${HOME}/go/bin/swag"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

SERVICES=(auth-service user-service profil-service post-service message-service notification-service media-service)

echo "→ Génération par service..."
for svc in "${SERVICES[@]}"; do
  dir="$ROOT/$svc"
  echo "  swag init : $svc"
  (cd "$dir" && "$SWAG" init \
    --generalInfo main.go \
    --dir . \
    --output ./docs \
    --parseDependency \
    --parseInternal \
    --quiet 2>&1 || true)
done

echo "→ Agrégation..."
mkdir -p "$ROOT/doc"

# Assemble paths + definitions de tous les services dans le spec du premier.
BASE="$ROOT/auth-service/docs/swagger.json"
if [ ! -f "$BASE" ]; then
  echo "❌ $BASE introuvable — génération auth-service a échoué"
  exit 1
fi

# Construit le spec agrégé avec jq : merge paths et definitions de chaque service.
MERGED=$(cat "$BASE")

for svc in "${SERVICES[@]:1}"; do
  spec="$ROOT/$svc/docs/swagger.json"
  [ -f "$spec" ] || { echo "  ⚠ $spec absent, ignoré"; continue; }
  MERGED=$(echo "$MERGED" | jq \
    --slurpfile other <(cat "$spec") \
    '.paths += $other[0].paths | .definitions += ($other[0].definitions // {})')
done

# Mise à jour des métadonnées globales du spec agrégé.
MERGED=$(echo "$MERGED" | jq '
  .info.title = "Breezy API" |
  .info.description = "API agrégée de la plateforme Breezy (réseau social microservices)." |
  .info.version = "1.0" |
  .host = "localhost:8080" |
  .basePath = "/"
')

echo "$MERGED" | jq '.' > "$ROOT/doc/openapi.json"

# Conversion JSON → YAML (python3 est disponible partout).
python3 - <<'PY'
import json, sys
try:
    import yaml
    data = json.load(open("doc/openapi.json"))
    with open("doc/openapi.yaml", "w") as f:
        yaml.dump(data, f, allow_unicode=True, sort_keys=False)
    print("  ✓ doc/openapi.yaml généré")
except ImportError:
    print("  ⚠ pyyaml absent — seul doc/openapi.json produit (pip install pyyaml pour le YAML)")
PY

echo "✓ Specs agrégés → doc/openapi.json"
ls -lh "$ROOT/doc/"
