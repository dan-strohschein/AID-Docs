// Package callchain provides test data for transitive callee emission tests.
package callchain

// Exported calls an unexported helper to produce a string.
func Exported() string {
	return helper()
}

func helper() string {
	return deepHelper()
}

func deepHelper() string {
	return "deep"
}
