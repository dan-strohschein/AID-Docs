package l2

import (
	"fmt"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

// BuildIncrementalPrompt constructs a prompt that asks an AI agent to
// re-generate only the stale claims identified by CheckStaleness.
// This avoids re-generating the entire L2 AID file.
func BuildIncrementalPrompt(aidFile *parser.AidFile, staleClaims []StaleClaim, projectRoot string) string {
	var b strings.Builder

	b.WriteString("You are a Layer 2 AID Updater. Some claims in this AID file are stale because ")
	b.WriteString("the referenced source code has changed. Re-verify and update ONLY the stale claims listed below.\n\n")

	b.WriteString("## Current AID file\n\n")
	b.WriteString(fmt.Sprintf("Module: %s\n", aidFile.Header.Module))
	b.WriteString(fmt.Sprintf("Code version: %s\n\n", aidFile.Header.CodeVersion))

	b.WriteString("## Stale claims to update\n\n")
	b.WriteString(fmt.Sprintf("%d claim(s) need re-verification:\n\n", len(staleClaims)))

	// Collect unique files the updater needs to read
	fileSet := map[string]bool{}

	for i, sc := range staleClaims {
		b.WriteString(fmt.Sprintf("### Stale claim %d\n", i+1))
		b.WriteString(fmt.Sprintf("- **Entry:** %s\n", sc.Entry))
		b.WriteString(fmt.Sprintf("- **Field:** %s\n", sc.Field))
		b.WriteString(fmt.Sprintf("- **Reference:** %s\n", sc.Ref))
		b.WriteString(fmt.Sprintf("- **Reason:** %s\n", sc.Reason))
		b.WriteString(fmt.Sprintf("- **Current claim:** %s\n\n", sc.ClaimText))
		fileSet[sc.Ref.File] = true
	}

	b.WriteString("## Source files to read\n\n")
	b.WriteString(fmt.Sprintf("Project root: %s\n\n", projectRoot))
	for file := range fileSet {
		b.WriteString(fmt.Sprintf("- %s\n", file))
	}
	b.WriteString("\n")

	b.WriteString(incrementalInstructions)

	return b.String()
}

const incrementalInstructions = `## Instructions

For each stale claim above:
1. Read the referenced source file at the current line numbers
2. Determine if the claim is still accurate, needs updating, or should be removed
3. Output the updated claim with corrected [src:] references

Output format:
` + "```" + `
### Claim 1: [UPDATED | UNCHANGED | REMOVED]
@field_name updated content here
  with continuation lines if needed [src: file:new-line-numbers]
` + "```" + `

Only output changes for claims that actually need updating. If a claim is still accurate
(the code changed but the invariant still holds), mark it UNCHANGED and update the line numbers.

DO NOT re-generate the entire AID file. Only update the specific stale claims listed above.
`
