import { useEffect, useMemo, useState, useRef } from "react";
import Editor from "react-simple-code-editor";
import { highlight, languages } from "prismjs";
import "prismjs/components/prism-yaml";
import "prismjs/components/prism-json";

import "./styles.css";

type Issue = {
  line: number;
  severity: "error" | "warn";
  message: string;
  suggestedFix?: string;
};

const presets = {
  clean: `metadata:
  name: core-service-01
  env: staging
settings:
  replicas: 4
  timeout: 50
features:
  - name: sys-admin-v2
    enabled: true
  - name: legacy-mode
    enabled: false`,

  broken: `metadata:
  name: 
  env: local-dev # Invalid environment
settings:
  replicas: -1 # Must be positive
  # timeout missing (warning)
features:
  - name: 
    enabled: maybe # Invalid boolean`,

  mixed: `metadata:
  name: analytics-worker
  env: prod
settings:
  replicas: 2
  timeout: 0 # Warning: should be positive
features:
  - name: beta-opt-in
    enabled: true`
};

const API_BASE = import.meta.env.VITE_LINTER_API ?? "http://localhost:8080";

type ViewMode = "editor" | "builder" | "import";

function App() {
  const [viewMode, setViewMode] = useState<ViewMode>("editor");
  const [config, setConfig] = useState(presets.clean);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [strict, setStrict] = useState(true);
  const [fixSuggestions, setFixSuggestions] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [apiKey, setApiKey] = useState("");

  // Builder State
  const [builderState, setBuilderState] = useState({
    name: "service-01",
    env: "dev",
    replicas: 1,
    timeout: 30,
    feature1: true,
    feature2: false
  });

  // Import State
  const [importUrl, setImportUrl] = useState("");

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

  const generateConfigFromBuilder = () => {
    const yaml = `metadata:
  name: ${builderState.name}
  env: ${builderState.env}
settings:
  replicas: ${builderState.replicas}
  timeout: ${builderState.timeout}
features:
  - name: feature-a
    enabled: ${builderState.feature1}
  - name: feature-b
    enabled: ${builderState.feature2}`;
    setConfig(yaml);
    setViewMode("editor");
  };

  const fetchUrl = async () => {
    if (!importUrl) return;
    setLoading(true);
    setError("");
    try {
      const res = await fetch(`${API_BASE}/fetch`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": apiKey,
        },
        body: JSON.stringify({ url: importUrl }),
      });
      if (!res.ok) {
        const body = await res.json();
        throw new Error(body.error || "Fetch failed");
      }
      const data = await res.json();
      setConfig(data.content);
      setViewMode("editor");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

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

  const downloadReport = () => {
    const report = {
      timestamp: new Date().toISOString(),
      summary,
      issues,
      config_snapshot: config
    };
    const blob = new Blob([JSON.stringify(report, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `sentinel-report-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const lineCount = useMemo(() => config.split("\n").length, [config]);

  const handleScroll = (e: React.UIEvent<HTMLTextAreaElement> | React.UIEvent<HTMLDivElement>) => {
    const target = e.target as HTMLElement;
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = target.scrollTop;
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

      <div className="main-grid">
        <section className="panel editor">
          <div className="panel-header">
            <div style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
              <h2>Input Source</h2>
              <div className="preset-actions">
                <button onClick={() => setViewMode("editor")} className={`preset-btn ${viewMode === "editor" ? "active-tab" : ""}`}>EDITOR</button>
                <button onClick={() => setViewMode("builder")} className={`preset-btn ${viewMode === "builder" ? "active-tab" : ""}`}>BUILDER</button>
                <button onClick={() => setViewMode("import")} className={`preset-btn ${viewMode === "import" ? "active-tab" : ""}`}>IMPORT</button>
              </div>
            </div>
            {viewMode === "editor" && (
              <div className="toggles">
                <label className="toggle">
                  <input type="checkbox" checked={strict} onChange={() => setStrict(!strict)} />
                  <span>STRICT_MODE</span>
                </label>
                <button className="cta" onClick={runLint} disabled={loading}>
                  {loading ? "PROCESSING..." : "EXECUTE_LINT"}
                </button>
              </div>
            )}
          </div>

          <div className="editor-content-wrapper" style={{ height: '500px', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
            {viewMode === "editor" && (
              <div className="editor-content">
                <div className="line-numbers" ref={lineNumbersRef}>
                  {Array.from({ length: Math.max(lineCount, 15) }, (_, i) => (
                    <div key={i + 1}>{i + 1}</div>
                  ))}
                </div>
                <div style={{ flex: 1, position: 'relative' }}>
                  <Editor
                    value={config}
                    onValueChange={(code) => setConfig(code)}
                    highlight={(code) => highlight(code, languages.yaml, "yaml")}
                    padding={24}
                    className="prism-editor"
                    textareaId="config-editor"
                    onScroll={handleScroll}
                    style={{
                      fontFamily: '"JetBrains Mono", "Fira Code", monospace',
                      fontSize: "0.9rem",
                      minHeight: "100%",
                      backgroundColor: "transparent",
                    }}
                  />
                </div>
              </div>
            )}

            {viewMode === "builder" && (
              <div className="builder-form">
                <div className="form-group">
                  <label>Service Name</label>
                  <input type="text" value={builderState.name} onChange={e => setBuilderState({ ...builderState, name: e.target.value })} />
                </div>
                <div className="form-group">
                  <label>Environment</label>
                  <select value={builderState.env} onChange={e => setBuilderState({ ...builderState, env: e.target.value })}>
                    <option value="dev">Dev</option>
                    <option value="staging">Staging</option>
                    <option value="prod">Prod</option>
                  </select>
                </div>
                <div className="form-row">
                  <div className="form-group">
                    <label>Replicas</label>
                    <input type="number" value={builderState.replicas} onChange={e => setBuilderState({ ...builderState, replicas: parseInt(e.target.value) })} />
                  </div>
                  <div className="form-group">
                    <label>Timeout (s)</label>
                    <input type="number" value={builderState.timeout} onChange={e => setBuilderState({ ...builderState, timeout: parseInt(e.target.value) })} />
                  </div>
                </div>
                <div className="api-card" style={{ marginTop: 'auto' }}>
                  <button className="cta" onClick={generateConfigFromBuilder}>GENERATE CONFIG</button>
                </div>
              </div>
            )}

            {viewMode === "import" && (
              <div className="builder-form">
                <p className="lead">Load configuration from raw remote URL (GitHub Raw, Gist, etc).</p>
                <div className="form-group">
                  <label>Source URL</label>
                  <input
                    type="text"
                    placeholder="https://raw.githubusercontent.com/..."
                    value={importUrl}
                    onChange={e => setImportUrl(e.target.value)}
                  />
                </div>
                <div className="api-card">
                  <button className="cta" onClick={fetchUrl} disabled={loading}>
                    {loading ? "FETCHING..." : "FETCH SOURCE"}
                  </button>
                </div>
              </div>
            )}
          </div>
          {error && <p className="error-message">SYSTEM_ERROR: {error}</p>}
        </section>

        <section className="panel results">
          <div className="panel-header">
            <h2>Analysis Report</h2>
            {issues.length > 0 && (
              <button onClick={downloadReport} className="download-btn">
                [⬇] EXPORT_LOGS
              </button>
            )}
          </div>
          {/* ... keeping existing results section ... */}
          <div className="status-bar-container">
            <span className="status-label">SYS_STATUS</span>
            <div className="status-bar">
              {Array.from({ length: 40 }).map((_, i) => {
                let statusClass = "";
                if (summary.error && i < 20) statusClass = "critical";
                else if (summary.warn && i < 20) statusClass = "warning";
                else if (issues.length === 0) statusClass = "active";
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
          {/* ... keeping existing docs section ... */}
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
              <h3>Endpoint // Fetch</h3>
              <p>
                POST /fetch <code>{`{url}`}</code>
                <br />
                Load remote config securely.
              </p>
            </article>
          </div>
        </section>
      </div>
    </div>
  );
}

export default App;
