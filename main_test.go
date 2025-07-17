package main

import (
	"archive/zip"
	"os"
	"testing"
)

func TestValidateFile_NotExist(t *testing.T) {
	err := ValidateFile("/tmp/doesnotexist.bar")
	if err == nil {
		t.Errorf("Expected error for non-existent file")
	}
}

func TestValidateFile_NotZipGzip(t *testing.T) {
	f, err := os.CreateTemp("", "notzip")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.Close()
	err = ValidateFile(f.Name())
	if err == nil {
		t.Errorf("Expected error for non-zip/gzip file")
	}
}

func TestValidateFile_Zip(t *testing.T) {
	barFile := createTestBar(t)
	defer os.Remove(barFile)
	err := ValidateFile(barFile)
	if err != nil {
		t.Errorf("Expected .bar file to validate, got error: %v", err)
	}
}

func createTestBar(t *testing.T) string {
	f, err := os.CreateTemp("", "testbar*.bar")
	if err != nil {
		t.Fatalf("failed to create temp .bar file: %v", err)
	}
	w := zip.NewWriter(f)
	_, err = w.Create("test.appzip")
	if err != nil {
		w.Close()
		f.Close()
		t.Fatalf("failed to create .appzip entry: %v", err)
	}
	w.Close()
	f.Close()
	return f.Name()
}
