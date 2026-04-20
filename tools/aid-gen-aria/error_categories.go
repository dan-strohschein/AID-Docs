package main

// knownErrorCategories is the well-known category-trait vocabulary from
// spec/fields.md § "Error category traits". Any `impl <name> for <TypeName>`
// whose trait name appears in this set attributes the category to TypeName's
// TypeEntry under @error_categories.
//
// Unknown trait names are ignored here (they already surface as plain
// @implements edges on the target type). Projects can still introduce their
// own categories by inspecting @implements downstream.
var knownErrorCategories = map[string]bool{
	"Transient":   true,
	"Permanent":   true,
	"UserFault":   true,
	"SystemFault": true,
	"Retryable":   true,
}

// isKnownErrorCategory reports whether a trait name is one of the well-known
// error categories defined in the AID spec.
func isKnownErrorCategory(name string) bool { return knownErrorCategories[name] }

// annotateErrorCategory appends a category trait name to the TypeEntry named
// typeName if it exists in entries. Dedupes and is a no-op if the type is
// absent (e.g. an impl for a type the visibility filter excluded).
func annotateErrorCategory(entries []Entry, typeName, category string) {
	for i, e := range entries {
		te, ok := e.(TypeEntry)
		if !ok || te.Name != typeName {
			continue
		}
		for _, existing := range te.ErrorCategories {
			if existing == category {
				return
			}
		}
		te.ErrorCategories = append(te.ErrorCategories, category)
		entries[i] = te
		return
	}
}
