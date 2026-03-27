// Command aid-discover finds the nearest .aidocs/ directory and lists available AID files.
package main

import (
	"fmt"
	"os"

	"github.com/dan-strohschein/aidkit/pkg/discovery"
)

func main() {
	startDir := "."
	if len(os.Args) > 1 {
		startDir = os.Args[1]
	}

	result, err := discovery.Discover(startDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if result == nil {
		fmt.Println("No .aidocs/ directory found.")
		os.Exit(1)
	}

	fmt.Printf("Found: %s\n", result.AidDocsPath)
	fmt.Printf("Files: %d .aid file(s)\n\n", len(result.AidFiles))

	if result.Manifest != nil {
		fmt.Println("Manifest packages:")
		for _, e := range result.Manifest.Entries {
			if e.Kind == "package" {
				purpose := ""
				if p, has := e.Fields["purpose"]; has {
					purpose = " — " + p.InlineValue
				}
				layer := ""
				if l, has := e.Fields["layer"]; has {
					layer = " [" + l.InlineValue + "]"
				}
				risks := ""
				if r, has := e.Fields["key_risks"]; has {
					risks = "\n    risks: " + r.InlineValue
				}
				fmt.Printf("  %s%s%s%s\n", e.Name, layer, purpose, risks)
			}
		}
	} else {
		fmt.Println("Files:")
		for _, f := range result.AidFiles {
			fmt.Printf("  %s\n", f)
		}
	}
}
