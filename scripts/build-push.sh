#!/usr/bin/env bash
# Build des 3 images, test rapide avec docker run, puis push sur Docker Hub
# (faire docker login avant)
set -euo pipefail
cd "$(dirname "$0")/.."
source config.env
LOGS=docs/logs; mkdir -p "$LOGS"
FP=mip-04-11-2026
NET=net-$FP

for app in stock vitrine paiement; do
  img="$DOCKERHUB_USER/$app-$FP:$VERSION"
  echo "==> build $img"
  docker build -t "$img" -t "$DOCKERHUB_USER/$app-$FP:latest" "apps/$app"
done
docker images | grep "$FP"

echo "==> test des images en local"
docker network inspect "$NET" >/dev/null 2>&1 || docker network create "$NET"
docker rm -f "stock-$FP" "vitrine-$FP" "paiement-$FP" 2>/dev/null || true
docker run -d --name "stock-$FP" --network "$NET" "$DOCKERHUB_USER/stock-$FP:$VERSION"
docker run -d --name "vitrine-$FP" --network "$NET" -p 18080:8080 \
  -e STOCK_URL="http://stock-$FP:8080" -e HOST_PAIEMENT=localhost:18082 "$DOCKERHUB_USER/vitrine-$FP:$VERSION"
docker run -d --name "paiement-$FP" --network "$NET" -p 18082:8080 \
  -e STOCK_URL="http://stock-$FP:8080" -e STRIPE_SECRET_KEY=sk_test_factice "$DOCKERHUB_USER/paiement-$FP:$VERSION"
sleep 2
curl -fsS localhost:18080/ | grep -q "Comptoir Lorrain" && echo "vitrine OK"
curl -fsS localhost:18082/ | grep -q "Service de paiement" && echo "paiement OK"
docker stats --no-stream | grep "$FP" | tee "$LOGS/docker-stats-$FP.log"

for app in stock vitrine paiement; do
  docker logs "$app-$FP" > "$LOGS/history-projet-$GROUPE-docker-conteneur-$app.log" 2>&1
done
docker rm -f "stock-$FP" "vitrine-$FP" "paiement-$FP" >/dev/null

echo "==> push Docker Hub"
for app in stock vitrine paiement; do
  docker push "$DOCKERHUB_USER/$app-$FP:$VERSION"
  docker push "$DOCKERHUB_USER/$app-$FP:latest"
done
echo "images poussées : https://hub.docker.com/u/$DOCKERHUB_USER"
