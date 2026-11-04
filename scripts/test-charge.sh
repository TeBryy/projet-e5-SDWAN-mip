#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
source config.env
env="${1:?env requis (dev, mlops, preprod, prod)}"; cible="${2:-all}"; rps="${3:-800}"
ns="$env-mip-04-11-2026"; LOGS=docs/logs; mkdir -p "$LOGS"
suivi="$LOGS/hpa-$ns-$cible.log"

kubectl delete job loadtest-mip-04-11-2026 -n "$ns" --ignore-not-found --wait
kubectl kustomize k8s/loadtest \
  | sed -e "/name: CIBLE/{n;s/value: .*/value: \"$cible\"/}" \
        -e "/name: RPS_MAX/{n;s/value: .*/value: \"$rps\"/}" \
  | kubectl apply -n "$ns" -f -

echo "suivi HPA dans $suivi (Ctrl+C pour arrêter)"
debut=$(date +%s)
{
  while :; do
    echo "--- $(date '+%F %T') (t+$(( $(date +%s) - debut ))s)"
    kubectl get hpa -n "$ns" -o custom-columns=\
'HPA:.metadata.name,CPU%:.status.currentMetrics[0].resource.current.averageUtilization,PODS:.status.currentReplicas,CIBLE:.status.desiredReplicas'
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
