import http from 'k6/http';
import { check } from 'k6';

const RPS_MAX = parseInt(__ENV.RPS_MAX || '800');
const CIBLE = __ENV.CIBLE || 'all';
const PLATEAU = __ENV.PLATEAU || '3m';
const FP = 'mip-04-11-2026';

const cibles = {
  vitrine:  `http://vitrine-${FP}:8080/`,
  stock:    `http://stock-${FP}:8080/api/produits`,
  paiement: `http://paiement-${FP}:8080/`,
};

function scenario(nom) {
  return {
    executor: 'ramping-arrival-rate',
    exec: nom,
    startRate: 10,
    timeUnit: '1s',
    preAllocatedVUs: 50,
    maxVUs: 300,
    stages: [
      { duration: '30s', target: Math.round(RPS_MAX * 0.2) },
      { duration: '60s', target: RPS_MAX },
      { duration: PLATEAU, target: RPS_MAX },
      { duration: '30s', target: 0 },
    ],
  };
}

const actifs = CIBLE === 'all' ? Object.keys(cibles) : [CIBLE];
export const options = {
  scenarios: Object.fromEntries(actifs.map((n) => [n, scenario(n)])),
  thresholds: {
    http_req_failed: ['rate<0.05'],
    'http_req_duration{scenario:vitrine}': ['p(95)<800'],
    'http_req_duration{scenario:stock}': ['p(95)<500'],
    'http_req_duration{scenario:paiement}': ['p(95)<500'],
  },
};

function appel(nom) {
  const r = http.get(cibles[nom], { tags: { app: nom } });
  check(r, { [`${nom} 200`]: (x) => x.status === 200 });
}

export function vitrine()  { appel('vitrine'); }
export function stock()    { appel('stock'); }
export function paiement() { appel('paiement'); }
