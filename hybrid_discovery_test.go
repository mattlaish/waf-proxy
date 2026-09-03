package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func fieldNames(fields []DiscoveredField) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = f.Name
	}
	return out
}

func TestPassiveFieldsURLAndJSONKeepNamesOnly(t *testing.T) {
	formBody := []byte("username=alice&password=super-secret&bad%20name=x")
	form := passiveFields(http.MethodPost, "/login", "application/x-www-form-urlencoded", formBody)
	if got, want := fieldNames(form), []string{"username", "password"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("form fields = %#v, want %#v", got, want)
	}
	jsonBody := []byte(`{"user_id":"alice","password":"super-secret","profile":{"role":"admin"}}`)
	fields := passiveFields(http.MethodPatch, "/account", "application/json", jsonBody)
	if got, want := fieldNames(fields), []string{"user_id", "password", "profile"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON fields = %#v, want %#v", got, want)
	}
	encoded, _ := json.Marshal(append(form, fields...))
	if bytes.Contains(encoded, []byte("alice")) || bytes.Contains(encoded, []byte("super-secret")) {
		t.Fatalf("field values leaked into discovery state: %s", encoded)
	}
}

func TestPassiveFieldsMultipartNamesAndFileType(t *testing.T) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("comment", "private text")
	part, err := w.CreateFormFile("attachment", "secret.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("private file contents"))
	_ = w.Close()
	fields := passiveFields(http.MethodPost, "/upload", w.FormDataContentType(), body.Bytes())
	if got, want := fieldNames(fields), []string{"comment", "attachment"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("multipart fields = %#v, want %#v", got, want)
	}
	if fields[1].Type != "file" {
		t.Fatalf("file type not detected: %#v", fields[1])
	}
}

func TestPassiveDiscoveryWrapRestoresBodyAndAddsContext(t *testing.T) {
	original := "username=alice&password=secret"
	var gotBody string
	var gotFields []DiscoveredField
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotFields = passiveFieldsFromRequest(r)
	})
	r := httptest.NewRequest(http.MethodPost, "http://waf.local/login", strings.NewReader(original))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	requestBodyPrefixWrap(false, true, passiveDiscoveryWrap(true, next)).ServeHTTP(httptest.NewRecorder(), r)
	if gotBody != original {
		t.Fatalf("body changed: got %q want %q", gotBody, original)
	}
	if got, want := fieldNames(gotFields), []string{"username", "password"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("context fields = %#v, want %#v", got, want)
	}
}

func TestPassiveDiscoveryDisabledLeavesBodyUntouched(t *testing.T) {
	original := "username=alice&password=secret"
	var gotBody string
	var gotFields []DiscoveredField
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotFields = passiveFieldsFromRequest(r)
	})
	r := httptest.NewRequest(http.MethodPost, "http://waf.local/login", strings.NewReader(original))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	requestBodyPrefixWrap(false, false, passiveDiscoveryWrap(false, next)).ServeHTTP(httptest.NewRecorder(), r)
	if gotBody != original {
		t.Fatalf("body changed: got %q want %q", gotBody, original)
	}
	if len(gotFields) != 0 {
		t.Fatalf("disabled passive discovery added fields: %#v", gotFields)
	}
}

func TestRequestBodyPrefixSharedByPassiveAndAIConsumer(t *testing.T) {
	original := `{"user_id":"alice","profile":{"role":"admin"}}`
	var gotBody string
	var gotPrefix []byte
	var gotFields []DiscoveredField
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotPrefix = requestBodyPrefixFromRequest(r)
		gotFields = passiveFieldsFromRequest(r)
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	})
	r := httptest.NewRequest(http.MethodPost, "http://waf.local/account", strings.NewReader(original))
	r.Header.Set("Content-Type", "application/json")
	requestBodyPrefixWrap(true, true, passiveDiscoveryWrap(true, next)).ServeHTTP(httptest.NewRecorder(), r)
	if gotBody != original {
		t.Fatalf("body changed: got %q want %q", gotBody, original)
	}
	if string(gotPrefix) != original {
		t.Fatalf("shared prefix = %q, want %q", gotPrefix, original)
	}
	if got, want := fieldNames(gotFields), []string{"user_id", "profile"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("context fields = %#v, want %#v", got, want)
	}
}

