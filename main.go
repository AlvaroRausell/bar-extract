package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ot4i/bar-extract/utils/zip"
)

func main() {
	logFile, err := os.OpenFile("bar-extract.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error creating log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(os.Args) < 2 {
		log.Fatalf("Argument Exception: no target file provided")
	}
	filePath := os.Args[1]
	log.Printf("Opening: %v", filePath)
	if err := ValidateFile(filePath); err != nil {
		log.Fatal(err)
	}
	if err := zip.ExtractRecursive(filePath); err != nil {
		log.Fatalf("Extraction failed: %v", err)
	}
}

func ValidateFile(filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		return err
	}
	if !zip.IsZipOrGzip(filePath) {
		return fmt.Errorf("file %v is not zip/gzip", filePath)
	}
	return nil
}
