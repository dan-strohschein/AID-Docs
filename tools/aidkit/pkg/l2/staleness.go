package l2

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

// StaleClaim represents a claim whose source reference may be outdated.
type StaleClaim struct {
	Entry    string
	Field    string
	Ref      parser.SourceRef
	Reason   string // "file changed", "lines changed", "file deleted"
	ClaimText string
}

// CheckStaleness compares an AID file's @code_version against the current
// git HEAD and reports which [src:] references point to changed code.
func CheckStaleness(aidFile *parser.AidFile, projectRoot string) ([]StaleClaim, error) {
	codeVersion := aidFile.Header.CodeVersion
	if codeVersion == "" {
		return nil, fmt.Errorf("no @code_version in AID file")
	}

	// Extract git hash
	hash := strings.TrimPrefix(codeVersion, "git:")
	if hash == codeVersion {
		return nil, fmt.Errorf("@code_version %q doesn't start with 'git:'", codeVersion)
	}

	// Get current HEAD
	currentHead, err := gitHead(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("get git HEAD: %w", err)
	}

	if strings.HasPrefix(currentHead, hash) || strings.HasPrefix(hash, currentHead) {
		// Same commit — nothing is stale
		return nil, nil
	}

	// Get list of changed files between code_version and HEAD
	changedFiles, err := gitChangedFiles(projectRoot, hash, currentHead)
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	changedSet := map[string]bool{}
	for _, f := range changedFiles {
		changedSet[f] = true
	}

	// Check each source ref
	var stale []StaleClaim

	checkFields := func(entryName string, fields map[string]parser.Field) {
		for _, field := range fields {
			for _, ref := range field.SourceRefs {
				relPath := ref.File
				if changedSet[relPath] {
					// File was modified — check if the specific lines changed
					linesChanged, _ := gitLinesChanged(projectRoot, hash, currentHead, relPath, ref.StartLine, ref.EndLine)
					if linesChanged {
						stale = append(stale, StaleClaim{
							Entry:     entryName,
							Field:     field.Name,
							Ref:       ref,
							Reason:    "lines changed",
							ClaimText: truncate(field.Value(), 100),
						})
					}
				}
			}
		}
	}

	for _, e := range aidFile.Entries {
		checkFields(e.Name, e.Fields)
	}
	for _, w := range aidFile.Workflows {
		checkFields("workflow:"+w.Name, w.Fields)
	}

	return stale, nil
}

func gitHead(projectRoot string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitChangedFiles(projectRoot, fromHash, toHash string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", fromHash, toHash)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			files = append(files, l)
		}
	}
	return files, nil
}

// gitLinesChanged checks if specific lines in a file were modified between two commits.
func gitLinesChanged(projectRoot, fromHash, toHash, filePath string, startLine, endLine int) (bool, error) {
	// Use git diff with line range
	cmd := exec.Command("git", "diff", fromHash, toHash, "--", filePath)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return true, nil // Assume changed on error
	}

	diffOutput := string(out)
	if diffOutput == "" {
		return false, nil
	}

	// Parse diff hunks to check if our line range is affected
	for _, line := range strings.Split(diffOutput, "\n") {
		if !strings.HasPrefix(line, "@@") {
			continue
		}
		// Parse @@ -start,count +start,count @@ format
		hunkStart, hunkEnd := parseHunkRange(line)
		if hunkStart <= endLine && hunkEnd >= startLine {
			return true, nil // Hunk overlaps our line range
		}
	}

	return false, nil
}

func parseHunkRange(hunkLine string) (start, end int) {
	// @@ -oldStart,oldCount +newStart,newCount @@
	parts := strings.Split(hunkLine, " ")
	for _, p := range parts {
		if strings.HasPrefix(p, "+") && strings.Contains(p, ",") {
			p = strings.TrimPrefix(p, "+")
			nums := strings.Split(p, ",")
			if len(nums) == 2 {
				s, _ := strconv.Atoi(nums[0])
				c, _ := strconv.Atoi(nums[1])
				return s, s + c - 1
			}
		} else if strings.HasPrefix(p, "+") {
			p = strings.TrimPrefix(p, "+")
			s, _ := strconv.Atoi(p)
			return s, s
		}
	}
	return 0, 0
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
