# Sentinel: Infrastructure Configuration Linter

> **Infrastructure assurance suite for the modern stack.**  
> *Validate, Analyze, and Secure your configuration files with precision.*

![License](https://img.shields.io/badge/license-MIT-0099cc?style=flat-square)
![Go](https://img.shields.io/badge/go-1.22-00add8?style=flat-square)
![React](https://img.shields.io/badge/react-18-61dafb?style=flat-square)
![Docker](https://img.shields.io/badge/docker-ready-2496ed?style=flat-square)

**Sentinel** is a dual-interface validation tool designed to catch configuration drift before it hits production. It combines a high-performance **Go CLI** for CI/CD pipelines with a **High-fidelity Dashboard** for visual debugging and reporting.

---

## Key Features

- **Dual-Mode Architecture**: Runs as a standalone CLI tool or a containerized Full-Stack Web App.
- **Visual Dashboard**: A modern terminal-inspired interface with:
  - **Syntax Highlighting** (PrismJS) for YAML/JSON.
  - **Real-time Health Visualizers** & Status Indicators.
  - **Quick-Load Presets** (Clean/Broken/Mixed) for rapid testing.
- **Smart Validation**: Detects schema violations, type mismatches, and business logic errors.
- **Reporting**: One-click **JSON Audit Exports** for compliance and ticketing.
- **Docker Native**: Builds into a single lightweight binary + static asset container.

---

## Quick Start (Docker)

The fastest way to run Sentinel is via Docker. This spins up the API and the UI instantly.

```bash
# Build the image
docker build -t sentinel .

# Run the sentinel system
# The API key is set via env var (Default used if not provided, but recommended to set one)
docker run -p 8080:8080 -e CONFIG_LINTER_API_KEY=admin-key-123 sentinel
```

**OPEN**: `http://localhost:8080`  
**KEY**: `admin-key-123`

---

## Local Development

### 1. Backend (Go)
Runs the API server which handles validation logic.

```bash
# Run the server
export CONFIG_LINTER_API_KEY=my-secret-key
export STATIC_DIR=./frontend/dist # Point to frontend assets
go run ./cmd/server
```

### 2. Frontend (React + Vite)
Runs the Dashboard UI.

```bash
cd frontend
npm install
npm run dev
```
*UI will run at `http://localhost:5173`. Configure it to point to your local Go server.*

---

## CLI Usage

For integration into **GitHub Actions** or **Pre-commit hooks**, use the CLI mode.

```bash
# Install
go install ./cmd/...

# Run validation
cli-config-linter -strict -fix-suggestions config.yaml
```

**Output Example:**
```text
config.yaml:12 [error] settings.replicas must be a positive integer
  Fix suggestion: Set settings.replicas to at least 1
```

---

## Configuration Schema

Sentinel validates against this strict schema:

| Section    | Field      | Type    | Requirement |
|------------|------------|---------|-------------|
| `metadata` | `name`     | string  | Required    |
|            | `env`      | enum    | `dev`, `staging`, `prod` |
| `settings` | `replicas` | int     | > 0         |
|            | `timeout`  | int     | > 0 (Warn if missing) |
| `features` | `enabled`  | boolean | Required    |

---

## API Reference

The Sentinel Server exposes a RESTful API for external integrations.

### `GET /health`
**Description**: Checks system status and uptime.  
**Auth**: Public  
**Response**:
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": "1h 23m 45s"
}
```

### `POST /lint`
**Description**: Validates a configuration snippet.  
**Auth**: Required (`X-API-Key` header or `Authorization: Bearer <token>`)  
**Body**:
```json
{
  "config": "metadata: ...", // The raw config string
  "strict": true,            // Fail on warnings?
  "fixSuggestions": true     // Include fix tips?
}
```
**Response**:
```json
{
  "issues": [
    {
      "line": 4,
      "severity": "error",
      "message": "settings.replicas must be positive"
    }
  ],
  "fatal": true
}
```

---

## Portfolio Notes

This project demonstrates:
- **Full Stack Engineering**: Go (Backend), React/TypeScript (Frontend).
- **Architecture**: REST API design, Docker Multi-stage builds, Single-Binary deployment.
- **UI/UX Design**: Custom "Dark Mode" aesthetic, CSS animations, and developer-centric usability.
- **Tooling**: Custom Linters, AST parsing (conceptual), and CLI design.
