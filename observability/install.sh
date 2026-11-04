#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
source config.env
NS=monitoring-mip-04-11-2026
CHART_VERSION="${CHART_VERSION:-92.17.1}"

kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
kubectl label ns "$NS" nom=peiffer prenom=axel groupe=mip empreinte=MIP-04-11-2026 --overwrite

if ! kubectl get secret grafana-admin-mip-04-11-2026 -n "$NS" >/dev/null 2>&1; then
  MDP=$(head -c 18 /dev/urandom | base64 | tr -d '/+=')
  kubectl create secret generic grafana-admin-mip-04-11-2026 -n "$NS" \
    --from-literal=admin-user=admin --from-literal=admin-password="$MDP"
fi

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null
helm repo update >/dev/null
helm upgrade --install obs-mip-04-11-2026 prometheus-community/kube-prometheus-stack \
  --namespace "$NS" ${CHART_VERSION:+--version "$CHART_VERSION"} \
  -f observability/values-kube-prometheus-stack.yaml \
  --set "grafana.ingress.hosts[0]=grafana.$VM_IP.sslip.io" \
  --wait --timeout 10m

kubectl apply -k observability
helm list -n "$NS"
echo
echo "Grafana : http://grafana.$VM_IP.sslip.io"
echo "  identifiant : admin"
echo "  mot de passe : kubectl get secret grafana-admin-mip-04-11-2026 -n $NS -o jsonpath='{.data.admin-password}' | base64 -d"
echo "  dashboard : E5 Comptoir Lorrain - Autoscaling"
