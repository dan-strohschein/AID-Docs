// Command aid-gen-l2 manages the Layer 2 AID generation pipeline.
// It builds prompts for generator and reviewer agents, and checks staleness.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/l2"
	"github.com/dan-strohschein/aidkit/pkg/parser"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		cmdGenerate(os.Args[2:])
	case "review":
		cmdReview(os.Args[2:])
	case "stale":
		cmdStale(os.Args[2:])
	case "update":
		cmdUpdate(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: aid-gen-l2 <command> [flags]

Commands:
  generate    Build a generator prompt from L1 AID + source code
  review      Build a reviewer prompt from L2 draft AID
  stale       Check which [src:] references are stale vs current git HEAD
  update      Build an incremental update prompt for stale claims only

Run aid-gen-l2 <command> -help for command-specific flags.
`)
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	l1Path := fs.String("l1", "", "Path to Layer 1 .aid file (required)")
	sourceDir := fs.String("source", "", "Path to source code directory (required)")
	depsStr := fs.String("deps", "", "Comma-separated paths to dependency .aid files")
	fs.Parse(args)

	if *l1Path == "" || *sourceDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: aid-gen-l2 generate --l1 file.aid --source ./src/pkg/ [--deps a.aid,b.aid]\n")
		os.Exit(1)
	}

	l1, _, err := parser.ParseFile(*l1Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing L1 AID: %v\n", err)
		os.Exit(1)
	}

	var depAids []*parser.AidFile
	if *depsStr != "" {
		for _, depPath := range strings.Split(*depsStr, ",") {
			depPath = strings.TrimSpace(depPath)
			dep, _, err := parser.ParseFile(depPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not parse dep %s: %v\n", depPath, err)
				continue
			}
			depAids = append(depAids, dep)
		}
	}

	prompt, err := l2.BuildGeneratorPrompt(l1, *sourceDir, depAids)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building prompt: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(prompt)
}

func cmdReview(args []string) {
	fs := flag.NewFlagSet("review", flag.ExitOnError)
	draftPath := fs.String("draft", "", "Path to Layer 2 draft .aid file (required)")
	projectRoot := fs.String("project-root", ".", "Path to project root")
	fs.Parse(args)

	if *draftPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: aid-gen-l2 review --draft file-l2.aid [--project-root ./]\n")
		os.Exit(1)
	}

	draft, _, err := parser.ParseFile(*draftPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing L2 draft: %v\n", err)
		os.Exit(1)
	}

	prompt, err := l2.BuildReviewerPrompt(draft, *projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building prompt: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(prompt)
}

func cmdStale(args []string) {
	fs := flag.NewFlagSet("stale", flag.ExitOnError)
	aidPath := fs.String("aid", "", "Path to .aid file with @code_version (required)")
	projectRoot := fs.String("project-root", ".", "Path to project root (git repo)")
	fs.Parse(args)

	if *aidPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: aid-gen-l2 stale --aid file.aid [--project-root ./]\n")
		os.Exit(1)
	}

	aidFile, _, err := parser.ParseFile(*aidPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing AID file: %v\n", err)
		os.Exit(1)
	}

	staleClaims, err := l2.CheckStaleness(aidFile, *projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking staleness: %v\n", err)
		os.Exit(1)
	}

	if len(staleClaims) == 0 {
		fmt.Println("No stale claims found. AID is up to date.")
		return
	}

	fmt.Printf("Found %d stale claim(s):\n\n", len(staleClaims))
	for _, sc := range staleClaims {
		fmt.Printf("  %s.%s: %s\n", sc.Entry, sc.Field, sc.Reason)
		fmt.Printf("    ref: %s\n", sc.Ref)
		fmt.Printf("    claim: %s\n\n", sc.ClaimText)
	}
	os.Exit(1)
}

func cmdUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	aidPath := fs.String("aid", "", "Path to .aid file with @code_version (required)")
	projectRoot := fs.String("project-root", ".", "Path to project root (git repo)")
	fs.Parse(args)

	if *aidPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: aid-gen-l2 update --aid file.aid [--project-root ./]\n")
		os.Exit(1)
	}

	aidFile, _, err := parser.ParseFile(*aidPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing AID file: %v\n", err)
		os.Exit(1)
	}

	staleClaims, err := l2.CheckStaleness(aidFile, *projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking staleness: %v\n", err)
		os.Exit(1)
	}

	if len(staleClaims) == 0 {
		fmt.Fprintln(os.Stderr, "No stale claims found. AID is up to date.")
		return
	}

	prompt := l2.BuildIncrementalPrompt(aidFile, staleClaims, *projectRoot)
	fmt.Print(prompt)
}
