// Command aid-manifest-gen scans .aidocs/ and generates a manifest.aid from file headers.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

func main() {
	dir := flag.String("dir", ".aidocs", "Directory containing .aid files")
	project := flag.String("project", "", "Project name for the manifest")
	flag.Parse()

	if *project == "" {
		// Default to parent directory name
		abs, _ := filepath.Abs(*dir)
		*project = filepath.Base(filepath.Dir(abs))
	}

	entries, err := os.ReadDir(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", *dir, err)
		os.Exit(1)
	}

	fmt.Println("@manifest")
	fmt.Printf("@project %s\n", *project)
	fmt.Println("@aid_version 0.1")

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".aid") || e.Name() == "manifest.aid" {
			continue
		}

		path := filepath.Join(*dir, e.Name())
		f, _, err := parser.ParseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not parse %s: %v\n", path, err)
			continue
		}

		fmt.Println("\n---")
		fmt.Println()

		if f.Header.Module != "" {
			fmt.Printf("@package %s\n", f.Header.Module)
		} else {
			// Derive package name from filename
			name := strings.TrimSuffix(e.Name(), ".aid")
			name = strings.ReplaceAll(name, "-", "/")
			fmt.Printf("@package %s\n", name)
		}

		fmt.Printf("@aid_file %s\n", e.Name())

		if f.Header.AidStatus != "" {
			fmt.Printf("@aid_status %s\n", f.Header.AidStatus)
		}
		if len(f.Header.Depends) > 0 {
			fmt.Printf("@depends [%s]\n", strings.Join(f.Header.Depends, ", "))
		}
		if f.Header.Purpose != "" {
			fmt.Printf("@purpose %s\n", f.Header.Purpose)
		}

		// Detect layer from content
		if len(f.Annotations) > 0 || len(f.Workflows) > 0 {
			fmt.Println("@layer l2")
		} else {
			fmt.Println("@layer l1")
		}
	}
}
