package l2

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

// BuildReviewerPrompt constructs the prompt for a Layer 2 reviewer agent.
// The reviewer reads the L2 draft and checks every [src:] reference against source.
func BuildReviewerPrompt(l2Draft *parser.AidFile, projectRoot string) (string, error) {
	var b strings.Builder

	b.WriteString("You are a Layer 2 AID Reviewer. Verify the accuracy of every source-linked claim.\n\n")

	// L2 draft content
	b.WriteString("## Layer 2 AID Draft to Review\n\n")
	draftContent, err := readAidAsText(l2Draft)
	if err != nil {
		return "", fmt.Errorf("read L2 draft: %w", err)
	}
	b.WriteString(draftContent)
	b.WriteString("\n\n")

	// Collect all source refs and list the files to read
	refs := collectAllSourceRefs(l2Draft)
	if len(refs) > 0 {
		b.WriteString("## Source files to verify against\n\n")
		b.WriteString(fmt.Sprintf("Project root: %s\n\n", projectRoot))

		fileSet := map[string]bool{}
		for _, ref := range refs {
			fileSet[ref.File] = true
		}
		b.WriteString("Read ONLY these files (the ones referenced by [src:] links):\n\n")
		for file := range fileSet {
			fullPath := filepath.Join(projectRoot, file)
			if _, err := os.Stat(fullPath); err == nil {
				b.WriteString(fmt.Sprintf("- %s\n", fullPath))
			} else {
				b.WriteString(fmt.Sprintf("- %s (WARNING: file not found)\n", fullPath))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString(reviewerInstructions)

	return b.String(), nil
}

const reviewerInstructions = `## Instructions

For each claim with a [src: file:line] reference:
1. Read the referenced source file at the specified lines
2. Verify the claim matches what the code actually does
3. Record your findings

## Output format

` + "```" + `
## Verification Report

### Verified claims (accurate)
- [claim summary] — [src: file:line] — VERIFIED: [brief confirmation]

### Corrected claims (inaccurate)
- [claim summary] — [src: file:line] — CORRECTED: [what was wrong] → [what it should say]

### Missing claims (reviewer additions)
- [new claim] — [src: file:line] — ADDED: [why this matters]

### Stale references (line numbers wrong)
- [claim summary] — [src: file:line] — STALE: [correct location]

### Summary
- Total claims checked: N
- Verified accurate: N
- Corrected: N
- Added: N
- Stale references: N
` + "```" + `

Focus on the MOST IMPORTANT claims first: workflow steps, invariants, antipatterns.
DO NOT read CLAUDE.md or README.md.
`

// collectAllSourceRefs gathers all [src:] references from an AID file.
func collectAllSourceRefs(f *parser.AidFile) []parser.SourceRef {
	var refs []parser.SourceRef
	for _, e := range f.Entries {
		for _, field := range e.Fields {
			refs = append(refs, field.SourceRefs...)
		}
	}
	for _, w := range f.Workflows {
		for _, field := range w.Fields {
			refs = append(refs, field.SourceRefs...)
		}
	}
	return refs
}
