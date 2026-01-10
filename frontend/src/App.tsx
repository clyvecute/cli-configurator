import { useEffect, useMemo, useState, useRef } from "react";
import "./styles.css";

type Issue = {
  line: number;
  severity: "error" | "warn";
  message: string;
  suggestedFix?: string;
};

const sampleConfig = `metadata:
  name: core-service-01
  env: staging
settings:
  replicas: 4
  timeout: 50
features:
  - name: sys-admin-v2
    enabled: true
  - name: legacy-mode
    enabled: false`;

const API_BASE = import.meta.env.VITE_LINTER_API ?? "http://localhost:8080";

function App() {
  const [config, setConfig] = useState(sampleConfig);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [strict, setStrict] = useState(true);
  const [fixSuggestions, setFixSuggestions] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [apiKey, setApiKey] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const lineNumbersRef = useRef<HTMLDivElement>(null);

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

  const lineCount = useMemo(() => config.split("\n").length, [config]);

  // Sync scrolling between textarea and line numbers
  const handleScroll = () => {
    if (textareaRef.current && lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = textareaRef.current.scrollTop;
    }
  };

  return (
    <div className="app-shell">
      <header className="hero">
        <div>
          <span className="eyebrow">System // Config_Linter</span>
          <h1>Sentinel</h1>
          <p className="lead">
            Static analysis for infrastructure configuration.
            Ensure deployment stability.
          </p>
        </div>
        <div className="api-card">
          <span className="label">Access Key</span>
          <input
            type="password"
            value={apiKey}
            onChange={(event) => setApiKey(event.target.value)}
            placeholder="ENTER API KEY"
            autoComplete="off"
          />
        </div>
      </header>

      <main className="main-grid">
        <section className="panel editor">
          <div className="panel-header">
            <div>
              <h2>Input Source</h2>
            </div>
            <div className="toggles">
              <label className="toggle">
                <input type="checkbox" checked={strict} onChange={() => setStrict(!strict)} />
                <span>STRICT_MODE</span>
              </label>
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={fixSuggestions}
                  onChange={() => setFixSuggestions(!fixSuggestions)}
                />
                <span>AUTO_SUGGEST</span>
              </label>
              <button className="cta" onClick={runLint} disabled={loading}>
                {loading ? "PROCESSING..." : "EXECUTE_LINT"}
              </button>
            </div>
          </div>
          <div className="editor-content">
            <div className="line-numbers" ref={lineNumbersRef}>
              {Array.from({ length: Math.max(lineCount, 15) }, (_, i) => (
                <div key={i + 1}>{i + 1}</div>
              ))}
            </div>
            <textarea
              ref={textareaRef}
              value={config}
              onChange={(event) => setConfig(event.target.value)}
              onScroll={handleScroll}
              spellCheck={false}
              placeholder="# YAML/JSON Config"
            />
          </div>
          {error && <p className="error-message">SYSTEM_ERROR: {error}</p>}
        </section>

        <section className="panel results">
          <div className="panel-header">
            <h2>Analysis Report</h2>
          </div>

          <div className="status-bar-container">
            <span className="status-label">SYS_STATUS</span>
            <div className="status-bar">
              {Array.from({ length: 40 }).map((_, i) => {
                let statusClass = "";
                if (summary.error && i < 20) statusClass = "critical";
                else if (summary.warn && i < 20) statusClass = "warning";
                else if (issues.length === 0) statusClass = "active";

                // Add some randomness to "active" state or specific patterns
                if (issues.length === 0 && Math.random() > 0.8) statusClass = "";

                return <div key={i} className={`segment ${statusClass}`} />;
              })}
            </div>
            <span className="status-label" style={{ textAlign: "right" }}>
              {issues.length === 0 ? "NOMINAL" : "ALERT"}
            </span>
          </div>

          <div className="summary-row">
            <div className="summary-pill">
              <span>Critical Errors</span>
              <strong>{summary.error || 0}</strong>
            </div>
            <div className="summary-pill">
              <span>Warnings</span>
              <strong>{summary.warn || 0}</strong>
            </div>
          </div>
          <div className="issue-grid">
            {issues.length === 0 && (
              <div className="empty-state">
                NO ISSUES DETECTED. SYSTEM NOMINAL.
              </div>
            )}
            {issues.map((issue, idx) => (
              <article
                key={`${issue.line}-${idx}`}
                className="issue-card"
                data-severity={issue.severity}
              >
                <div className="badge">
                  <span>{issue.severity.toUpperCase()}</span>
                  <span>LN:{issue.line}</span>
                </div>
                <p className="issue-message">{issue.message}</p>
                {fixSuggestions && issue.suggestedFix && (
                  <p className="fix">{issue.suggestedFix}</p>
                )}
              </article>
            ))}
          </div>
        </section>

        <section className="panel docs">
          <div className="panel-header">
            <h2>System Documentation</h2>
          </div>
          <ul>
            <li>CLI: Run locally for pre-commit validation loops.</li>
            <li>STRICT: Halts build pipelines on any warning signal.</li>
            <li>API: Integrate with external CI/CD monitoring tools.</li>
          </ul>
          <div className="doc-grid">
            <article>
              <h3>Endpoint // Lint</h3>
              <p>
                POST /lint <code>{`{config, strict, fixSuggestions}`}</code>
                <br />
                Auth: <code>X-API-Key</code> or Bearer Token.
              </p>
            </article>
            <article>
              <h3>Usage Guide</h3>
              <p>
                Paste YAML/JSON configuration to validate schema compliance.
                Use generated report to patch infrastructure definitions.
              </p>
            </article>
          </div>
        </section>
      </main>
    </div>
  );
}

export default App;
