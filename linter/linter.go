package linter

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warn"
)

const defaultTimeout = 30

var allowedEnvironments = []string{"dev", "staging", "prod"}

type Issue struct {
	Line         int      `json:"line"`
	Severity     Severity `json:"severity"`
	Message      string   `json:"message"`
	SuggestedFix string   `json:"suggestedFix,omitempty"`
}

type fieldInfo struct {
	Value string
	Line  int
}

type featureEntry struct {
	Fields map[string]fieldInfo
	Line   int
}

type parsedConfig struct {
	Metadata     map[string]fieldInfo
	MetadataLine int
	Settings     map[string]fieldInfo
	SettingsLine int
	Features     []featureEntry
	FeaturesLine int
}

func LintConfig(path string) ([]Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LintBytes(data)
}

func LintBytes(data []byte) ([]Issue, error) {
	cfg, err := parseConfig(data)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	validateMetadata(cfg, &issues)
	validateSettings(cfg, &issues)
	validateFeatures(cfg, &issues)

	return issues, nil
}

func parseConfig(data []byte) (parsedConfig, error) {
	cfg := parsedConfig{
		Metadata: make(map[string]fieldInfo),
		Settings: make(map[string]fieldInfo),
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	section := ""
	var currentFeature featureEntry

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		trimmed = strings.TrimSuffix(trimmed, ",")
		clean := strings.TrimSpace(trimmed)
		switch clean {
		case "{", "}", "[", "]":
			if section == "features" && clean == "}" && len(currentFeature.Fields) > 0 {
				cfg.Features = append(cfg.Features, currentFeature)
				currentFeature = featureEntry{}
			}
			continue
		}

		if section == "features" {
			if strings.HasPrefix(clean, "-") {
				if len(currentFeature.Fields) > 0 {
					cfg.Features = append(cfg.Features, currentFeature)
				}
				currentFeature = featureEntry{
					Fields: make(map[string]fieldInfo),
					Line:   lineNo,
				}
				clean = strings.TrimSpace(strings.TrimPrefix(clean, "-"))
				if clean == "" {
					continue
				}
			}
			if strings.HasPrefix(clean, "{") {
				if len(currentFeature.Fields) > 0 {
					cfg.Features = append(cfg.Features, currentFeature)
				}
				currentFeature = featureEntry{
					Fields: make(map[string]fieldInfo),
					Line:   lineNo,
				}
				clean = strings.TrimSpace(strings.TrimPrefix(clean, "{"))
				if clean == "" {
					continue
				}
			}
			if clean == "}" {
				if len(currentFeature.Fields) > 0 {
					cfg.Features = append(cfg.Features, currentFeature)
					currentFeature = featureEntry{}
				}
				continue
			}
		}

		key, value, hasValue := parseKeyValue(clean)
		if key == "" {
			continue
		}

		if section == "" {
			switch key {
			case "metadata", `"metadata"`:
				section = "metadata"
				cfg.MetadataLine = lineNo
				continue
			case "settings", `"settings"`:
				section = "settings"
				cfg.SettingsLine = lineNo
				continue
			case "features", `"features"`:
				section = "features"
				cfg.FeaturesLine = lineNo
				continue
			}
		}

		switch key {
		case "metadata", `"metadata"`:
			section = "metadata"
			cfg.MetadataLine = lineNo
			continue
		case "settings", `"settings"`:
			section = "settings"
			cfg.SettingsLine = lineNo
			continue
		case "features", `"features"`:
			section = "features"
			cfg.FeaturesLine = lineNo
			continue
		}

		if section == "metadata" {
			if hasValue {
				cfg.Metadata[key] = fieldInfo{Value: value, Line: lineNo}
			}
			continue
		}

		if section == "settings" {
			if hasValue {
				cfg.Settings[key] = fieldInfo{Value: value, Line: lineNo}
			}
			continue
		}

		if section == "features" {
			if !hasValue {
				continue
			}
			if len(currentFeature.Fields) == 0 {
				currentFeature = featureEntry{
					Fields: make(map[string]fieldInfo),
					Line:   lineNo,
				}
			}
			currentFeature.Fields[key] = fieldInfo{Value: value, Line: lineNo}
		}
	}

	if len(currentFeature.Fields) > 0 {
		cfg.Features = append(cfg.Features, currentFeature)
	}

	if err := scanner.Err(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func parseKeyValue(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return "", "", false
	}

	key := strings.TrimSpace(line[:idx])
	key = strings.Trim(key, `"'`)
	value := strings.TrimSpace(line[idx+1:])
	value = strings.Trim(value, `"'`)

	if value == "{" || value == "[" {
		value = ""
	}

	return key, value, true
}

func looksLikeJSON(data []byte) bool {
	for _, b := range data {
		if b == ' ' || b == '\n' || b == '\r' || b == '\t' {
			continue
		}
		return b == '{' || b == '['
	}
	return false
}

func validateMetadata(cfg parsedConfig, issues *[]Issue) {
	baseLine := cfg.MetadataLine
	if baseLine == 0 {
		baseLine = 1
		if len(cfg.Metadata) == 0 {
			baseLine = 1
		}
	}

	if len(cfg.Metadata) == 0 {
		*issues = append(*issues, Issue{
			Line:     baseLine,
			Severity: SeverityError,
			Message:  "missing metadata section",
			SuggestedFix: "Add a 'metadata' mapping with 'name' and 'env' fields",
		})
		return
	}

	name, hasName := cfg.Metadata["name"]
	if !hasName || name.Value == "" {
		if name.Line == 0 {
			name.Line = baseLine
		}
		*issues = append(*issues, Issue{
			Line:     name.Line,
			Severity: SeverityError,
			Message:  "metadata.name is required",
			SuggestedFix: "Set metadata.name to a non-empty identifier, e.g. metadata.name: my-service",
		})
	}

	env, hasEnv := cfg.Metadata["env"]
	if !hasEnv || env.Value == "" {
		if env.Line == 0 {
			env.Line = baseLine
		}
		*issues = append(*issues, Issue{
			Line:     env.Line,
			Severity: SeverityError,
			Message:  "metadata.env is required",
			SuggestedFix: fmt.Sprintf("Set metadata.env to one of: %s", strings.Join(allowedEnvironments, ", ")),
		})
	} else if !contains(allowedEnvironments, env.Value) {
		*issues = append(*issues, Issue{
			Line:         env.Line,
			Severity:     SeverityWarning,
			Message:      fmt.Sprintf("metadata.env value %q is not recognized", env.Value),
			SuggestedFix: fmt.Sprintf("Use one of: %s", strings.Join(allowedEnvironments, ", ")),
		})
	}
}

func validateSettings(cfg parsedConfig, issues *[]Issue) {
	baseLine := cfg.SettingsLine
	if baseLine == 0 {
		baseLine = 1
	}

	if len(cfg.Settings) == 0 {
		*issues = append(*issues, Issue{
			Line:     baseLine,
			Severity: SeverityError,
			Message:  "missing settings section",
			SuggestedFix: "Add a 'settings' mapping with 'replicas' and 'timeout'",
		})
		return
	}

	replicas, hasReplicas := cfg.Settings["replicas"]
	if !hasReplicas {
		*issues = append(*issues, Issue{
			Line:     baseLine,
			Severity: SeverityError,
			Message:  "settings.replicas is required",
			SuggestedFix: "Add settings.replicas: 1",
		})
	} else if !isPositiveInt(replicas.Value) {
		*issues = append(*issues, Issue{
			Line:     replicas.Line,
			Severity: SeverityError,
			Message:  "settings.replicas must be a positive integer",
		})
	}

	timeout, hasTimeout := cfg.Settings["timeout"]
	if !hasTimeout {
		*issues = append(*issues, Issue{
			Line:     baseLine,
			Severity: SeverityWarning,
			Message:  "settings.timeout is missing; defaulting to 30",
			SuggestedFix: fmt.Sprintf("Add settings.timeout: %d", defaultTimeout),
		})
	} else if !isPositiveInt(timeout.Value) {
		*issues = append(*issues, Issue{
			Line:     timeout.Line,
			Severity: SeverityWarning,
			Message:  "settings.timeout should be a positive integer",
		})
	}
}

func validateFeatures(cfg parsedConfig, issues *[]Issue) {
	for _, feature := range cfg.Features {
		if len(feature.Fields) == 0 {
			*issues = append(*issues, Issue{
				Line:     feature.Line,
				Severity: SeverityWarning,
				Message:  "each feature entry should be a mapping",
			})
			continue
		}

		name, hasName := feature.Fields["name"]
		if !hasName || name.Value == "" {
			*issues = append(*issues, Issue{
				Line:     feature.Line,
				Severity: SeverityWarning,
				Message:  "feature entry missing name",
				SuggestedFix: "Add name: <feature-name>",
			})
		}

		enabled, hasEnabled := feature.Fields["enabled"]
		if !hasEnabled || !isBool(enabled.Value) {
			*issues = append(*issues, Issue{
				Line:     feature.Line,
				Severity: SeverityWarning,
				Message:  "feature enabled should be true or false",
			})
		}
	}
}

func isPositiveInt(value string) bool {
	if value == "" {
		return false
	}
	if _, err := strconv.Atoi(value); err != nil || value == "0" {
		return false
	}
	return true
}

func isBool(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return v == "true" || v == "false"
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
