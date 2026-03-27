// Command aid-parse parses .aid files and outputs them as JSON or summary text.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

func main() {
	summary := flag.Bool("summary", false, "Output only module annotations (invariants, antipatterns, decisions, notes)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: aid-parse [--summary] <file.aid> [file2.aid ...]\n\n")
		fmt.Fprintf(os.Stderr, "Parse .aid files and output as JSON (default) or summary text.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	for _, path := range flag.Args() {
		f, warns, err := parser.ParseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", path, err)
			os.Exit(1)
		}

		if len(warns) > 0 {
			for _, w := range warns {
				fmt.Fprintf(os.Stderr, "%s: %s\n", path, w)
			}
		}

		if *summary {
			printSummary(f)
		} else {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(f); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
		}
	}
}

// printSummary outputs only the high-value module-level content:
// header basics, annotations (invariants, antipatterns, decisions, notes),
// and workflow names. Skips all per-entry detail.
func printSummary(f *parser.AidFile) {
	// Header
	fmt.Printf("@module %s\n", f.Header.Module)
	if f.Header.Lang != "" {
		fmt.Printf("@lang %s\n", f.Header.Lang)
	}
	if f.Header.Purpose != "" {
		fmt.Printf("@purpose %s\n", f.Header.Purpose)
	}
	if f.Header.CodeVersion != "" {
		fmt.Printf("@code_version %s\n", f.Header.CodeVersion)
	}
	if f.Header.AidStatus != "" {
		fmt.Printf("@aid_status %s\n", f.Header.AidStatus)
	}
	if len(f.Header.Depends) > 0 {
		fmt.Printf("@depends [%s]\n", strings.Join(f.Header.Depends, ", "))
	}
	fmt.Println()

	// Entry/workflow count for orientation
	fmt.Printf("// %d entries, %d workflows, %d annotations\n\n", len(f.Entries), len(f.Workflows), len(f.Annotations))

	// Annotations — the highest-value content
	for _, a := range f.Annotations {
		if a.Name != "" {
			fmt.Printf("@%s %s\n", a.Kind, a.Name)
		} else {
			fmt.Printf("@%s\n", a.Kind)
		}
		for fieldName, field := range a.Fields {
			if fieldName == a.Kind {
				// The block's own continuation lines
				for _, line := range field.Lines {
					fmt.Printf("  %s\n", line)
				}
			} else {
				if field.InlineValue != "" {
					fmt.Printf("@%s %s\n", fieldName, field.InlineValue)
				}
				for _, line := range field.Lines {
					fmt.Printf("  %s\n", line)
				}
			}
		}
		fmt.Println()
	}

	// Workflow names only (not steps)
	if len(f.Workflows) > 0 {
		fmt.Println("// Workflows:")
		for _, w := range f.Workflows {
			purpose := ""
			if p, has := w.Fields["purpose"]; has {
				purpose = " — " + p.InlineValue
			}
			fmt.Printf("//   @workflow %s%s\n", w.Name, purpose)
		}
		fmt.Println()
	}
}
