#!/usr/bin/env bash
set -uo pipefail
cd "$(dirname "$0")/.."
source config.env
L=docs/logs; mkdir -p "$L"
P="history-projet-$GROUPE"

HISTTIMEFORMAT="%F %T " HISTFILE=~/.bash_history bash -ic 'history' > "$L/$P.log" 2>/dev/null || true
[[ -s "$L/$P.log" ]] || cp ~/.bash_history "$L/$P.log" 2>/dev/null || true
[[ -s "$L/$P.log" ]] || echo "historique vide, faire : history > $L/$P.log" >&2

{
  echo "cluster $(date '+%F %T')"; kubectl version; echo
  kubectl get nodes -o wide; echo
  kubectl get ns --show-labels | grep -E "NAME|mip-04-11-2026"; echo
  helm list -A 2>/dev/null
} > "$L/$P-cluster.log" 2>&1

for env in $ENVS; do
  ns="$env-mip-04-11-2026"
  kubectl get ns "$ns" >/dev/null 2>&1 || continue
  kubectl get all,ingress,hpa,netpol,quota,limitrange,cm,secret -n "$ns" -o wide \
    > "$L/$P-k8s-$env-get.log" 2>&1
  kubectl describe deploy,rs,pod,svc,ingress,hpa,netpol,quota,limitrange -n "$ns" \
    > "$L/$P-k8s-$env-describe.log" 2>&1
  kubectl events -n "$ns" > "$L/$P-k8s-$env-events.log" 2>&1
  for app in stock vitrine paiement; do
    kubectl logs -n "$ns" -l app="$app-mip-04-11-2026" --prefix --tail=200 \
      > "$L/$P-k8s-$env-logs-$app.log" 2>&1
  done
done

ns=monitoring-mip-04-11-2026
if kubectl get ns "$ns" >/dev/null 2>&1; then
  kubectl get all,ingress -n "$ns" > "$L/$P-k8s-monitoring-get.log" 2>&1
  kubectl events -n "$ns" > "$L/$P-k8s-monitoring-events.log" 2>&1
fi

grep -rlE "sk_(test|live)_[A-Za-z0-9]{10,}" "$L" && { echo "Erreur : clé Stripe trouvée dans $L, à enlever avant de commit" >&2; exit 1; }
ls -la "$L"
