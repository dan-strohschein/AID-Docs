package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// AriaExtractor is the Aria-language implementation of Extractor.
// Responsibilities are split across files in this package following SRP:
//
//	types.go          — Aria TypeExpr → AID universal type notation (M2)
//	source_index.go   — leading-// comment indexing for @purpose (M3)
//	extract_module.go — header / deps (M3)
//	extract_fn.go     — FnDecl extraction (M3)
//	extract_type.go   — TypeDecl / EnumDecl / AliasDecl (M3)
//	extract_trait.go  — TraitDecl / ImplDecl (M3)
//	extract_const.go  — ConstDecl (M3)
//	callgraph.go      — CallExpr walking + receiver resolution (M5)
//	positions.go      — @source_file / @source_line (covered inline in M3)
type AriaExtractor struct{}

func NewAriaExtractor() *AriaExtractor { return &AriaExtractor{} }

// ExtractFile parses a single .aria file and emits an AidFile. Used by
// per-file mode for stdlib directories where each source file is its own
// module. @source_file paths are basename-only since there's no enclosing
// package directory to relativize against.
func (a *AriaExtractor) ExtractFile(file, modName, version string, opts ExtractOptions) (*AidFile, error) {
	return a.extractFromFiles([]string{file}, filepath.Dir(file), modName, version, opts)
}

// Extract parses every .aria file in dir and emits a single AidFile.
func (a *AriaExtractor) Extract(dir, modName, version string, opts ExtractOptions) (*AidFile, error) {
	files, err := listAriaFiles(dir, false)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return &AidFile{
			Header: ModuleHeader{
				Module:     modName,
				Lang:       "aria",
				Version:    version,
				AidVersion: "0.2",
			},
		}, nil
	}
	return a.extractFromFiles(files, dir, modName, version, opts)
}

// extractFromFiles is the common core shared by Extract and ExtractFile.
func (a *AriaExtractor) extractFromFiles(files []string, dir, modName, version string, opts ExtractOptions) (*AidFile, error) {
	emit := func(v parser.Visibility) bool {
		return v == parser.Public || opts.Internal || opts.All
	}

	progs := make([]*parser.Program, 0, len(files))
	docs := newSourceIndex()

	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		if err := docs.loadFile(f); err != nil {
			return nil, err
		}
		prog := parser.Parse(f, string(src))
		progs = append(progs, prog)
	}

	aid := &AidFile{Header: buildHeader(progs, modName, version)}
	typeNames := map[string]bool{}
	var methodBuckets []FnEntry

	// Registry of every FnDecl in the package keyed by its qualified name
	// (free fns by bare name, methods by "Type.method"). Used by the
	// transitive-closure pass to backfill private callees with minimal
	// entries when they're referenced by an emitted function's @calls.
	type fnRef struct {
		decl      *parser.FnDecl
		qualifier string
		file      string
	}
	allFns := map[string]*fnRef{}
	registerFn := func(fn *parser.FnDecl, qualifier, file string) {
		key := fn.Name
		if qualifier != "" {
			key = qualifier + "." + fn.Name
		}
		if _, exists := allFns[key]; !exists {
			allFns[key] = &fnRef{decl: fn, qualifier: qualifier, file: file}
		}
	}

	// First pass: collect type / trait / const / fn declarations (non-method).
	for i, prog := range progs {
		if prog == nil {
			continue
		}
		file := files[i]
		rel := relPath(dir, file)

		for _, d := range prog.Decls {
			switch dd := d.(type) {
			case *parser.FnDecl:
				registerFn(dd, "", rel)
				if !emit(dd.Vis) {
					continue
				}
				aid.Entries = append(aid.Entries, extractFn(dd, "", rel, docs))

			case *parser.TypeDecl:
				if !emit(dd.Vis) {
					continue
				}
				typeNames[dd.Name] = true
				aid.Entries = append(aid.Entries, extractTypeDecl(dd, rel, docs))

			case *parser.EnumDecl:
				if !emit(dd.Vis) {
					continue
				}
				typeNames[dd.Name] = true
				aid.Entries = append(aid.Entries, extractEnum(dd, rel, docs))

			case *parser.AliasDecl:
				if !emit(dd.Vis) {
					continue
				}
				typeNames[dd.Name] = true
				aid.Entries = append(aid.Entries, extractAlias(dd, rel, docs))

			case *parser.TraitDecl:
				for _, m := range dd.Methods {
					registerFn(m, dd.Name, rel)
				}
				if !emit(dd.Vis) {
					continue
				}
				typeNames[dd.Name] = true
				trait, methods := extractTrait(dd, rel, docs)
				aid.Entries = append(aid.Entries, trait)
				methodBuckets = append(methodBuckets, methods...)

			case *parser.ConstDecl:
				if !emit(dd.Vis) {
					continue
				}
				aid.Entries = append(aid.Entries, extractConst(dd, rel, docs))

			case *parser.ImplDecl:
				for _, m := range dd.Methods {
					registerFn(m, dd.TypeName, rel)
				}
				methods := extractImpl(dd, rel, docs)
				// Record trait-implementation edges on the owning TypeEntry.
				// Category-trait impls (Transient, Retryable, …) are also
				// attributed to @error_categories for mechanical retry/escalation
				// policies; they remain in @implements as well.
				if dd.TraitName != "" {
					annotateImplements(aid.Entries, dd.TypeName, dd.TraitName)
					if isKnownErrorCategory(dd.TraitName) {
						annotateErrorCategory(aid.Entries, dd.TypeName, dd.TraitName)
					}
				}
				methodBuckets = append(methodBuckets, methods...)
			}
		}
	}

	// Append methods after types so ordering (consts → types+methods → traits → fns)
	// in emitter.orderEntries picks them up by name prefix. Impl methods are
	// always emitted: per-decl visibility on a method is rare in Aria, and
	// the owning type having passed the emit() filter implies its API surface.
	for _, m := range methodBuckets {
		aid.Entries = append(aid.Entries, m)
	}

	// Transitive closure: when -internal is off, backfill minimal entries for
	// private functions that are referenced via @calls from any emitted fn.
	// This keeps downstream call-graph consumers (cartograph) whole even
	// though the entries carry only name + sig + calls + position.
	//
	// Skipped when -internal or -all is set, because those modes already
	// emit every fn with full detail.
	if !opts.Internal && !opts.All {
		emitted := map[string]bool{}
		for _, e := range aid.Entries {
			if fn, ok := e.(FnEntry); ok {
				emitted[fn.Name] = true
			}
		}
		for {
			var added []FnEntry
			for _, e := range aid.Entries {
				fn, ok := e.(FnEntry)
				if !ok {
					continue
				}
				for _, callee := range fn.Calls {
					if emitted[callee] {
						continue
					}
					ref, exists := allFns[callee]
					if !exists {
						continue // external / unresolved — not ours to backfill
					}
					added = append(added, extractFnMinimal(ref.decl, ref.qualifier, ref.file))
					emitted[callee] = true
				}
			}
			if len(added) == 0 {
				break
			}
			for _, m := range added {
				aid.Entries = append(aid.Entries, m)
			}
		}
	}

	// Stable ordering within each kind group — emitter re-orders by kind but
	// preserves our within-group order, so sort by name for deterministic output.
	sort.SliceStable(aid.Entries, func(i, j int) bool {
		return entryName(aid.Entries[i]) < entryName(aid.Entries[j])
	})

	return aid, nil
}

