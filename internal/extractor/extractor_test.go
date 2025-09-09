package extractor

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// helper to create a single-file gzip
func makeSingleGzip(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	gw := gzip.NewWriter(f)
	gw.Name = "inner.txt"
	if _, err := gw.Write([]byte(content)); err != nil {
		t.Fatalf("write: %v", err)
	}
	gw.Close()
	f.Close()
	return path
}

// helper to create a tar.gz with nested gzip inside
func makeTarWithNested(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// nested gzip file bytes
	nestedPath := filepath.Join(dir, "nested.gz")
	nf, _ := os.Create(nestedPath)
	ngw := gzip.NewWriter(nf)
	ngw.Name = "deep.txt"
	ngw.Write([]byte("deep data"))
	ngw.Close()
	nf.Close()
	nestedBytes, _ := os.ReadFile(nestedPath)

	files := map[string][]byte{
		"a/b/c.txt":        []byte("hello"),
		"a/nested.gz":      nestedBytes,
		"top.txt":          []byte("root"),
		"../traversal.txt": []byte("blocked"),
	}
	for name, data := range files {
		hdr := &tar.Header{Name: name, Size: int64(len(data)), Mode: 0o644}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("hdr: %v", err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatalf("data: %v", err)
		}
	}
	tw.Close()
	gw.Close()
	f.Close()
	os.Remove(nestedPath)
	return path
}

func TestIsGzip(t *testing.T) {
	dir := t.TempDir()
	gz := makeSingleGzip(t, dir, "sample.bar", "data")
	ok, err := IsGzip(gz)
	if err != nil || !ok {
		t.Fatalf("expected gzip true got %v err %v", ok, err)
	}
}

func TestExtractSingleFile(t *testing.T) {
	dir := t.TempDir()
	gz := makeSingleGzip(t, dir, "one.bar", "hello world")
	outs, err := ExtractFile(gz, NewDefaultConfig())
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(outs) != 1 {
		t.Fatalf("expected 1 output got %d", len(outs))
	}
	b, _ := os.ReadFile(outs[0])
	if string(b) != "hello world" {
		t.Fatalf("content mismatch: %q", string(b))
	}
}

func TestExtractTarWithNested(t *testing.T) {
	dir := t.TempDir()
	tarGz := makeTarWithNested(t, dir, "arc.bar")
	cfg := NewDefaultConfig()
	outs, err := ExtractFile(tarGz, cfg)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(outs) == 0 {
		t.Fatalf("expected outputs")
	}
	// ensure traversal path did not escape outside (no plain traversal.txt at root of temp dir unless sanitized)
	if _, err := os.Stat(filepath.Join(dir, "traversal.txt")); err == nil {
		t.Fatalf("unexpected traversal.txt at root (path traversal not blocked)")
	}
	// now recursive nested extraction
	if err := ExtractNested(outs, cfg); err != nil {
		t.Fatalf("nested: %v", err)
	}
	// deep.txt should exist somewhere
	var found bool
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Base(path) == "deep.txt" {
			found = true
		}
		return nil
	})
	if !found {
		t.Fatalf("expected deep.txt from nested gzip")
	}
}

func TestResolveNameConflict(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(p, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	np, err := ResolveNameConflict(p)
	if err != nil {
		t.Fatal(err)
	}
	if np == p {
		t.Fatalf("expected different path")
	}
	if filepath.Base(np) != "file-1.txt" {
		t.Fatalf("unexpected conflict name: %s", np)
	}
}

func TestSanitizePath(t *testing.T) {
	cases := map[string]string{
		"../etc/passwd": "_", // becomes underscore fallback
		"./a/b":         "a/b",
		"a/../../b":     "_", // now collapses due to traversal
	}
	for in, expect := range cases {
		out := SanitizePath(in)
		if out != expect {
			t.Fatalf("sanitize %q => %q want %q", in, out, expect)
		}
	}
}

// Ensure single-file extraction respects provided gzip name
func TestGzipHeaderName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "named.bar")
	f, _ := os.Create(path)
	gw := gzip.NewWriter(f)
	gw.Name = "custom.txt"
	io.WriteString(gw, "x")
	gw.Close()
	f.Close()
	outs, err := ExtractFile(path, NewDefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 || filepath.Base(outs[0]) != "custom.txt" {
		t.Fatalf("expected custom.txt got %#v", outs)
	}
}
