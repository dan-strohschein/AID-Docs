// Package discovery implements the AID file discovery protocol per spec §10.5.
package discovery

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

const AidDocsDir = ".aidocs"
const ManifestFile = "manifest.aid"

// Result holds the discovery outcome.
type Result struct {
	AidDocsPath  string            // Absolute path to .aidocs/ directory
	ManifestPath string            // Path to manifest.aid (empty if not found)
	Manifest     *parser.AidFile   // Parsed manifest (nil if no manifest)
	AidFiles     []string          // All .aid files found
}

// Discover walks up from startDir looking for .aidocs/ per the spec protocol.
func Discover(startDir string) (*Result, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}

	dir := absStart
	for {
		candidate := filepath.Join(dir, AidDocsDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return inspectAidDocs(candidate)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}

	return nil, nil // no .aidocs/ found
}

func inspectAidDocs(aidDocsPath string) (*Result, error) {
	result := &Result{
		AidDocsPath: aidDocsPath,
	}

	// List .aid files
	entries, err := os.ReadDir(aidDocsPath)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".aid") {
			result.AidFiles = append(result.AidFiles, e.Name())
		}
	}

	// Check for manifest
	manifestPath := filepath.Join(aidDocsPath, ManifestFile)
	if _, err := os.Stat(manifestPath); err == nil {
		result.ManifestPath = manifestPath
		manifest, _, err := parser.ParseFile(manifestPath)
		if err == nil {
			result.Manifest = manifest
		}
	}

	return result, nil
}
