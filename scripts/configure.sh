#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
source config.env

if [[ "$DOCKERHUB_USER" == "DOCKERHUB_USER" || "$DOCKERHUB_USER" =~ [A-Z] ]]; then
  echo "Erreur : mettre DOCKERHUB_USER (en minuscules) dans config.env" >&2; exit 1
fi

sed -i -E "s#newName: [^/]+/(stock|vitrine|paiement)-#newName: ${DOCKERHUB_USER}/\1-#; s#newTag: .*#newTag: \"${VERSION}\"#" \
  k8s/base/kustomization.yaml

for env in $ENVS; do
  cat > "k8s/overlays/$env/hosts.env" <<EOF
HOST_VITRINE=vitrine.$env.$VM_IP.sslip.io
HOST_PAIEMENT=paiement.$env.$VM_IP.sslip.io
EOF
done

echo "images : ${DOCKERHUB_USER}/<app>-mip-04-11-2026:${VERSION}"
echo "URLs :"
for env in $ENVS; do
  echo "   $env : http://vitrine.$env.$VM_IP.sslip.io  |  http://paiement.$env.$VM_IP.sslip.io"
done
echo "   supervision : http://grafana.$VM_IP.sslip.io"
