#!/usr/bin/env bash
# Test de charge k6 + suivi des HPA toutes les 10 s jusqu'au retour à 3 pods
# ./scripts/test-charge.sh <env> [all|vitrine|stock|paiement] [req/s]
set -euo pipefail
cd "$(dirname "$0")/.."
source config.env
env="${1:?env requis (dev, mlops, preprod, prod)}"; cible="${2:-all}"; rps="${3:-800}"
ns="$env-mip-04-11-2026"; LOGS=docs/logs; mkdir -p "$LOGS"
suivi="$LOGS/hpa-$ns-$cible.log"

kubectl delete job loadtest-mip-04-11-2026 -n "$ns" --ignore-not-found --wait
# on remplace la cible et le débit dans le job avant de l'appliquer
kubectl kustomize k8s/loadtest \
  | sed -e "/name: CIBLE/{n;s/value: .*/value: \"$cible\"/}" \
        -e "/name: RPS_MAX/{n;s/value: .*/value: \"$rps\"/}" \
  | kubectl apply -n "$ns" -f -

echo "==> suivi HPA dans $suivi (Ctrl+C pour arrêter)"
debut=$(date +%s)
{
  while :; do
    echo "--- $(date '+%F %T') (t+$(( $(date +%s) - debut ))s)"
    kubectl get hpa -n "$ns" -o custom-columns=\
'HPA:.metadata.name,CPU%:.status.currentMetrics[0].resource.current.averageUtilization,PODS:.status.currentReplicas,CIBLE:.status.desiredReplicas'
    # on s'arrête quand le job est fini et que tout est revenu à 3 pods
    if kubectl get job loadtest-mip-04-11-2026 -n "$ns" -o jsonpath='{.status.conditions[*].type}' | grep -qE "Complete|Failed" \
       && [[ $(kubectl get hpa -n "$ns" -o jsonpath='{range .items[*]}{.status.currentReplicas}{"\n"}{end}' | sort -u) == "3" ]]; then
      echo "✓ test terminé et retour à 3 pods"; break
    fi
    sleep 10
  done
} | tee "$suivi"

kubectl logs -n "$ns" job/loadtest-mip-04-11-2026 > "$LOGS/k6-$ns-$cible.log"
kubectl describe hpa -n "$ns" > "$LOGS/describe-hpa-$ns.log"
echo "✓ Résultats k6 : $LOGS/k6-$ns-$cible.log"
