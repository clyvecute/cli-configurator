package linter

import (
	"os"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	tmp, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmp.Close()

	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	return tmp.Name()
}

func TestLintConfigValid(t *testing.T) {
	content := `
metadata:
  name: awesome
  env: prod
settings:
  replicas: 2
  timeout: 60
features:
  - name: featureA
    enabled: true
`

	path := writeTempConfig(t, content)
	defer os.Remove(path)

	issues, err := LintConfig(path)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(issues))
	}
}

func TestLintConfigErrorsAndWarnings(t *testing.T) {
	content := `
metadata:
  env: unknown
settings:
  replicas: 0
  timeout: -5
features:
  - enabled: maybe
`

	path := writeTempConfig(t, content)
	defer os.Remove(path)

	issues, err := LintConfig(path)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(issues) < 4 {
		t.Fatalf("expected at least 4 issues, got %d", len(issues))
	}

	var hasMissingName bool
	var hasBadEnv bool
	var hasReplicas bool
	var hasFeature bool

	for _, issue := range issues {
		switch issue.Message {
		case "metadata.name is required":
			hasMissingName = true
		case "metadata.env value \"unknown\" is not recognized":
			hasBadEnv = true
		case "settings.replicas must be a positive integer":
			hasReplicas = true
		case "feature entry missing name":
			hasFeature = true
		}
	}

	if !hasMissingName || !hasBadEnv || !hasReplicas || !hasFeature {
		t.Fatalf("missing expected issue detail: %+v", issues)
	}
}
