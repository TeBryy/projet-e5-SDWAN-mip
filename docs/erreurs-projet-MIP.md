# Erreurs rencontrées - groupe MIP

---

### Erreur 01 : Multipass ne démarre pas la VM (Hyper-V absent)

| | |
|---|---|
| **Date** | 2026-11-04 |
| **Objectif recherché** | Créer la VM du cluster en ligne de commande avec Multipass |
| **Commande** | `multipass launch 24.04 --name vm-k3s-mip-04-11-2026 --cpus 4 --memory 8G --disk 30G` |
| **Erreur affichée** | `launch failed: The Hyper-V service does not exist. Ensure Hyper-V is installed correctly.` |
| **Cause** | Sous Windows, Multipass utilise Hyper-V par défaut, et Hyper-V n'est pas activé sur le PC. VMware est installé mais Multipass ne sait pas l'utiliser. |
| **Résolution** | Installation de VirtualBox et changement de pilote (comme dans le cours) : `winget install Oracle.VirtualBox` puis `multipass set local.driver=virtualbox`. VM relancée avec `--network Ethernet` pour obtenir une IP du réseau local (192.168.1.110). |

---

### Erreur 02 : avertissement « permission denied » et mauvaise IP du nœud

| | |
|---|---|
| **Date** | 2026-11-04 |
| **Objectif recherché** | Vérifier le nœud k3s après installation |
| **Commande** | `kubectl get nodes -o wide` |
| **Erreur affichée** | `WARN[0000] open /etc/rancher/k3s/config.yaml: permission denied` à chaque connexion ; colonne `EXTERNAL-IP` à `<none>`, `INTERNAL-IP` à `10.0.2.15` |
| **Cause** | 1) Le fichier de config k3s était en `0600`, donc lisible seulement par root, alors que `kubectl` le lit aussi. 2) Avec VirtualBox la VM a deux cartes réseau : `enp0s3` (NAT, 10.0.2.15) et `enp0s8` (pont, 192.168.1.110). k3s a pris la première. |
| **Logs** | `ip -4 a` : `10.0.2.15/24 … enp0s3`, `192.168.1.110/24 … enp0s8` |
| **Résolution** | `sudo chmod 644 /etc/rancher/k3s/config.yaml` (aucun secret dedans), ajout de `node-external-ip: 192.168.1.110` dans la configuration, `sudo systemctl restart k3s`. Après ~20 s : `EXTERNAL-IP 192.168.1.110`. On a aussi corrigé `infra/install-k3s.sh` pour qu'il prenne l'IP en paramètre. |

---

### Erreur 03 : format du fichier `.env.stripe`

| | |
|---|---|
| **Date** | 2026-11-04 |
| **Objectif recherché** | Déployer l'environnement dev |
| **Commande** | `./scripts/deploy.sh dev` |
| **Erreur affichée** | `.env.stripe: line 1: sk_test_…: command not found` (la clé apparaissait en clair dans le message) |
| **Cause** | Le fichier contenait juste la clé, sans `STRIPE_SECRET_KEY=`. Le script faisait un `source` du fichier, donc bash a essayé d'exécuter la clé et l'a affichée. |
| **Résolution** | Nouvelle clé de test générée dans Stripe (l'ancienne avait été affichée). Fichier réécrit au format `STRIPE_SECRET_KEY=sk_test_…`. `scripts/secret-stripe.sh` modifié pour lire la clé avec `grep` au lieu de `source`, comme ça elle ne s'affiche plus. Déploiement relancé : 9 pods prêts. |

---

### Erreur 04 : Grafana inaccessible (« no available server »), redémarrages en boucle

| | |
|---|---|
| **Date** | 2026-11-04 |
| **Objectif recherché** | Ouvrir Grafana après l'installation de la supervision (Partie 3) |
| **Commande** | `./observability/install.sh` puis navigateur sur `http://grafana.192.168.1.110.sslip.io` |
| **Erreur affichée** | Page Traefik `no available server` ; pod Grafana `RESTARTS 4`, `Last State: Terminated, Reason: Error, Exit Code: 137` |
| **Logs** | `kubectl describe pod … -l app.kubernetes.io/name=grafana` ; `kubectl get events` (pas d'échec de sonde de vie) ; `kubectl logs … -c grafana --previous` : pas d'erreur, le pod s'arrête pendant le démarrage |
| **Cause** | Limite mémoire de Grafana à 256Mi dans les values Helm : Grafana dépasse au démarrage, le conteneur est tué (code 137) et Traefik n'a plus de pod derrière l'Ingress. |
| **Résolution** | `requests.memory` 256Mi, `limits.memory` 512Mi dans `observability/values-kube-prometheus-stack.yaml`, `./observability/install.sh` (révision Helm 2). Pod `3/3 Running`, plus de redémarrage. |
