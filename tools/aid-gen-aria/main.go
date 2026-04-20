package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	outputDir       = flag.String("output", ".aidocs", "Output directory for .aid files")
	stdout          = flag.Bool("stdout", false, "Print output to stdout instead of writing files")
	moduleName      = flag.String("module", "", "Override the module name")
	version         = flag.String("version", "0.0.0", "Library version for the AID header")
	verbose         = flag.Bool("v", false, "Print progress information")
	includeInternal = flag.Bool("internal", false, "Include unexported functions (minimal: @fn + @sig only, for call-graph tools)")
	includeAll      = flag.Bool("all", false, "Emit every declaration with full detail regardless of `pub` (use when documenting code that predates visibility annotations, e.g. the Aria stdlib)")
	includeTests    = flag.Bool("test", false, "Generate AID files for test blocks (mock types, test helpers)")
	perFile         = flag.Bool("per-file", false, "Emit one .aid per .aria file rather than one per directory (use for dirs containing multiple modules, e.g. /aria/lib/)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: aid-gen-aria [flags] <package-dir> [package-dir...]\n\n")
		fmt.Fprintf(os.Stderr, "Generate AID files from Aria source packages.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	for _, arg := range flag.Args() {
		dirs, err := expandPath(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error expanding %s: %v\n", arg, err)
			os.Exit(1)
		}

		for _, dir := range dirs {
			if err := processDir(dir); err != nil {
				fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", dir, err)
			}
		}
	}
}

func processDir(dir string) error {
	if *perFile {
		return processDirPerFile(dir)
	}

	modName := *moduleName
	if modName == "" {
		modName = filepath.Base(dir)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Extracting: %s → %s\n", dir, modName)
	}

	aidFile, err := ExtractPackage(dir, modName, *version, ExtractOptions{
		Internal: *includeInternal,
		All:      *includeAll,
	})
	if err != nil {
		return err
	}

	if err := writeAidFile(modName, aidFile); err != nil {
		return err
	}

	if *includeTests {
		if err := processTestDir(dir, modName); err != nil {
			if *verbose {
				fmt.Fprintf(os.Stderr, "  (no test symbols in %s: %v)\n", dir, err)
			}
		}
	}

	return nil
}

// processDirPerFile treats each .aria file in dir as its own single-file
// module. Used for directories like /aria/lib/ where every source file
// declares a distinct `mod`.
func processDirPerFile(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".aria") {
			continue
		}
		fileName := entry.Name()
		modName := strings.TrimSuffix(fileName, ".aria")

		if *verbose {
			fmt.Fprintf(os.Stderr, "Extracting (per-file): %s → %s\n", filepath.Join(dir, fileName), modName)
		}

		e := NewAriaExtractor()
		aid, err := e.ExtractFile(filepath.Join(dir, fileName), modName, *version, ExtractOptions{
			Internal: *includeInternal,
			All:      *includeAll,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", fileName, err)
			continue
		}
		if err := writeAidFile(modName, aid); err != nil {
			return err
		}
	}
	return nil
}

func processTestDir(dir string, modName string) error {
	testModName := modName + "_test"

	if *verbose {
		fmt.Fprintf(os.Stderr, "Extracting tests: %s → %s\n", dir, testModName)
	}

	aidFile, err := ExtractTestPackage(dir, testModName, *version)
	if err != nil {
		return err
	}

	return writeAidFile(testModName, aidFile)
}

func writeAidFile(modName string, aidFile *AidFile) error {
	output := Emit(aidFile)

	if *stdout {
		fmt.Print(output)
		return nil
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	filename := strings.ReplaceAll(modName, "/", "-") + ".aid"
	outPath := filepath.Join(*outputDir, filename)

	if err := os.WriteFile(outPath, []byte(output), 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "  → %s\n", outPath)
	}

	return nil
}

func expandPath(path string) ([]string, error) {
	if strings.HasSuffix(path, "/...") {
		root := strings.TrimSuffix(path, "/...")
		if root == "." || root == "" {
			root = "."
		}
		return findAriaDirs(root)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", path)
	}
	return []string{path}, nil
}

func findAriaDirs(root string) ([]string, error) {
	var dirs []string
	seen := map[string]bool{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := info.Name()
		if info.IsDir() && (strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules") {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(name, ".aria") {
			dir := filepath.Dir(path)
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
		return nil
	})
	return dirs, err
}
