# CLI Config Linter

`cli-config-linter` reads YAML or JSON configuration files, validates a fixed schema, and reports issues with line numbers. It supports a strict mode (treat warnings as errors) and outputs fix suggestions when requested.

## Installation

1. Install Go 1.22 or newer: https://go.dev/dl/  
2. Run `go install ./...` from this repository root.

## Usage

```sh
cli-config-linter [flags] <config-file>...
```

### Flags

- `-strict` (`bool`): Treat warnings as fatal issues.  
- `-fix-suggestions` (`bool`): Print optional fix guidance for each issue.

### Sample config

```yaml
metadata:
  name: sample-app
  env: prod
settings:
  replicas: 3
  timeout: 45
features:
  - name: login
    enabled: true
```

### Output

```
/path/to/config.yaml:
  /path/to/config.yaml:12 [warn] settings.timeout is missing; defaulting to 30
    Fix suggestion: Add settings.timeout: 30
```

Warnings are always reported. When `-strict` is provided, warnings cause the CLI to exit with a non-zero status. Errors always abort the run regardless of mode.

## Workflow for the full stack

1. **Backend service**
   ```sh
   CONFIG_LINTER_API_KEY=key123 go run ./server
   ```
   - Starts on `:8080` unless you set `LINTER_SERVER_PORT`.  
   - `CONFIG_LINTER_API_KEY` may contain comma-separated keys; the server rejects requests without a configured API key.

2. **Frontend UI**
   ```sh
   cd frontend
   npm install      # run once, already done
   npm run dev
   ```
   - Open `http://localhost:5173` to reach the React dashboard.
   - Paste your YAML/JSON or load `sample-config.json` (located at the repo root) directly into the editor.
   - Enter one of the configured API keys, toggle **Strict**/ **Fix hints**, and click **Lint config**.

3. **API contract**
   - `POST /lint` with `{ config, strict, fixSuggestions }`.  
   - Send API key via `X-API-Key` header or `Authorization: Bearer <key>`.
   - The response includes `issues`, `strict`, `fatal`, and a timestamp for auditing.
   - Grab the JSON output for automation: e.g., pipe into GitHub Actions, Slack bots, or VS Code tasks.

4. **Bind CLI → API**
   - The CLI still works standalone: `cli-config-linter -strict sample.yaml`.  
   - Use it in scripts or pre-commit hooks to gate commits while the UI serves reporting/game day review.

## Testing

Run `go test ./...` after installing Go.
