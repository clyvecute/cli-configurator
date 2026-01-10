package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cli-config-linter/linter"
)

type lintRequest struct {
	Config         string `json:"config"`
	Strict         bool   `json:"strict"`
	FixSuggestions bool   `json:"fixSuggestions"`
}

type lintResponse struct {
	Issues      []linter.Issue `json:"issues"`
	Strict      bool            `json:"strict"`
	Fatal       bool            `json:"fatal"`
	GeneratedAt time.Time       `json:"generatedAt"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	keys := loadAPIKeys()
	if len(keys) == 0 {
		fmt.Println("WARNING: no API keys configured; requests will be rejected. Set CONFIG_LINTER_API_KEY.")
	}

	mux := http.NewServeMux()
	mux.Handle("/lint", withCORS(withAPIKeyAuth(keys, http.HandlerFunc(lintHandler))))

	addr := ":8080"
	if port := os.Getenv("LINTER_SERVER_PORT"); port != "" {
		addr = ":" + port
	}

	fmt.Printf("Listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("server error: %v\n", err)
	}
}

func lintHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var req lintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Config) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "config body is required"})
		return
	}

	issues, err := linter.LintBytes([]byte(req.Config))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	fatal := false
	for _, issue := range issues {
		if issue.Severity == linter.SeverityError || (req.Strict && issue.Severity == linter.SeverityWarning) {
			fatal = true
			break
		}
	}

	resp := lintResponse{
		Issues:      issues,
		Strict:      req.Strict,
		Fatal:       fatal,
		GeneratedAt: time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func loadAPIKeys() map[string]struct{} {
	keys := make(map[string]struct{})
	raw := os.Getenv("CONFIG_LINTER_API_KEY")
	for _, part := range strings.Split(raw, ",") {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}
	return keys
}

func withAPIKeyAuth(allowed map[string]struct{}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(allowed) == 0 {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "API key not configured"})
			return
		}
		key := r.Header.Get("X-API-Key")
		if key == "" {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				key = strings.TrimSpace(auth[7:])
			}
		}
		if _, ok := allowed[key]; !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid api key"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
