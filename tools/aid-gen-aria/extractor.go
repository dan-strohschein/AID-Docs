package main

// Extractor is the language-agnostic extraction contract (OCP/DIP).
// A future aid-gen-<lang> implementation would provide its own Extractor.
type Extractor interface {
	Extract(dir, modName, version string, opts ExtractOptions) (*AidFile, error)
	ExtractTests(dir, testModName, version string) (*AidFile, error)
}

// ExtractOptions controls which declarations are emitted and at what detail level.
type ExtractOptions struct {
	// Internal includes private declarations as minimal entries (name + sig only),
	// matching aid-gen-go's -internal behaviour for call-graph tooling.
	Internal bool
	// All emits every declaration with full detail regardless of visibility.
	// Intended for documenting code that has not yet been pub-annotated.
	All bool
}

// ExtractPackage is the main entry point for L1 extraction from an Aria package.
func ExtractPackage(dir, modName, version string, opts ExtractOptions) (*AidFile, error) {
	e := NewAriaExtractor()
	return e.Extract(dir, modName, version, opts)
}

// ExtractTestPackage extracts test-block-only symbols from an Aria package.
// Implementation lands in milestone 6.
func ExtractTestPackage(dir, testModName, version string) (*AidFile, error) {
	e := NewAriaExtractor()
	return e.ExtractTests(dir, testModName, version)
}
