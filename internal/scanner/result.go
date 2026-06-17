package scanner

import (
	"GetRoute/internal/archive"
	"GetRoute/internal/classfile"
)

// ScanResult holds all parsed data from scanning archives.
type ScanResult struct {
	Classes     map[string]*classfile.ClassFile // Key: slash-separated class path
	XMLFiles    map[string][]byte               // Key: filename within archive
	Manifest    *archive.Manifest
	JarNames    []string // All .jar filenames found
	ArchiveName string
}
