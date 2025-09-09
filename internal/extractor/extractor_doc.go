package extractor

// Package extractor provides functionality to expand .bar (gzip-based) archives.
//
// Overview
//   A .bar file is treated as a gzip stream. If it contains a tar archive (detected
//   heuristically by checking for a ustar/posix header) we expand it preserving
//   directory structure. If it is a single compressed file we write that file out
//   next to the source. After extraction (and if recursive mode is enabled) any
//   nested .gz files within the extracted tree are also decompressed in-place.
//
// Safety
//   Filenames are sanitized to avoid path traversal or absolute path writes.
//   Conflicting filenames get an auto-incremented suffix.
//
// Configuration
//   Behavior is controlled via the Config struct. See that type's documentation
//   for details of each option.
