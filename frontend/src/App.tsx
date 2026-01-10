import { useEffect, useMemo, useState } from "react";
import "./styles.css";

type Issue = {
  line: number;
  severity: "error" | "warn";
  message: string;
  suggestedFix?: string;
};

const sampleConfig = `metadata:
  name: radiant-service
  env: staging
settings:
  replicas: 2
  timeout: 50
features:
  - name: shimmering-ui
    enabled: true
  - name: dark-mode
    enabled: false`;

const severityBadge: Record<Issue["severity"], string> = {
  error: "badge-error",
  warn: "badge-warn",
};

const API_BASE = import.meta.env.VITE_LINTER_API ?? "http://localhost:8080";

const issueHeaders: Record<Issue["severity"], string> = {
  error: "Critical",
  warn: "Warning",
};

function App() {
  const [config, setConfig] = useState(sampleConfig);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [strict, setStrict] = useState(true);
  const [fixSuggestions, setFixSuggestions] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [apiKey, setApiKey] = useState("");

  useEffect(() => {
    const stored = localStorage.getItem("cli-linter-api-key");
    if (stored) {
      setApiKey(stored);
    }
  }, []);

  useEffect(() => {
    localStorage.setItem("cli-linter-api-key", apiKey);
  }, [apiKey]);

  const runLint = async () => {
    setLoading(true);
    setError("");
    try {
      const res = await fetch(`${API_BASE}/lint`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": apiKey,
        },
        body: JSON.stringify({
          config,
          strict,
          fixSuggestions,
        }),
      });
      if (!res.ok) {
        const body = await res.json();
        throw new Error(body.error || "Lint failed");
      }
      const data = await res.json();
      setIssues(data.issues);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const summary = useMemo(() => {
    const counts = issues.reduce(
      (acc, issue) => {
        acc[issue.severity] = (acc[issue.severity] || 0) + 1;
        return acc;
      },
      {} as Record<Issue["severity"], number>,
    );
    return counts;
  }, [issues]);

  return (
    <div className="app-shell">
      <div className="blurred-light" />
      <header className="hero">
        <div>
          <p className="eyebrow">Config Assurance Suite</p>
          <h1>Config Sentinel</h1>
          <p className="lead">
            Upload YAML/JSON configuration, surface line-aware issues, and ship stable
            deployments faster.
          </p>
        </div>
        <div className="api-card">
          <p className="label">API Key</p>
          <input
            className="input-field"
            type="password"
            value={apiKey}
            onChange={(event) => setApiKey(event.target.value)}
            placeholder="Paste your Config Sentinel API key"
          />
          <small>Stored only in this browser for convenience.</small>
        </div>
      </header>

      <main className="main-grid">
        <section className="panel editor">
          <div className="panel-header">
            <div>
              <h2>Config source</h2>
              <p>Supports YAML or JSON. Strict mode makes warnings fatal.</p>
            </div>
            <div className="toggles">
              <label className="toggle">
                <input type="checkbox" checked={strict} onChange={() => setStrict(!strict)} />
                <span>Strict</span>
              </label>
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={fixSuggestions}
                  onChange={() => setFixSuggestions(!fixSuggestions)}
                />
                <span>Fix hints</span>
              </label>
              <button className="cta" onClick={runLint} disabled={loading}>
                {loading ? "Linting…" : "Lint config"}
              </button>
            </div>
          </div>
          <textarea value={config} onChange={(event) => setConfig(event.target.value)} />
          {error && <p className="error-message">{error}</p>}
        </section>

        <section className="panel results">
          <div className="panel-header">
            <h2>Verdict</h2>
            <p>
              Issues detected: <strong>{issues.length}</strong>
            </p>
          </div>
          <div className="summary-row">
            <div className="summary-pill">
              <span>Errors</span>
              <strong>{summary.error || 0}</strong>
            </div>
            <div className="summary-pill warn">
              <span>Warnings</span>
              <strong>{summary.warn || 0}</strong>
            </div>
          </div>
          <div className="issue-grid">
            {issues.length === 0 && (
              <div className="empty-state">
                Config looks solid. Use the strict toggle to gate warnings.
              </div>
            )}
            {issues.map((issue) => (
              <article key={`${issue.line}-${issue.message}`} className="issue-card">
                <div className={`badge ${severityBadge[issue.severity]}`}>
                  <p>{issueHeaders[issue.severity]}</p>
                  <small>line {issue.line}</small>
                </div>
                <p className="issue-message">{issue.message}</p>
                {fixSuggestions && issue.suggestedFix && (
                  <p className="fix">Fix: {issue.suggestedFix}</p>
                )}
              </article>
            ))}
          </div>
        </section>

        <section className="panel docs">
          <div className="panel-header">
            <h2>Documentation & Workflow</h2>
            <p>Built for CI, design ops, and product teams.</p>
          </div>
          <ul>
            <li>Run the CLI locally or via the API for automation and pre-commit hooks.</li>
            <li>Strict mode fails the build on any warning when toggled.</li>
            <li>Fix suggestions help writers correct configs quickly.</li>
            <li>Copy the JSON output or hook the API into GitHub Actions.</li>
          </ul>
          <div className="doc-grid">
            <article>
              <h3>API contract</h3>
              <p>
                POST /lint with <code>{`{config, strict, fixSuggestions}`}</code> and send
                the API key via <code>X-API-Key</code> or a Bearer token.
              </p>
            </article>
            <article>
              <h3>Frontend guidance</h3>
              <p>
                Build custom dashboards, compare configs, integrate VS Code policies, or
                pipe results to Slack / email alerts.
              </p>
            </article>
          </div>
        </section>
      </main>
    </div>
  );
}

export default App;
