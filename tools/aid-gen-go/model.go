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
	Calls      []string // Functions/methods this function calls
	Pre        string
	Post       string
	Effects    []string
	ThreadSafe string
	Complexity string
	Since      string
	Deprecated string
	Related    []string
	Example    string
	SourceFile string // Relative path to source file
	SourceLine int    // Line number of definition
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

// TypeEntry describes a struct, enum, alias, or newtype.
type TypeEntry struct {
	Name         string
	Kind         string // struct, enum, union, class, alias, newtype
	Purpose      string
	Fields       []Field
	Variants     []Variant
	Invariants   []string
	Constructors string
	Methods      []string
	Extends      []string
	Implements   []string
	GenericParams string
	Since        string
	Deprecated   string
	Related      []string
}

func (TypeEntry) entryKind() string { return "type" }

// Field describes a field on a struct/class type.
type Field struct {
	Name string
	Type string
	Desc string
}

// Variant describes an enum variant.
type Variant struct {
	Name    string
	Payload string
	Desc    string
}

// TraitEntry describes an interface.
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
