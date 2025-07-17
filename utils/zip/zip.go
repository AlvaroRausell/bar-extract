package zip

import (
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

func IsZipOrGzip(filePath string) bool {
	return isZipFile(filePath) || isGzipFile(filePath)
}

func isZipFile(filePath string) bool {
	header, ok := readHeader(filePath, 4)
	if !ok {
		return false
	}
	return header[0] == 'P' && header[1] == 'K' && header[2] == 0x03 && header[3] == 0x04
}

func isGzipFile(filePath string) bool {
	header, ok := readHeader(filePath, 2)
	if !ok {
		return false
	}
	return header[0] == 0x1F && header[1] == 0x8B
}

func readHeader(filePath string, n int) ([]byte, bool) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	header := make([]byte, n)
	read, err := f.Read(header)
	if err != nil || read < n {
		return nil, false
	}
	return header, true
}

func ExtractRecursive(filePath string) error {
	log.Printf("Extracting file: %v", filePath)
	var err error
	if isZipFile(filePath) {
		err = extractZipRecursive(filePath)
	} else {
		err = extractGzipRecursive(filePath)
	}
	// Pretty-print the extracted structure
	if err == nil {
		baseDestination := removeAllExtensions(filePath)
		printDirStructure(baseDestination, "")
	}
	return err
}

// Pretty-print directory structure
func printDirStructure(root string, prefix string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Printf("%s[error reading dir: %v]", prefix, err)
		return
	}
	for _, entry := range entries {
		log.Printf("%s%s", prefix, entry.Name())
		if entry.IsDir() {
			printDirStructure(path.Join(root, entry.Name()), prefix+"  ")
		}
	}
}

func extractZipRecursive(filePath string) error {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	baseDestination := removeAllExtensions(filePath)
	if err := os.MkdirAll(baseDestination, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	for _, f := range archive.File {
		destPath := path.Join(baseDestination, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}
		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
		// Unified recursive extraction
		if IsZipOrGzip(destPath) {
			if err := ExtractRecursive(destPath); err != nil {
				return fmt.Errorf("failed to recursively extract %s: %w", destPath, err)
			}
		}
	}
	return nil
}

func extractGzipRecursive(filePath string) error {
	baseDestination := removeAllExtensions(filePath)
	if err := os.MkdirAll(baseDestination, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Decompress gzip file to a single file in the destination
	gzFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open gzip file: %w", err)
	}
	defer gzFile.Close()
	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Use the gzip header name if available, otherwise use the base name
	outName := gzReader.Name
	if outName == "" {
		outName = path.Base(baseDestination)
	}
	destPath := path.Join(baseDestination, outName)
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create decompressed file: %w", err)
	}
	_, err = io.Copy(outFile, gzReader)
	outFile.Close()
	if err != nil {
		return fmt.Errorf("failed to decompress gzip: %w", err)
	}

	// Unified recursive extraction
	if IsZipOrGzip(destPath) {
		if err := ExtractRecursive(destPath); err != nil {
			return fmt.Errorf("failed to recursively extract %s: %w", destPath, err)
		}
	}
	return nil
}

// Helper to remove all extensions from a file path
func removeAllExtensions(filePath string) string {
	base := filePath
	for ext := path.Ext(base); ext != ""; ext = path.Ext(base) {
		base = strings.TrimSuffix(base, ext)
	}
	return base
}
