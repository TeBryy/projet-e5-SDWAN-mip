package main

import (
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPrix(t *testing.T) {
	if got := (Produit{PrixCentimes: 690}).Prix(); got != "6,90 €" {
		t.Fatalf("Prix() = %q", got)
	}
}

// Si stock est injoignable, la vitrine doit rester en ligne (mode dégradé).
func TestModeDegrade(t *testing.T) {
	a := &app{stockURL: "http://127.0.0.1:1", client: &http.Client{Timeout: 200 * time.Millisecond},
		tpl: template.Must(template.ParseFS(fichiers, "templates/index.html")),
		log: slog.New(slog.NewTextHandler(io.Discard, nil)), env: "test"}
	rec := httptest.NewRecorder()
	a.accueil(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "indisponible") {
		t.Fatalf("mode dégradé attendu, reçu %d", rec.Code)
	}
}
