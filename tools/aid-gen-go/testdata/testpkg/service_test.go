package testpkg

import "testing"

// mockBundleService is a test mock implementing BundleService.
type mockBundleService struct {
	getBundleResult string
	getPageResult   string
}

func (m *mockBundleService) GetBundleByName(database string, name string) (string, error) {
	return m.getBundleResult, nil
}

func (m *mockBundleService) GetDocumentPage(bundleName string, pageID int) (string, error) {
	return m.getPageResult, nil
}

// stubIndex is a no-op Index for testing.
type stubIndex struct{}

func (s *stubIndex) Close() error { return nil }
func (s *stubIndex) Flush() error { return nil }

// SetupTestIndex is a test helper that creates an index for testing.
func SetupTestIndex() Index {
	return &stubIndex{}
}

func TestGetBundleByName(t *testing.T) {
	m := &mockBundleService{getBundleResult: "test-bundle"}
	result, err := m.GetBundleByName("db", "name")
	if err != nil {
		t.Fatal(err)
	}
	if result != "test-bundle" {
		t.Fatalf("got %s, want test-bundle", result)
	}
}

func BenchmarkFlush(b *testing.B) {
	idx := &stubIndex{}
	for i := 0; i < b.N; i++ {
		idx.Flush()
	}
}
