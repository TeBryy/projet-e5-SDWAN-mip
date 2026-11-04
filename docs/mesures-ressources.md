# Mesures CPU / RAM avant de fixer les limites

On a lancé les 3 images avec docker, puis envoyé du trafic avec k6 (20 puis
200 req/s) en regardant `docker stats`.

| Appli | RAM au repos | CPU à 20 req/s | CPU à 200 req/s | RAM à 200 req/s |
|---|---|---|---|---|
| vitrine  | 1,9 Mi | ~20m | ~200m | 8,7 Mi |
| stock    | 1,8 Mi | ~8m  | ~80m  | 7,0 Mi |
| paiement | 1,9 Mi | ~10m | ~90m  | 7,7 Mi |

## Valeurs mises dans les manifests

| Appli | requests | limits |
|---|---|---|
| vitrine  | 50m / 16Mi | 200m / 64Mi |
| stock    | 25m / 16Mi | 100m / 48Mi |
| paiement | 25m / 16Mi | 150m / 64Mi |

- requests : à peu près la conso avec un trafic normal. Le HPA calcule son
  pourcentage CPU par rapport à cette valeur (cible 70 %).
- limits CPU : environ 4 fois le request, pour tenir un pic le temps que les
  nouveaux pods démarrent.
- limits RAM : large par rapport à ce qu'on a mesuré, pour éviter les OOMKill.
- paiement a un peu plus de CPU à cause des appels HTTPS vers Stripe.
