// paiement-mip-04-11-2026 : paiement avec Stripe Checkout (mode test)
// le prix est toujours repris depuis stock, pas depuis le navigateur
// et l'appli refuse de démarrer avec une clé live
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const empreinte = "MIP-04-11-2026"

//go:embed templates/*.html
var fichiers embed.FS

type Produit struct {
	ID           string `json:"id"`
	Nom          string `json:"nom"`
	PrixCentimes int64  `json:"prix_centimes"`
	Stock        int    `json:"stock"`
}

type app struct {
	stripeKey, stripeAPI, stockURL, publicURL, vitrineURL, env, pod string
	client                                                          *http.Client
	tpl                                                             *template.Template
	log                                                             *slog.Logger
}

// cleValide accepte seulement les clés de test
func cleValide(k string) bool {
	return strings.HasPrefix(k, "sk_test_") || strings.HasPrefix(k, "rk_test_")
}

func (a *app) produit(ctx context.Context, id string) (Produit, error) {
	var p Produit
	req, _ := http.NewRequestWithContext(ctx, "GET", a.stockURL+"/api/produits/"+url.PathEscape(id), nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return p, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return p, fmt.Errorf("produit %q introuvable (stock: %d)", id, resp.StatusCode)
	}
	return p, json.NewDecoder(resp.Body).Decode(&p)
}

// stripe appelle l'API Stripe
func (a *app) stripe(ctx context.Context, method, path string, form url.Values, out any) error {
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	req, _ := http.NewRequestWithContext(ctx, method, a.stripeAPI+path, body)
	req.Header.Set("Authorization", "Bearer "+a.stripeKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct{ Error struct{ Message string } }
		json.NewDecoder(resp.Body).Decode(&e)
		return fmt.Errorf("stripe %d: %s", resp.StatusCode, e.Error.Message)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (a *app) checkout(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("produit")
	p, err := a.produit(r.Context(), id)
	if err != nil {
		a.log.Warn("produit refusé", "produit", id, "err", err)
		http.Error(w, "Produit inconnu", http.StatusBadRequest)
		return
	}
	if p.Stock <= 0 {
		http.Error(w, "Produit en rupture de stock", http.StatusConflict)
		return
	}

	f := url.Values{}
	f.Set("mode", "payment")
	f.Set("success_url", a.publicURL+"/succes?session_id={CHECKOUT_SESSION_ID}")
	f.Set("cancel_url", a.publicURL+"/annule")
	f.Set("line_items[0][quantity]", "1")
	f.Set("line_items[0][price_data][currency]", "eur")
	f.Set("line_items[0][price_data][unit_amount]", fmt.Sprint(p.PrixCentimes))
	f.Set("line_items[0][price_data][product_data][name]", p.Nom)
	// metadata visibles dans le dashboard Stripe (env + pod)
	for _, prefix := range []string{"metadata", "payment_intent_data[metadata]"} {
		f.Set(prefix+"[empreinte]", empreinte)
		f.Set(prefix+"[env]", a.env)
		f.Set(prefix+"[pod]", a.pod)
		f.Set(prefix+"[produit]", p.ID)
	}
	f.Set("payment_intent_data[description]", fmt.Sprintf("Comptoir Lorrain %s - %s", a.env, p.Nom))

	var session struct{ ID, URL string }
	if err := a.stripe(r.Context(), "POST", "/v1/checkout/sessions", f, &session); err != nil {
		a.log.Error("création session Stripe", "err", err)
		http.Error(w, "Paiement indisponible", http.StatusBadGateway)
		return
	}
	a.log.Info("session Checkout créée", "session", session.ID, "produit", p.ID, "montant", p.PrixCentimes)
	http.Redirect(w, r, session.URL, http.StatusSeeOther)
}

func (a *app) succes(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("session_id")
	if !strings.HasPrefix(id, "cs_") {
		http.Error(w, "session invalide", http.StatusBadRequest)
		return
	}
	var s struct {
		ID            string `json:"id"`
		PaymentStatus string `json:"payment_status"`
		AmountTotal   int64  `json:"amount_total"`
		Customer      struct {
			Email string `json:"email"`
		} `json:"customer_details"`
	}
	if err := a.stripe(r.Context(), "GET", "/v1/checkout/sessions/"+url.PathEscape(id), nil, &s); err != nil {
		a.log.Error("lecture session", "err", err)
		http.Error(w, "Impossible de vérifier le paiement", http.StatusBadGateway)
		return
	}
	a.log.Info("retour paiement", "session", s.ID, "statut", s.PaymentStatus, "montant", s.AmountTotal)
	a.render(w, "succes.html", map[string]any{
		"Session": s.ID, "Statut": s.PaymentStatus, "Email": s.Customer.Email,
		"Montant": fmt.Sprintf("%d,%02d €", s.AmountTotal/100, s.AmountTotal%100),
	})
}

func (a *app) render(w http.ResponseWriter, name string, data map[string]any) {
	data["Env"], data["Pod"], data["Vitrine"], data["Empreinte"] = a.env, a.pod, a.vitrineURL, empreinte
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Servi-Par", a.pod)
	if err := a.tpl.ExecuteTemplate(w, name, data); err != nil {
		a.log.Error("template", "err", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	pod, _ := os.Hostname()
	env := getenv("APP_ENV", "local")
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("app", "paiement-mip-04-11-2026", "env", env, "pod", pod)

	key := os.Getenv("STRIPE_SECRET_KEY")
	if !cleValide(key) {
		log.Error("STRIPE_SECRET_KEY absente ou non-test (sk_test_/rk_test_ attendu) : arrêt")
		os.Exit(1)
	}

	a := &app{
		stripeKey:  key,
		stripeAPI:  getenv("STRIPE_API", "https://api.stripe.com"),
		stockURL:   getenv("STOCK_URL", "http://localhost:8081"),
		publicURL:  "http://" + getenv("HOST_PAIEMENT", "localhost:8082"),
		vitrineURL: "http://" + getenv("HOST_VITRINE", "localhost:8080"),
		env:        env,
		pod:        pod,
		client:     &http.Client{Timeout: 10 * time.Second},
		tpl:        template.Must(template.ParseFS(fichiers, "templates/*.html")),
		log:        log,
	}

	var ready atomic.Bool
	ready.Store(true)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { a.render(w, "accueil.html", map[string]any{}) })
	mux.HandleFunc("POST /checkout", a.checkout)
	mux.HandleFunc("GET /succes", a.succes)
	mux.HandleFunc("GET /annule", func(w http.ResponseWriter, r *http.Request) { a.render(w, "annule.html", map[string]any{}) })
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "arrêt en cours", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})

	srv := &http.Server{Addr: ":" + getenv("PORT", "8080"), Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info("démarrage", "addr", srv.Addr, "stripe", a.stripeAPI, "cle", key[:8]+"…")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("serveur arrêté", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	ready.Store(false)
	log.Info("SIGTERM reçu, arrêt propre")
	time.Sleep(3 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
