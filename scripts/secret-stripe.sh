#!/usr/bin/env bash
# Crée le secret Stripe à partir de .env.stripe (pas dans git)
# ./scripts/secret-stripe.sh [env]
set -euo pipefail
cd "$(dirname "$0")/.."
source config.env
[[ -f .env.stripe ]] || { echo "Erreur : créer .env.stripe avec STRIPE_SECRET_KEY=sk_test_..." >&2; exit 1; }
source .env.stripe
[[ "$STRIPE_SECRET_KEY" =~ ^(sk|rk)_test_ ]] || { echo "Erreur : il faut une clé de test (sk_test_)" >&2; exit 1; }

for env in ${1:-$ENVS}; do
  ns="$env-mip-04-11-2026"
  kubectl create secret generic stripe-mip-04-11-2026 -n "$ns" \
    --from-literal=STRIPE_SECRET_KEY="$STRIPE_SECRET_KEY" \
    --dry-run=client -o yaml \
  | kubectl label --local -f - nom=irsutti prenom=axel groupe=mip empreinte=MIP-04-11-2026 -o yaml \
  | kubectl apply -f -
done
