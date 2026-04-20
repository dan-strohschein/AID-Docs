package main

import (
	"strings"

	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// buildHeader constructs the AID module header from one or more parsed Aria
// programs (one per .aria file in the package). The first file's ModDecl
// determines the module name; @deps is the sorted union of import paths.
func buildHeader(progs []*parser.Program, modName, version string) ModuleHeader {
	h := ModuleHeader{
		Module:     modName,
		Lang:       "aria",
		Version:    version,
		AidVersion: "0.2",
	}

	// Prefer explicit mod declaration over the directory-derived modName.
	for _, p := range progs {
		if p != nil && p.Module != nil && p.Module.Name != "" {
			h.Module = p.Module.Name
			break
		}
	}

	seen := map[string]bool{}
	var deps []string
	for _, p := range progs {
		if p == nil {
			continue
		}
		for _, imp := range p.Imports {
			path := strings.Join(imp.Path, ".")
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			deps = append(deps, path)
		}
	}
	// Deterministic order.
	sortStrings(deps)
	h.Deps = deps
	return h
}
