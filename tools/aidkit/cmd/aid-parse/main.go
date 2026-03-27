// Command aid-parse parses .aid files and outputs them as JSON.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: aid-parse <file.aid> [file2.aid ...]\n")
		os.Exit(1)
	}

	for _, path := range os.Args[1:] {
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

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(f); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	}
}
