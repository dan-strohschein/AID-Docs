// Command aid-validate checks .aid files against AID spec rules.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
	"github.com/dan-strohschein/aidkit/pkg/validator"
)

func main() {
	dir := flag.String("dir", "", "Validate all .aid files in a directory")
	flag.Parse()

	var files []string

	if *dir != "" {
		entries, err := os.ReadDir(*dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
			os.Exit(1)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".aid") {
				files = append(files, filepath.Join(*dir, e.Name()))
			}
		}
	}

	files = append(files, flag.Args()...)

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: aid-validate [--dir DIR] [file.aid ...]\n")
		os.Exit(1)
	}

	hasErrors := false
	totalIssues := 0

	for _, path := range files {
		f, parseWarns, err := parser.ParseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: parse error: %v\n", path, err)
			hasErrors = true
			continue
		}

		issues := validator.Validate(f)
		fileIssues := len(issues) + len(parseWarns)
		totalIssues += fileIssues

		if fileIssues == 0 {
			fmt.Printf("%s: OK\n", path)
			continue
		}

		fmt.Printf("%s: %d issue(s)\n", path, fileIssues)

		for _, w := range parseWarns {
			fmt.Printf("  [WARN] parse: %s\n", w)
		}
		for _, i := range issues {
			fmt.Printf("  %s\n", i)
			if i.Severity == validator.SeverityError {
				hasErrors = true
			}
		}
	}

	fmt.Printf("\n%d file(s) checked, %d issue(s) found\n", len(files), totalIssues)

	if hasErrors {
		os.Exit(1)
	}
}
