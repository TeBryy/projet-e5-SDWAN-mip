# Projet E5 5SDWANA - Comptoir Lorrain sur Kubernetes

Groupe MIP : Bryan MAHAUT, Axel IRSUTTI, Axel PEIFFER (ESTIAM Metz, 5e année)
Empreinte : `MIP-04-11-2026`

3 applis en Go déployées sur un cluster k3s, dans 4 environnements (dev, mlops,
preprod, prod), avec autoscaling, paiement Stripe en mode test et supervision
Prometheus / Grafana.

| Appli | Rôle | Accès |
|---|---|---|
| vitrine | catalogue + bouton acheter | externe (Ingress) |
| paiement | paiement Stripe Checkout | externe (Ingress) |
| stock | API produits / stock | interne seulement |

## Contenu

- `apps/` : code des 3 applis + Dockerfile
- `k8s/` : manifests (base commune + un overlay par environnement), test de charge k6
- `observability/` : Prometheus / Grafana + dashboard
- `infra/` : installation de k3s
- `scripts/` : build, déploiement, test de charge, récupération des logs
- `docs/` : mesures, erreurs rencontrées, logs et captures

## Lancer le projet

Sur une VM Ubuntu 24.04 (4 vCPU, 8 Go) :

```bash
sudo ./infra/install-k3s.sh <IP_VM>
# renseigner DOCKERHUB_USER et VM_IP dans config.env
./scripts/configure.sh
./scripts/build-push.sh
echo "STRIPE_SECRET_KEY=sk_test_..." > .env.stripe
./scripts/deploy.sh
./observability/install.sh
./scripts/test-charge.sh prod all 800
./scripts/collecter-preuves.sh
```

Applis : `http://vitrine.<env>.<IP_VM>.sslip.io` et `http://paiement.<env>.<IP_VM>.sslip.io`
Grafana : `http://grafana.<IP_VM>.sslip.io`

Carte de test Stripe : 4242 4242 4242 4242
