package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

const empreinte = "MIP-04-11-2026"

type Produit struct {
	ID           string `json:"id"`
	Nom          string `json:"nom"`
	Description  string `json:"description"`
	PrixCentimes int64  `json:"prix_centimes"`
	Stock        int    `json:"stock"`
}

var catalogue = []Produit{
	{"beret", "Béret en laine", "Laine des Vosges, taille unique", 2490, 42},
	{"mariniere", "Marinière", "Coton bio, coupe droite", 3990, 18},
	{"mug", "Mug émaillé", "Le mug du café de 8h", 1290, 120},
	{"opinel", "Couteau de poche", "Lame inox, manche hêtre", 1890, 0},
	{"mirabelles", "Confiture de mirabelles", "De Lorraine, 340 g", 690, 75},
}

func indexCatalogue() map[string]Produit {
	m := make(map[string]Produit, len(catalogue))
	for _, p := range catalogue {
		m[p.ID] = p
	}
	return m
}

func newMux(pod, env string, ready *atomic.Bool) *http.ServeMux {
	parID := indexCatalogue()
	mux := http.NewServeMux()

	withPod := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Servi-Par", pod)
			w.Header().Set("X-Empreinte", empreinte)
			h(w, r)
		}
	}

	mux.HandleFunc("GET /api/produits", withPod(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"env": env, "pod": pod, "produits": catalogue})
	}))

	mux.HandleFunc("GET /api/produits/{id}", withPod(func(w http.ResponseWriter, r *http.Request) {
		p, ok := parID[r.PathValue("id")]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"erreur": "produit inconnu"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	}))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "arrêt en cours", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
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
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("app", "stock-mip-04-11-2026", "env", env, "pod", pod)

	var ready atomic.Bool
	ready.Store(true)
	srv := &http.Server{
		Addr:              ":" + getenv("PORT", "8080"),
		Handler:           newMux(pod, env, &ready),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("démarrage", "addr", srv.Addr, "produits", len(catalogue))
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
