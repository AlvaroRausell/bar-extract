package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlvaroRausell/bar-extract/internal/extractor"
)

// version is set via -ldflags "-X main.version=vX.Y.Z" at build time.
var version = "dev"

func main() {
	cfg := extractor.NewDefaultConfig()
	showVersion := flag.Bool("version", false, "print version and exit")
	dryRun := flag.Bool("dry-run", false, "simulate actions without writing")
	noRecursive := flag.Bool("no-recursive", false, "disable nested gzip extraction")
	removeCompressed := flag.Bool("remove-compressed", false, "remove .gz files after successful nested extraction")
	verbose := flag.Bool("verbose", false, "verbose logging to stderr")
	allowSymlinks := flag.Bool("allow-symlinks", false, "allow creation of symlinks from tar archives")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: bar-extract [flags] <file.bar> [more.bar ...]")
		flag.PrintDefaults()
		os.Exit(1)
	}
	cfg.DryRun = *dryRun
	cfg.Verbose = *verbose
	cfg.Recursive = !*noRecursive
	cfg.RemoveCompressed = *removeCompressed
	cfg.AllowSymlinks = *allowSymlinks

	exitCode := 0
	for _, in := range flag.Args() {
		// Expand globs if provided by shell on some platforms? We'll handle patterns manually.
		matches, _ := filepath.Glob(in)
		if len(matches) == 0 {
			matches = []string{in}
		}
		for _, m := range matches {
			outs, err := extractor.ExtractFile(m, cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				exitCode = 2
				continue
			}
			if cfg.Verbose {
				for _, o := range outs {
					fmt.Fprintf(os.Stderr, "-> %s\n", o)
				}
			}
			if cfg.Recursive && !cfg.DryRun {
				if err := extractor.ExtractNested(outs, cfg); err != nil {
					fmt.Fprintf(os.Stderr, "nested error: %v\n", err)
					exitCode = 2
				}
			}
		}
	}
	os.Exit(exitCode)
}
