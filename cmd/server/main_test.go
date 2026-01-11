package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// We need to export/refactor handler logic to test it easily,
// but for this portfolio, we will simulate the handler logic in test
// or rely on the fact that we can access public functions if we move handlers.
//
// Since main.go functions are main package, we can test them if we are in package main_test
// or package main.

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", data["status"])
	}
}

func TestLintHandler_Auth(t *testing.T) {
	// Setup middleware chain for testing auth
	keys := map[string]struct{}{"secret": {}}
	handler := withAPIKeyAuth(keys, http.HandlerFunc(handleLint))

	req := httptest.NewRequest("POST", "/lint", nil)
	// No Auth Header
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}

	// Correct Auth
	req = httptest.NewRequest("POST", "/lint", strings.NewReader(`{"config": "metadata:\n  name: test"}`))
	req.Header.Set("X-API-Key", "secret")
	req.Header.Set("Content-Type", "application/json")
	
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Since body is just partial config, it might return 200 with issues, or 400 if strictly invalid json.
	// But auth should pass.
	if w.Code == http.StatusUnauthorized {
		t.Errorf("expected authorized request, got 401")
	}
}

func TestLintHandler_Logic(t *testing.T) {
	configPayload := LintRequest{
		Config: "metadata:\n  name: unit-test\n  env: dev\nsettings:\n  replicas: 1\n  timeout: 10\nfeatures:\n  - name: f1\n    enabled: true",
		Strict: true,
	}
	body, _ := json.Marshal(configPayload)

	req := httptest.NewRequest("POST", "/lint", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleLint(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	var result LintResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(result.Issues) > 0 {
		t.Errorf("expected 0 issues for valid config, got %d", len(result.Issues))
	}
}
