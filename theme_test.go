package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminThemeAssetAndCSP(t *testing.T) {
	a := &adminServer{}
	h := a.handler()

	themeReq := httptest.NewRequest(http.MethodGet, "http://admin.local/theme.css", nil)
	themeRec := httptest.NewRecorder()
	h.ServeHTTP(themeRec, themeReq)
	if themeRec.Code != http.StatusOK {
		t.Fatalf("theme status = %d, want 200", themeRec.Code)
	}
	if ct := themeRec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Fatalf("theme content-type = %q", ct)
	}
	if body := themeRec.Body.String(); !strings.Contains(body, "--cg-green: #247a4d") {
		t.Fatal("served theme is missing canonical green token")
	}

	rootReq := httptest.NewRequest(http.MethodGet, "http://admin.local/", nil)
	rootRec := httptest.NewRecorder()
	h.ServeHTTP(rootRec, rootReq)
	if rootRec.Code != http.StatusOK {
		t.Fatalf("root status = %d, want 200", rootRec.Code)
	}
	if !strings.Contains(rootRec.Body.String(), `href="/theme.css"`) {
		t.Fatal("shipping admin page does not link theme.css")
	}
	csp := rootRec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "style-src 'self' 'unsafe-inline'") {
		t.Fatalf("CSP does not allow same-origin theme stylesheet: %q", csp)
	}
}
