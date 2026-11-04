#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
source config.env
[[ -f .env.stripe ]] || { echo "Erreur : créer .env.stripe avec STRIPE_SECRET_KEY=sk_test_..." >&2; exit 1; }
STRIPE_SECRET_KEY=$( (grep -E '^STRIPE_SECRET_KEY=' .env.stripe || true) | head -1 | cut -d= -f2- | tr -d '[:space:]"')
if [[ ! "$STRIPE_SECRET_KEY" =~ ^(sk|rk)_test_ ]]; then
  echo "Erreur : .env.stripe doit contenir STRIPE_SECRET_KEY=sk_test_... (clé de test)" >&2
  exit 1
fi

for env in ${1:-$ENVS}; do
  ns="$env-mip-04-11-2026"
  kubectl create secret generic stripe-mip-04-11-2026 -n "$ns" \
    --from-literal=STRIPE_SECRET_KEY="$STRIPE_SECRET_KEY" \
    --dry-run=client -o yaml \
  | kubectl label --local -f - nom=irsutti prenom=axel groupe=mip empreinte=MIP-04-11-2026 -o yaml \
  | kubectl apply -f -
done
