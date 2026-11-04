// vitrine-mip-04-11-2026 : site qui affiche les produits (via stock)
// et envoie vers paiement pour acheter
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
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

const empreinte = "MIP-04-11-2026"

//go:embed templates/index.html
var fichiers embed.FS

type Produit struct {
	ID           string `json:"id"`
	Nom          string `json:"nom"`
	Description  string `json:"description"`
	PrixCentimes int64  `json:"prix_centimes"`
	Stock        int    `json:"stock"`
}

func (p Produit) Prix() string {
	return fmt.Sprintf("%d,%02d €", p.PrixCentimes/100, p.PrixCentimes%100)
}

type page struct {
	Env, Pod, PodStock, URLPaiement, Erreur, Empreinte string
	Produits                                           []Produit
}

type app struct {
	stockURL, urlPaiement, env, pod string
	client                          *http.Client
	tpl                             *template.Template
	log                             *slog.Logger
}

func (a *app) catalogue(ctx context.Context) ([]Produit, string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", a.stockURL+"/api/produits", nil)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("stock a répondu %d", resp.StatusCode)
	}
	var body struct{ Produits []Produit }
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, "", err
	}
	return body.Produits, resp.Header.Get("X-Servi-Par"), nil
}

func (a *app) accueil(w http.ResponseWriter, r *http.Request) {
	p := page{Env: a.env, Pod: a.pod, URLPaiement: a.urlPaiement, Empreinte: empreinte}
	produits, podStock, err := a.catalogue(r.Context())
	if err != nil {
		// si stock ne répond pas, la page s'affiche quand même
		a.log.Warn("catalogue indisponible", "err", err)
		p.Erreur = "Catalogue momentanément indisponible, réessayez dans un instant."
	}
	p.Produits, p.PodStock = produits, podStock
	w.Header().Set("X-Servi-Par", a.pod)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tpl.Execute(w, p); err != nil {
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
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("app", "vitrine-mip-04-11-2026", "env", env, "pod", pod)

	a := &app{
		stockURL:    getenv("STOCK_URL", "http://localhost:8081"),
		urlPaiement: "http://" + getenv("HOST_PAIEMENT", "localhost:8082"),
		env:         env,
		pod:         pod,
		client:      &http.Client{Timeout: 2 * time.Second},
		tpl:         template.Must(template.ParseFS(fichiers, "templates/index.html")),
		log:         log,
	}

	var ready atomic.Bool
	ready.Store(true)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.accueil)
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
		log.Info("démarrage", "addr", srv.Addr, "stock", a.stockURL, "paiement", a.urlPaiement)
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
