package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

type Scenario struct {
	Name                string
	Method              string
	URI                 string
	ContentType         string
	Body                []byte
	ResponseContentType string
	ResponseBody        []byte
}

func scenarios(responseBytes int) map[string]Scenario {
	if responseBytes < 0 {
		responseBytes = 0
	}
	out := map[string]Scenario{}
	out["clean-get"] = Scenario{
		Name: "clean-get", Method: http.MethodGet, URI: "/benchmark/clean",
		ResponseContentType: "application/json", ResponseBody: sizedJSON(responseBytes, "ok"),
	}
	for _, kb := range []int{1, 16, 64, 256} {
		name := fmt.Sprintf("json-%dk", kb)
		out[name] = Scenario{
			Name: name, Method: http.MethodPost, URI: "/benchmark/json", ContentType: "application/json",
			Body: sizedJSON(kb*1024, "clean"), ResponseContentType: "application/json", ResponseBody: sizedJSON(responseSizeFor(responseBytes, kb*1024), "response"),
		}
	}
	out["sqli-64k"] = Scenario{
		Name: "sqli-64k", Method: http.MethodPost, URI: "/benchmark/login", ContentType: "application/json",
		Body:                maliciousJSON(64*1024, `admin' OR 1=1 UNION SELECT username,password FROM users--`),
		ResponseContentType: "application/json", ResponseBody: sizedJSON(responseSizeFor(responseBytes, 8*1024), "response"),
	}
	out["xss-64k"] = Scenario{
		Name: "xss-64k", Method: http.MethodPost, URI: "/benchmark/comment", ContentType: "application/json",
		Body:                maliciousJSON(64*1024, `<script>alert(1)</script><img src=x onerror=alert(document.domain)>`),
		ResponseContentType: "application/json", ResponseBody: sizedJSON(responseSizeFor(responseBytes, 8*1024), "response"),
	}
	out["traversal"] = Scenario{
		Name: "traversal", Method: http.MethodGet, URI: "/benchmark/download?file=../../../../etc/passwd&path=%2e%2e%2f%2e%2e%2fetc%2fshadow",
		ResponseContentType: "application/json", ResponseBody: sizedJSON(responseSizeFor(responseBytes, 1024), "response"),
	}
	return out
}

func responseSizeFor(override, fallback int) int {
	if override > 0 {
		return override
	}
	return fallback
}

func sizedJSON(target int, marker string) []byte {
	if target <= 0 {
		return nil
	}
	prefix := []byte(`{"kind":"` + marker + `","payload":"`)
	suffix := []byte(`"}`)
	if target <= len(prefix)+len(suffix) {
		return append(append([]byte{}, prefix...), suffix...)
	}
	b := make([]byte, 0, target)
	b = append(b, prefix...)
	b = append(b, bytes.Repeat([]byte("a"), target-len(prefix)-len(suffix))...)
	b = append(b, suffix...)
	return b
}

func maliciousJSON(target int, attack string) []byte {
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(attack)
	prefix := []byte(`{"attack":"` + escaped + `","payload":"`)
	suffix := []byte(`"}`)
	if target < len(prefix)+len(suffix) {
		target = len(prefix) + len(suffix)
	}
	b := make([]byte, 0, target)
	b = append(b, prefix...)
	b = append(b, bytes.Repeat([]byte("A"), target-len(prefix)-len(suffix))...)
	b = append(b, suffix...)
	return b
}

func chooseScenarios(name string, responseBytes int) ([]Scenario, error) {
	all := scenarios(responseBytes)
	if name == "all" {
		order := []string{"clean-get", "json-1k", "json-16k", "json-64k", "json-256k", "sqli-64k", "xss-64k", "traversal"}
		out := make([]Scenario, 0, len(order))
		for _, n := range order {
			out = append(out, all[n])
		}
		return out, nil
	}
	parts := strings.Split(name, ",")
	out := make([]Scenario, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		sc, ok := all[p]
		if !ok {
			return nil, fmt.Errorf("unknown scenario %q; available: %s", p, strings.Join(sortedScenarios(all), ", "))
		}
		out = append(out, sc)
	}
	return out, nil
}
