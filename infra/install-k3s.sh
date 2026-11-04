#!/usr/bin/env bash
# Installe k3s, docker et helm sur la VM Ubuntu
# sudo ./infra/install-k3s.sh
set -euo pipefail

K3S_VERSION="${K3S_VERSION:-v1.33.5+k3s1}"
DIR="$(cd "$(dirname "$0")" && pwd)"
UTILISATEUR="${SUDO_USER:-$USER}"
HOME_UTIL="$(getent passwd "$UTILISATEUR" | cut -d: -f6)"

echo "==> paquets"
apt-get update -qq
apt-get install -y -qq curl git make jq apache2-utils docker.io >/dev/null
usermod -aG docker "$UTILISATEUR"

echo "==> config k3s"
install -D -m 0600 "$DIR/k3s-config.yaml" /etc/rancher/k3s/config.yaml

echo "==> installation k3s $K3S_VERSION"
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="$K3S_VERSION" sh -

echo "==> kubeconfig pour $UTILISATEUR"
install -d -m 0700 -o "$UTILISATEUR" "$HOME_UTIL/.kube"
install -m 0600 -o "$UTILISATEUR" /etc/rancher/k3s/k3s.yaml "$HOME_UTIL/.kube/config"

echo "==> Helm"
curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

echo "==> alias k + historique avec la date"
cat >> "$HOME_UTIL/.bashrc" <<'EOF'

# --- projet E5 (mip-04-11-2026) ---
export KUBECONFIG=$HOME/.kube/config
export HISTTIMEFORMAT="%F %T "
export HISTSIZE=100000 HISTFILESIZE=100000
shopt -s histappend
PROMPT_COMMAND="history -a; ${PROMPT_COMMAND:-}"
source <(kubectl completion bash)
alias k=kubectl
complete -o default -F __start_kubectl k
EOF

echo "==> attente du noeud"
until kubectl get nodes 2>/dev/null | grep -q " Ready"; do sleep 3; done
kubectl get nodes -o wide
echo
echo "OK, se déconnecter / reconnecter pour docker et le .bashrc"
