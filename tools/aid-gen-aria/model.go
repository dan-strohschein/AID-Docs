package main

// AidFile represents a complete AID document.
type AidFile struct {
	Header    ModuleHeader
	Entries   []Entry
	Workflows []Workflow
}

// ModuleHeader is the module-level metadata.
type ModuleHeader struct {
	Module     string
	Lang       string
	Version    string
	Stability  string
	Purpose    string
	Deps       []string
	Source     string
	AidVersion string
}

// Entry is one of FnEntry, TypeEntry, TraitEntry, or ConstEntry.
type Entry interface {
	entryKind() string
}

// FnEntry describes a function or method.
type FnEntry struct {
	Name       string
	Purpose    string
	Sigs       []string
	Params     []Param
	Returns    string
	Errors     []string
	Calls      []string
	Pre        string
	Post       string
	Effects    []string
	ThreadSafe string
	Complexity string
	Since      string
	Deprecated string
	Related    []string
	Example    string
	SourceFile string
	SourceLine int
}

func (FnEntry) entryKind() string { return "fn" }

// Param describes a function parameter.
type Param struct {
	Name      string
	Type      string
	Desc      string
	Default   string
	Variadic  bool
	SubParams []Param
}

// TypeEntry describes a struct, enum, alias, newtype, or sum type.
type TypeEntry struct {
	Name          string
	Kind          string
	Purpose       string
	Fields        []Field
	Variants      []Variant
	Invariants    []string
	Constructors  string
	Methods       []string
	Extends       []string
	Implements    []string
	GenericParams string
	Since         string
	Deprecated    string
	Related         []string
	ErrorCategories []string // well-known: Transient, Permanent, UserFault, SystemFault, Retryable
	SourceFile      string
	SourceLine      int
}

func (TypeEntry) entryKind() string { return "type" }

// Field describes a field on a struct/class type.
type Field struct {
	Name string
	Type string
	Desc string
}

// Variant describes a sum-type / enum variant.
type Variant struct {
	Name    string
	Payload string
	Desc    string
}

// TraitEntry describes a trait / interface.
type TraitEntry struct {
	Name         string
	Purpose      string
	Requires     []string
	Provided     []string
	Implementors []string
	Extends      []string
	Related      []string
}

func (TraitEntry) entryKind() string { return "trait" }

// ConstEntry describes a constant or sentinel error.
type ConstEntry struct {
	Name    string
	Purpose string
	Type    string
	Value   string
	Since   string
}

func (ConstEntry) entryKind() string { return "const" }

// Workflow describes a multi-step usage pattern.
type Workflow struct {
	Name         string
	Purpose      string
	Steps        []string
	ErrorsAt     []string
	Antipatterns []string
	Variants     []string
	Example      string
}
