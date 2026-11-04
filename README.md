# Projet E5 5SDWANA - Comptoir Lorrain

Groupe MIP : Bryan MAHAUT, Axel IRSUTTI, Axel PEIFFER - ESTIAM Metz

Empreinte : MIP-04-11-2026

Projet Kubernetes avec k3s : 3 applis en Go, vitrine, paiement avec Stripe et stock.
Elles sont déployées dans 4 environnements, dev, mlops, preprod et prod, avec un HPA et Grafana.

Le compte rendu est dans `docs/`.

## Lancer le projet

```bash
sudo ./infra/install-k3s.sh <IP_VM>
./scripts/configure.sh
echo "STRIPE_SECRET_KEY=sk_test_..." > .env.stripe
./scripts/deploy.sh
./observability/install.sh
./scripts/test-charge.sh prod all 800
./scripts/collecter-preuves.sh
```

Les applis sont sur `http://vitrine.<env>.<IP_VM>.sslip.io` et `http://paiement.<env>.<IP_VM>.sslip.io`, Grafana sur `http://grafana.<IP_VM>.sslip.io`.
