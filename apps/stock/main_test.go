package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func testMux() *http.ServeMux {
	var r atomic.Bool
	r.Store(true)
	return newMux("pod-test", "test", &r)
}

func TestListeProduits(t *testing.T) {
	rec := httptest.NewRecorder()
	testMux().ServeHTTP(rec, httptest.NewRequest("GET", "/api/produits", nil))
	if rec.Code != 200 {
		t.Fatalf("code %d", rec.Code)
	}
	if rec.Header().Get("X-Servi-Par") != "pod-test" {
		t.Fatal("en-tête X-Servi-Par absent")
	}
	var body struct{ Produits []Produit }
	json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Produits) != len(catalogue) {
		t.Fatalf("attendu %d produits, reçu %d", len(catalogue), len(body.Produits))
	}
}

func TestProduitInconnu(t *testing.T) {
	rec := httptest.NewRecorder()
	testMux().ServeHTTP(rec, httptest.NewRequest("GET", "/api/produits/nexistepas", nil))
	if rec.Code != 404 {
		t.Fatalf("attendu 404, reçu %d", rec.Code)
	}
}
