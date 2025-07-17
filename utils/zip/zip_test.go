package zip

import (
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pretty-print directory structure
func printDirStructureTest(t *testing.T, root string, prefix string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Logf("%s[error reading dir: %v]", prefix, err)
		return
	}
	for _, entry := range entries {
		t.Logf("%s%s", prefix, entry.Name())
		if entry.IsDir() {
			printDirStructureTest(t, filepath.Join(root, entry.Name()), prefix+"  ")
		}
	}
}

func createNestedZip(t *testing.T, baseDir string) string {
	// Create a nested zip file inside another zip
	nestedAppZipPath := filepath.Join(baseDir, "nested.appzip")
	nestedAppZipFile, err := os.Create(nestedAppZipPath)
	if err != nil {
		t.Fatalf("failed to create nested zip: %v", err)
	}
	nestedAppZipWriter := zip.NewWriter(nestedAppZipFile)
	_, err = nestedAppZipWriter.Create("file_in_nested.txt")
	if err != nil {
		t.Fatalf("failed to create file in nested appzip: %v", err)
	}
	nestedAppZipWriter.Close()
	nestedAppZipFile.Close()

	// Create the outer zip file
	outerBarPath := filepath.Join(baseDir, "outer.bar")
	outerBarFile, err := os.Create(outerBarPath)
	if err != nil {
		t.Fatalf("failed to create outer zip: %v", err)
	}
	outerBarWriter := zip.NewWriter(outerBarFile)
	// Add a regular file
	f1, err := outerBarWriter.Create("file1.txt")
	if err != nil {
		t.Fatalf("failed to create file1.txt in outer bar: %v", err)
	}
	f1.Write([]byte("hello world"))
	// Add the nested appzip file
	nestedAppZipData, err := os.ReadFile(nestedAppZipPath)
	if err != nil {
		t.Fatalf("failed to read nested appzip: %v", err)
	}
	f2, err := outerBarWriter.Create("nested.appzip")
	if err != nil {
		t.Fatalf("failed to create nested.appzip in outer bar: %v", err)
	}
	f2.Write(nestedAppZipData)
	outerBarWriter.Close()
	outerBarFile.Close()
	return outerBarPath
}

func TestExtractRecursive_ZipNested(t *testing.T) {
	tempDir := t.TempDir()
	barPath := createNestedZip(t, tempDir)
	err := ExtractRecursive(barPath)
	if err != nil {
		t.Fatalf("ExtractRecursive failed: %v", err)
	}
	t.Logf("Extracted structure for %s:", barPath)
	printDirStructureTest(t, tempDir, "")
	// Check for expected files
	extractedDir := strings.TrimSuffix(barPath, ".bar")
	if _, err := os.Stat(filepath.Join(extractedDir, "file1.txt")); err != nil {
		t.Errorf("file1.txt not found in extracted bar")
	}
	if _, err := os.Stat(filepath.Join(extractedDir, "nested.appzip")); err != nil {
		t.Errorf("nested.appzip not found in extracted bar")
	}
	nestedExtractedDir := strings.TrimSuffix(filepath.Join(extractedDir, "nested.appzip"), ".appzip")
	if _, err := os.Stat(filepath.Join(nestedExtractedDir, "file_in_nested.txt")); err != nil {
		t.Errorf("file_in_nested.txt not found in nested extracted appzip")
	}
}

func createTestFile(t *testing.T, name string, data []byte) string {
	f, err := os.CreateTemp("", name)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_, err = f.Write(data)
	if err != nil {
		f.Close()
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func createZipFile(t *testing.T) string {
	f, err := os.CreateTemp("", "testzip*.zip")
	if err != nil {
		t.Fatalf("failed to create temp zip file: %v", err)
	}
	w := zip.NewWriter(f)
	_, err = w.Create("test.txt")
	if err != nil {
		w.Close()
		f.Close()
		t.Fatalf("failed to create zip entry: %v", err)
	}
	w.Close()
	f.Close()
	return f.Name()
}

func createGzipFile(t *testing.T) string {
	f, err := os.CreateTemp("", "testgzip*.gz")
	if err != nil {
		t.Fatalf("failed to create temp gzip file: %v", err)
	}
	gw := gzip.NewWriter(f)
	_, err = gw.Write([]byte("test"))
	if err != nil {
		gw.Close()
		f.Close()
		t.Fatalf("failed to write gzip data: %v", err)
	}
	gw.Close()
	f.Close()
	return f.Name()
}

func TestIsZipOrGzip(t *testing.T) {
	zipFile := createZipFile(t)
	defer os.Remove(zipFile)
	gzipFile := createGzipFile(t)
	defer os.Remove(gzipFile)
	notZipGzip := []byte{0x00, 0x00, 0x00, 0x00}
	otherFile := createTestFile(t, "testother", notZipGzip)
	defer os.Remove(otherFile)

	if !IsZipOrGzip(zipFile) {
		t.Errorf("Expected zip file to be detected as zip/gzip")
	}
	if !IsZipOrGzip(gzipFile) {
		t.Errorf("Expected gzip file to be detected as zip/gzip")
	}
	if IsZipOrGzip(otherFile) {
		t.Errorf("Expected non-zip/gzip file to not be detected as zip/gzip")
	}
}
