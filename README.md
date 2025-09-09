# bar-extract
This repository provides code for a BAR file extraction utility for MacOS.  With this utility, users can double-click on a BAR file in the MacOS Finder and the tool will recursively unzip contents of the BAR such as .appzip files.
bar-extract
===========

Utility to expand .BAR (gzip-based) archives and recursively extract nested gzip files, producing an output tree mirroring the compressed structure.

Why
----
You have files with a `.bar` extension that are really gzip streams (sometimes containing tar archives) and you want a simple double‑click / CLI tool to unpack them (and any nested `.gz` files within) safely and predictably on macOS and Windows.

Key Features
------------
* Single static binary (Go) – easy to distribute (.exe / .app)
* Detects whether a gzip holds a tar archive vs a single file
* Recursively extracts nested `.gz` files (`--no-recursive` to disable)
* Path traversal protection & name conflict avoidance
* Optional removal of compressed nested files after extraction
* Dry‑run mode for inspection

Safety Model
------------
* Any archive entry with path traversal (`..`) or absolute path becomes `_`.
* Existing file collisions are resolved with `-1`, `-2`, ... suffixes.
* Symlinks are skipped unless `--allow-symlinks` is provided.

Install / Build
---------------
Prerequisite: Go 1.23 (devcontainer already configured).

Build locally:

```
make build
```

Cross-compile common targets:

```
make build-all
```

macOS .app bundle (after build on macOS):

```
make macos-app
```

Version injection (example):

```
make build VERSION=v0.2.0
```

CLI Usage
---------
```
bar-extract [flags] <file.bar> [more.bar ...]

Flags:
	-allow-symlinks       allow creation of symlinks from tar archives (default false)
	-dry-run              simulate actions without writing
	-no-recursive         disable nested gzip extraction
	-remove-compressed    remove .gz files after successful nested extraction
	-verbose              verbose logging to stderr
	-version              print version and exit
```

Behavior
--------
* If the gzip stream is a tar, every regular file & directory is recreated beneath the source directory.
* If it is a single compressed file, it is written next to the source using the gzip header name if available, else the original basename minus `.bar`/`.gz`.
* Recursion: after initial extraction each discovered `*.gz` file is extracted in place (unless `-no-recursive`).

Examples
--------
```
# Basic extraction
bar-extract sample.bar

# Dry-run to preview
bar-extract -dry-run large.bar

# Disable recursion
bar-extract -no-recursive dataset.bar

# Remove nested compressed files after expansion
bar-extract -remove-compressed archive.bar
```

File Association
----------------
### macOS
1. Build and create the app bundle: `make macos-app`
2. Move `dist/bar-extract.app` to `/Applications` or another location.
3. In Finder, right-click a `.bar` file > Get Info.
4. Under "Open with" choose `bar-extract` and click "Change All…".

To reset: repeat and choose a different application.

### Windows
1. Build: `make build-all` (grab the `bar-extract-windows-amd64.exe`).
2. Rename to `bar-extract.exe` and place it in a folder e.g. `C:\Tools\bar-extract`.
3. Right-click a `.bar` file > Open With > Choose another app > More apps > Look for another app on this PC > select the exe. Check "Always use this app".

Registry (advanced / optional) sample (`associate-bar.reg`):
```
Windows Registry Editor Version 5.00

[HKEY_CLASSES_ROOT\.bar]
@="barfile"

[HKEY_CLASSES_ROOT\barfile]
@="BAR Archive"

[HKEY_CLASSES_ROOT\barfile\shell\open\command]
@="\"C:\\Tools\\bar-extract\\bar-extract.exe\" \"%1\""
```
Double-click registry file to apply (requires admin in some contexts).

Exit Codes
----------
* 0 success (even if some nested skips occurred)
* 1 usage error (no args)
* 2 extraction error encountered

Development
-----------
Devcontainer is included; opening in VS Code will provision Go 1.23. Run tests:
```
make test
```

Project Structure
-----------------
```
cmd/bar-extract/        CLI entrypoint
internal/extractor/     Core extraction logic & helpers
scripts/                macOS bundling script
.devcontainer/          Development container config
```

Limitations / Notes
-------------------
* Only gzip (.bar assumed to be gzip). Other compression formats not supported.
* Tar detection is heuristic (looks for ustar magic). False positives are unlikely but possible for specially crafted files.
* Symlinks skipped unless explicitly allowed.

Roadmap / Ideas
---------------
* Optional progress reporting
* Parallel extraction for large batches
* GUI wrapper

License
-------
See `LICENSE`.