func TestPassiveJSONScannerSkipsLargeValuesWithoutLosingKeys(t *testing.T) {
	body := []byte(`{"first":"` + strings.Repeat("x", 60000) + `","nested":{"items":[1,2,{"x":"y"}]},"last":true}`)
	fields := passiveFields(http.MethodPost, "/api", "application/json", body)
	if got, want := fieldNames(fields), []string{"first", "nested", "last"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON fields = %#v, want %#v", got, want)
	}
}

func TestPassiveFieldsReachRecorderOnlyAfterBackendResponse(t *testing.T) {
	var backendBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		backendBody = string(b)
		w.WriteHeader(http.StatusSeeOther)
	}))
	defer backend.Close()
	target, _ := url.Parse(backend.URL)
	pool := &poolRuntime{name: "test", method: "round_robin", members: []*memberRuntime{{target: target, weight: 1}}}
	var recorded []DiscoveredField
	proxy := buildProxy(pool, SiteConfig{}, Config{BackendTimeoutSec: 5},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		func(_, _, _, _ string, _ int, fields []DiscoveredField) { recorded = fields })
	original := "username=alice&password=secret"
	r := httptest.NewRequest(http.MethodPost, "http://waf.local/login", strings.NewReader(original))
	r.RemoteAddr = "192.0.2.10:1234"
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	requestBodyPrefixWrap(false, true, passiveDiscoveryWrap(true, proxy)).ServeHTTP(httptest.NewRecorder(), r)
	if backendBody != original {
		t.Fatalf("backend body changed: got %q want %q", backendBody, original)
	}
	if got, want := fieldNames(recorded), []string{"username", "password"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("recorded fields = %#v, want %#v", got, want)
	}
}

func TestMergePassiveAndCrawledFields(t *testing.T) {
	passive := []DiscoveredField{{Name: "password", Method: "POST", Action: "/login", DiscoverySource: fieldSourcePassive}}
	crawled := []DiscoveredField{{Name: "password", Type: "password", Method: "POST", Action: "", Required: true, DiscoverySource: fieldSourceCrawled}}
	got := mergeDiscoveredFields(passive, crawled, "/login")
	if len(got) != 1 || got[0].DiscoverySource != fieldSourceBoth || got[0].Type != "password" || !got[0].Required {
		t.Fatalf("unexpected merged field: %#v", got)
	}
}

func TestCrawlerHTMLIndexesFieldsByFormAction(t *testing.T) {
	store := newSignalStore()
	html := `<form method="post" action="/login"><input name="username"><input type="password" name="password" required></form>`
	fields := store.noteContent("site", "/", "text/html", html)
	if len(fields) != 2 {
		t.Fatalf("crawler found %d fields", len(fields))
	}
	actionFields := store.discoveredFields("site", "/login")
	if got, want := fieldNames(actionFields), []string{"username", "password"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("action fields = %#v, want %#v", got, want)
	}
	store.noteRequestShape("site", "/login", "POST", "", "application/x-www-form-urlencoded",
		[]DiscoveredField{{Name: "password", Method: "POST", Action: "/login", DiscoverySource: fieldSourcePassive}})
	merged := store.discoveredFields("site", "/login")
	if merged[1].DiscoverySource != fieldSourceBoth {
		t.Fatalf("source was not merged: %#v", merged)
	}
}

func TestSeenGETPathsSeedCrawlerSafely(t *testing.T) {
	maps := newSiteMaps(100)
	maps.record("site", http.MethodGet, "/login", http.StatusOK, srcObserved)
	maps.record("site", http.MethodPost, "/submit", http.StatusSeeOther, srcObserved)
	maps.record("site", http.MethodGet, "/logout", http.StatusOK, srcObserved)
	got := maps.forSite("site").seenGETPaths()
	if !reflect.DeepEqual(got, []string{"/login"}) {
		t.Fatalf("crawl seeds = %#v", got)
	}
}

func TestDiscoverFormActionsNormalizesAndRejectsExternal(t *testing.T) {
	html := `<form method="post" action="login"></form><form action="https://evil.example/collect"></form><form action="//evil.example/also"></form><form action="javascript:alert(1)"></form>`
	got := discoverFormActions(html, "/account/")
	if !reflect.DeepEqual(got, []string{"/account/login"}) {
		t.Fatalf("actions = %#v", got)
	}
}
