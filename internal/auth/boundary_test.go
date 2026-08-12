package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type validator struct{}

func (validator) Validate(context.Context, string) (Principal, error) {
	return Principal{Subject: "test"}, nil
}

func TestBoundaryRejectsMissingBearerToken(t *testing.T) {
	h := Boundary{Enabled: true, Validator: validator{}}.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", res.Code)
	}
}

func TestBoundaryAddsPrincipal(t *testing.T) {
	h := Boundary{Enabled: true, Validator: validator{}}.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := PrincipalFromContext(r.Context()); !ok || p.Subject != "test" {
			t.Fatal("principal missing")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d", res.Code)
	}
}
