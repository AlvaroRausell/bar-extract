package extractor

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config holds runtime options controlling extraction behavior.
type Config struct {
	DryRun           bool // simulate actions without writing files
	Verbose          bool // print detailed logging
	Recursive        bool // recursively extract nested gzip files
	RemoveCompressed bool // remove .gz files after successful extraction
	AllowSymlinks    bool // allow creation of symlinks from tar archives
}

// NewDefaultConfig returns a config with sensible defaults.
func NewDefaultConfig() Config { return Config{Recursive: true} }

// IsGzip checks the file header magic numbers (0x1f, 0x8b) to decide if a file is gzip compressed.
// It only reads the first two bytes. Caller should handle errors.
func IsGzip(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	var hdr [2]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return false, nil
		}
		return false, err
	}
	return hdr[0] == 0x1f && hdr[1] == 0x8b, nil
}

// sanitizePath cleans a path coming from inside an archive avoiding path traversal and
// absolute paths. It returns a relative, safe path.
func SanitizePath(p string) string {
	p = filepath.Clean(p)
	// Remove drive letters on Windows-like paths (e.g., C:..)
	if len(p) > 1 && p[1] == ':' {
		p = p[2:]
	}
	// Strip any leading separators to force relative path
	p = strings.TrimLeft(p, string(filepath.Separator))
	// If any parent traversal remains, collapse to underscore for safety
	parts := strings.Split(p, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return "_"
		}
	}
	if p == "" || p == "." { // fallback
		return "_" // underscore placeholder
	}
	return p
}

// resolveNameConflict ensures we do not overwrite an existing path.
// If the candidate path does not exist it is returned unmodified.
// Otherwise it appends -1, -2, ... before the extension until a free name is found.
func ResolveNameConflict(basePath string) (string, error) {
	if _, err := os.Lstat(basePath); err != nil {
		if os.IsNotExist(err) {
			return basePath, nil
		}
		return "", err
	}
	dir := filepath.Dir(basePath)
	name := filepath.Base(basePath)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i < 10_000; i++ { // arbitrary upper bound
		cand := filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, i, ext))
		if _, err := os.Lstat(cand); err != nil {
			if os.IsNotExist(err) {
				return cand, nil
			}
			return "", err
		}
	}
	return "", fmt.Errorf("could not resolve unique name for %s after many attempts", basePath)
}

// ExtractFile is the entrypoint for expanding a .bar (gzip) file. It returns the root output paths created.
func ExtractFile(path string, cfg Config) ([]string, error) {
	is, err := IsGzip(path)
	if err != nil {
		return nil, err
	}
	if !is {
		return nil, fmt.Errorf("%s is not recognized as gzip", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	br := bufio.NewReader(gz)
	// Peek to detect tar: need at least 512 bytes for standard header.
	peek, _ := br.Peek(512)
	isTar := false
	if len(peek) >= 512 && len(peek) >= 257+6 {
		if strings.Contains(string(peek[257:257+6]), "ustar") { // posix tar magic
			isTar = true
		}
	}
	// We'll reconstruct a reader that includes the peeked bytes.
	var outputs []string
	if isTar {
		tr := tar.NewReader(br)
		for {
			hdr, err := tr.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return outputs, err
			}
			if hdr == nil {
				continue
			}
			name := SanitizePath(hdr.Name)
			if name == "_" { // skip empty names silently
				continue
			}
			target := filepath.Join(filepath.Dir(path), name)
			switch hdr.Typeflag {
			case tar.TypeDir:
				if cfg.Verbose {
					fmt.Fprintf(os.Stderr, "dir  %s\n", target)
				}
				if cfg.DryRun {
					continue
				}
				if err := os.MkdirAll(target, 0o755); err != nil {
					return outputs, err
				}
			case tar.TypeReg, tar.TypeRegA:
				if cfg.Verbose {
					fmt.Fprintf(os.Stderr, "file %s (%d bytes)\n", target, hdr.Size)
				}
				if cfg.DryRun {
					continue
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return outputs, err
				}
				target, err = ResolveNameConflict(target)
				if err != nil {
					return outputs, err
				}
				of, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_EXCL, fs.FileMode(hdr.Mode)&0o777)
				if err != nil {
					return outputs, err
				}
				if _, err := io.Copy(of, tr); err != nil {
					of.Close()
					return outputs, err
				}
				of.Close()
				outputs = append(outputs, target)
			case tar.TypeSymlink, tar.TypeLink:
				if !cfg.AllowSymlinks {
					if cfg.Verbose {
						fmt.Fprintf(os.Stderr, "skip symlink %s -> %s\n", target, hdr.Linkname)
					}
					continue
				}
				if cfg.DryRun {
					continue
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return outputs, err
				}
				// best-effort create symlink
				if err := os.Symlink(hdr.Linkname, target); err != nil {
					if cfg.Verbose {
						fmt.Fprintf(os.Stderr, "warn: symlink create failed: %v\n", err)
					}
				}
			default:
				if cfg.Verbose {
					fmt.Fprintf(os.Stderr, "skip unknown type %c %s\n", hdr.Typeflag, target)
				}
			}
		}
	} else {
		// Single file. Derive name from gzip header if provided.
		name := gz.Name
		if name == "" {
			base := filepath.Base(path)
			// strip .bar, .gz, .gzip if present
			for _, ext := range []string{".bar", ".gz", ".gzip"} {
				if strings.HasSuffix(strings.ToLower(base), ext) {
					base = base[:len(base)-len(ext)]
					break
				}
			}
			name = base
		}
		name = SanitizePath(name)
		target := filepath.Join(filepath.Dir(path), name)
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "file %s (single)\n", target)
		}
		if !cfg.DryRun {
			target, err = ResolveNameConflict(target)
			if err != nil {
				return outputs, err
			}
			of, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
			if err != nil {
				return outputs, err
			}
			if _, err := io.Copy(of, br); err != nil {
				of.Close()
				return outputs, err
			}
			of.Close()
		}
		outputs = append(outputs, target)
	}
	return outputs, nil
}

// ExtractNested scans root directories for additional gzip (.gz) files and extracts them.
func ExtractNested(paths []string, cfg Config) error {
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			return err
		}
		walkRoot := root
		if !info.IsDir() {
			walkRoot = filepath.Dir(root)
		}
		filepath.WalkDir(walkRoot, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			lower := strings.ToLower(p)
			if !(strings.HasSuffix(lower, ".gz")) {
				return nil
			}
			ok, err := IsGzip(p)
			if err != nil || !ok {
				return nil
			}
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "nested gzip: %s\n", p)
			}
			outs, err := ExtractFile(p, cfg)
			if err != nil {
				return err
			}
			if cfg.RemoveCompressed && !cfg.DryRun {
				_ = os.Remove(p)
			}
			// Recurse into any new outputs (if they are directories they will be walked anyway)
			_ = outs
			return nil
		})
	}
	return nil
}