// ExtractTests produces an AidFile containing Aria test-scaffolding: Mock*/
// Stub*/Fake*/Spy* types, helper functions, and a synthetic entry per
// `test "name" { }` block with its @calls → production edges.
func (a *AriaExtractor) ExtractTests(dir, testModName, version string) (*AidFile, error) {
	files, err := listAriaFiles(dir, true)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return extractTestsFromPrograms(nil, nil, dir, testModName, version), nil
	}

	progs := make([]*parser.Program, 0, len(files))
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		progs = append(progs, parser.Parse(f, string(src)))
	}

	aid := extractTestsFromPrograms(progs, files, dir, testModName, version)
	if len(aid.Entries) == 0 {
		return nil, fmt.Errorf("no test scaffolding found in %s", dir)
	}
	return aid, nil
}

// ---

// listAriaFiles returns all .aria files in dir (non-recursive, matching
// aid-gen-go's per-package granularity).
func listAriaFiles(dir string, includeTests bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".aria") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	return files, nil
}

func relPath(base, file string) string {
	if r, err := filepath.Rel(base, file); err == nil {
		return r
	}
	return file
}

func entryName(e Entry) string {
	switch v := e.(type) {
	case FnEntry:
		return v.Name
	case TypeEntry:
		return v.Name
	case TraitEntry:
		return v.Name
	case ConstEntry:
		return v.Name
	}
	return ""
}

// annotateImplements finds a TypeEntry with the given name and appends the
// trait name to its Implements list. No-op if the type is not (yet) in the
// entry slice (e.g. type declared in another file processed later — rare for
// single-package extraction).
func annotateImplements(entries []Entry, typeName, traitName string) {
	for i, e := range entries {
		if te, ok := e.(TypeEntry); ok && te.Name == typeName {
			for _, existing := range te.Implements {
				if existing == traitName {
					return
				}
			}
			te.Implements = append(te.Implements, traitName)
			entries[i] = te
			return
		}
	}
}
