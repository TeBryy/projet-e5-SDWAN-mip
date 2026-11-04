#!/usr/bin/env bash
# Déploie les environnements
# ./scripts/deploy.sh          tous les envs
# ./scripts/deploy.sh dev      seulement dev
set -euo pipefail
cd "$(dirname "$0")/.."
source config.env
grep -q "newName: DOCKERHUB_USER/" k8s/base/kustomization.yaml && { echo "Erreur : lancer ./scripts/configure.sh avant" >&2; exit 1; }

for env in ${@:-$ENVS}; do
  ns="$env-mip-04-11-2026"
  echo "==================== $ns ===================="
  # namespace + secret stripe avant le reste
  kubectl apply -f "k8s/overlays/$env/namespace.yaml"
  kubectl label ns "$ns" nom=mahaut prenom=bryan groupe=mip empreinte=MIP-04-11-2026 env="$env" --overwrite >/dev/null
  ./scripts/secret-stripe.sh "$env"
  kubectl apply -k "k8s/overlays/$env" --dry-run=server >/dev/null
  kubectl apply -k "k8s/overlays/$env"
  for app in stock vitrine paiement; do
    kubectl rollout status -n "$ns" "deploy/$app-mip-04-11-2026" --timeout=180s
  done
  kubectl get deploy,hpa,ingress -n "$ns"
done
