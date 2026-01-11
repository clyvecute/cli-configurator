package main

import (
	"flag"
	"fmt"
	"os"

	"cli-config-linter/linter"
)

var (
	strict         bool
	fixSuggestions bool
)

func init() {
	flag.BoolVar(&strict, "strict", false, "Treat warnings as fatal")
	flag.BoolVar(&fixSuggestions, "fix-suggestions", false, "Show fix suggestions for each issue")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <config-file>...\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Lint YAML or JSON configs, reporting structural or semantic issues.")
		fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	exitCode := 0
	for _, path := range flag.Args() {
		fatal, err := lintOne(path)
		if err != nil {
			exitCode = 2
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if fatal {
			exitCode = 2
		}
	}

	os.Exit(exitCode)
}

func lintOne(path string) (fatal bool, err error) {
	issues, err := linter.LintConfig(path)
	if err != nil {
		return true, fmt.Errorf("%s: %w", path, err)
	}

	if len(issues) == 0 {
		fmt.Fprintf(os.Stdout, "%s: OK\n", path)
		return false, nil
	}

	fmt.Fprintf(os.Stderr, "%s:\n", path)
	for _, issue := range issues {
		fmt.Fprintf(os.Stderr, "  %s:%d [%s] %s\n", path, issue.Line, issue.Severity, issue.Message)
		if fixSuggestions && issue.SuggestedFix != "" {
			fmt.Fprintf(os.Stderr, "    Fix suggestion: %s\n", issue.SuggestedFix)
		}

		if issue.Severity == linter.SeverityError || (strict && issue.Severity == linter.SeverityWarning) {
			fatal = true
		}
	}

	return fatal, nil
}
