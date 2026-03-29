// Discovery tests verify walk-up discovery of .aidocs/, listing of .aid files, and manifest.aid parsing.
package discovery

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// minimalManifestAID is valid manifest content that parser.ParseFile accepts.
const minimalManifestAID = `@manifest
@project TestProj
@aid_version 0.1

---

@package demo/pkg
@aid_file demo.aid
@purpose Demo package
@layer l1
`

func TestDiscover_WalksUpToAidocs(t *testing.T) {
	root := t.TempDir()
	aidocs := filepath.Join(root, AidDocsDir)
	if err := os.MkdirAll(aidocs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aidocs, "lib.aid"), []byte("@module lib\n@lang go\n@version 1.0.0\n@aid_version 0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "sub", "deep")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Discover(nested)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil Result")
	}
	if res.AidDocsPath != aidocs {
		t.Errorf("AidDocsPath = %q, want %q", res.AidDocsPath, aidocs)
	}
	if !slices.Contains(res.AidFiles, "lib.aid") {
		t.Errorf("AidFiles = %v, want lib.aid", res.AidFiles)
	}
	if res.ManifestPath != "" {
		t.Errorf("ManifestPath = %q, want empty", res.ManifestPath)
	}
	if res.Manifest != nil {
		t.Error("expected no manifest")
	}
}

func TestDiscover_WithManifest(t *testing.T) {
	root := t.TempDir()
	aidocs := filepath.Join(root, AidDocsDir)
	if err := os.MkdirAll(aidocs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aidocs, ManifestFile), []byte(minimalManifestAID), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aidocs, "other.aid"), []byte("@module other\n@lang go\n@version 1.0.0\n@aid_version 0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil Result")
	}
	manifestPath := filepath.Join(aidocs, ManifestFile)
	if res.ManifestPath != manifestPath {
		t.Errorf("ManifestPath = %q, want %q", res.ManifestPath, manifestPath)
	}
	if res.Manifest == nil {
		t.Fatal("expected parsed Manifest")
	}
	if !res.Manifest.IsManifest {
		t.Error("expected IsManifest on parsed manifest")
	}
	if len(res.Manifest.Entries) != 1 {
		t.Fatalf("manifest entries = %d, want 1", len(res.Manifest.Entries))
	}
	if !slices.Contains(res.AidFiles, ManifestFile) || !slices.Contains(res.AidFiles, "other.aid") {
		t.Errorf("AidFiles = %v, want manifest.aid and other.aid", res.AidFiles)
	}
}

func TestDiscover_NoAidocs(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Discover(sub)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil Result when no .aidocs, got %+v", res)
	}
}

func TestDiscover_StartDirIsAidocsParent(t *testing.T) {
	root := t.TempDir()
	aidocs := filepath.Join(root, AidDocsDir)
	if err := os.MkdirAll(aidocs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aidocs, "x.aid"), []byte("@module x\n@lang go\n@version 1.0.0\n@aid_version 0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if res == nil || res.AidDocsPath != aidocs {
		t.Fatalf("unexpected result: %+v", res)
	}
}
