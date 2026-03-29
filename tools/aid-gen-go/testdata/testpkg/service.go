// Package testpkg is a fixture for testing test-package AID extraction.
package testpkg

// BundleService provides access to bundles.
type BundleService interface {
	GetBundleByName(database string, name string) (string, error)
	GetDocumentPage(bundleName string, pageID int) (string, error)
}

// Index is a generic index interface.
type Index interface {
	Close() error
	Flush() error
}
