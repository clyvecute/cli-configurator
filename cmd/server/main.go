package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"cli-config-linter/linter"
)

// -- Configuration --

type Config struct {
	Port      string
	APIKeys   map[string]struct{}
	StaticDir string
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("LINTER_SERVER_PORT")
	}
	if port == "" {
		port = "8080"
	}

	keys := make(map[string]struct{})
	rawKeys := os.Getenv("CONFIG_LINTER_API_KEY")
	for _, k := range strings.Split(rawKeys, ",") {
		trimmed := strings.TrimSpace(k)
		if trimmed != "" {
			keys[trimmed] = struct{}{}
		}
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	return Config{
		Port:      port,
		APIKeys:   keys,
		StaticDir: staticDir,
	}
}

// -- API Models --

type LintRequest struct {
	Config         string `json:"config"`
	Strict         bool   `json:"strict"`
	FixSuggestions bool   `json:"fixSuggestions"`
}

type LintResponse struct {
	Issues      []linter.Issue `json:"issues"`
	Strict      bool           `json:"strict"`
	Fatal       bool           `json:"fatal"`
	GeneratedAt time.Time      `json:"generatedAt"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// -- Main --

var startTime time.Time

func main() {
	startTime = time.Now()
	
	// 1. Logging Setup (Structured JSON Logger)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := loadConfig()

	if len(cfg.APIKeys) == 0 {
		logger.Warn("security_alert: no API keys configured. service is unprotected.")
	}

	// 2. Router Setup
	mux := http.NewServeMux()

	// 2a. Public Endpoints
	mux.HandleFunc("GET /health", handleHealth)

	// 2b. Private Endpoints (Secured)
	// We handle auth manually in the chain for granular control
	secured := withAPIKeyAuth(cfg.APIKeys, http.HandlerFunc(handleLint))
	mux.Handle("POST /lint", secured)

	// 2c. Static Assets
	if info, err := os.Stat(cfg.StaticDir); err == nil && info.IsDir() {
		logger.Info("static_files_enabled", "directory", cfg.StaticDir)
		// Serve static files (HTML/JS/CSS)
		// We wrap this with minimal middlewares (CORS etc) if needed, 
		// but usually static files are public.
		fs := http.FileServer(http.Dir(cfg.StaticDir))
		mux.Handle("GET /", fs)
	} else {
		logger.Warn("static_files_disabled", "reason", "directory not found", "path", cfg.StaticDir)
	}

	// 3. Global Middleware Chain (Recovery -> Logging -> CORS -> Mux)
	finalHandler := withRecovery(withLogging(withCORS(mux)))

	// 4. Server Start
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      finalHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("server_starting", "port", cfg.Port, "env", "production")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server_failed", "error", err)
		os.Exit(1)
	}
}

// -- Handlers --

func handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
		Uptime:  time.Since(startTime).String(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleLint(w http.ResponseWriter, r *http.Request) {
	// 1. Decode
	var req LintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("bad_request", "error", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON body"})
		return
	}

	if strings.TrimSpace(req.Config) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Config content cannot be empty"})
		return
	}

	// 2. Logic (Core Linter)
	issues, err := linter.LintBytes([]byte(req.Config))
	if err != nil {
		slog.Error("linter_internal_error", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal linter error"})
		return
	}

	// 3. Process Results
	fatal := false
	for _, issue := range issues {
		if issue.Severity == linter.SeverityError || (req.Strict && issue.Severity == linter.SeverityWarning) {
			fatal = true
			break
		}
	}

	// 4. Respond
	resp := LintResponse{
		Issues:      issues,
		Strict:      req.Strict,
		Fatal:       fatal,
		GeneratedAt: time.Now().UTC(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// -- Middleware --

// withRecovery handles panics gracefully
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic_recovered", "error", err, "stack", string(debug.Stack()))
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal Server Error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withLogging logs request details
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap ResponseWriter to capture status code
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		
		next.ServeHTTP(ww, r)
		
		duration := time.Since(start)
		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration_ms", duration.Milliseconds(),
			"ip", r.RemoteAddr,
		)
	})
}

// withCORS adds Cross-Origin Resource Sharing headers
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// withAPIKeyAuth enforces security
func withAPIKeyAuth(allowedKeys map[string]struct{}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if no keys configured (dev mode warning already logged)
		if len(allowedKeys) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("X-API-Key")
		if key == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				key = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			}
		}

		if _, ok := allowedKeys[key]; !ok {
			slog.Warn("auth_failed", "ip", r.RemoteAddr)
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized: Invalid or missing API Key"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// -- Helpers --

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("json_encode_fail", "error", err)
	}
}
