package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCleValide(t *testing.T) {
	for k, ok := range map[string]bool{
		"sk_test_abc": true, "rk_test_abc": true,
		"sk_live_abc": false, "rk_live_abc": false, "": false,
	} {
		if cleValide(k) != ok {
			t.Errorf("cleValide(%q) = %v", k, !ok)
		}
	}
}

// Le prix envoyé à Stripe doit venir du service stock, jamais du client.
func TestCheckoutUtiliseLePrixDuStock(t *testing.T) {
	stock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Produit{ID: "mug", Nom: "Mug", PrixCentimes: 1290, Stock: 3})
	}))
	defer stock.Close()

	var recu url.Values
	stripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_x" {
			t.Error("clé Stripe absente")
		}
		b, _ := io.ReadAll(r.Body)
		recu, _ = url.ParseQuery(string(b))
		json.NewEncoder(w).Encode(map[string]string{"id": "cs_test_1", "url": "https://checkout.stripe.com/c/pay/cs_test_1"})
	}))
	defer stripe.Close()

	a := &app{stripeKey: "sk_test_x", stripeAPI: stripe.URL, stockURL: stock.URL,
		publicURL: "http://paiement.test", env: "dev-mip-04-11-2026", pod: "p1",
		client: http.DefaultClient, log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		tpl: template.Must(template.ParseFS(fichiers, "templates/*.html"))}

	// Un client malveillant tente d'imposer un prix : il doit être ignoré.
	req := httptest.NewRequest("POST", "/checkout", strings.NewReader("produit=mug&prix=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	a.checkout(rec, req)

	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "https://checkout.stripe.com/") {
		t.Fatalf("redirection attendue vers Stripe, reçu %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if got := recu.Get("line_items[0][price_data][unit_amount]"); got != "1290" {
		t.Fatalf("prix envoyé à Stripe = %s, attendu 1290", got)
	}
	if recu.Get("metadata[env]") != "dev-mip-04-11-2026" || recu.Get("payment_intent_data[metadata][empreinte]") != empreinte {
		t.Fatal("métadonnées env/empreinte manquantes")
	}
}

func TestCheckoutRupture(t *testing.T) {
	stock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Produit{ID: "opinel", PrixCentimes: 1890, Stock: 0})
	}))
	defer stock.Close()
	a := &app{stockURL: stock.URL, client: http.DefaultClient, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	rec := httptest.NewRecorder()
	a.checkout(rec, httptest.NewRequest("POST", "/checkout?produit=opinel", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("attendu 409, reçu %d", rec.Code)
	}
}
