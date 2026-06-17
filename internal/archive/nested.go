package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// NestedArchiveHandler manages recursive archive scanning.
type NestedArchiveHandler struct {
	MaxDepth int
	tempDir  string
	cleanup  []string
}

// NewNestedArchiveHandler creates a handler for nested archive scanning.
func NewNestedArchiveHandler(maxDepth int) (*NestedArchiveHandler, error) {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	dir, err := os.MkdirTemp("", "getroute-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	return &NestedArchiveHandler{
		MaxDepth: maxDepth,
		tempDir:  dir,
	}, nil
}

// Cleanup removes all temp files.
func (h *NestedArchiveHandler) Cleanup() {
	if h.tempDir != "" {
		os.RemoveAll(h.tempDir)
		h.tempDir = ""
	}
}

// ReadInnerJar reads an inner jar entry and returns an ArchiveReader for it.
// If the inner jar is large (> 100MB), it's extracted to a temp file first.
func (h *NestedArchiveHandler) ReadInnerJar(f *zip.File, parentName string) (*ArchiveReader, error) {
	// For large files, extract to temp file.
	if f.UncompressedSize64 > largeJarThreshold {
		return h.readLargeInnerJar(f, parentName)
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open inner jar %s: %w", f.Name, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, maxEntrySize))
	if err != nil {
		return nil, fmt.Errorf("read inner jar %s: %w", f.Name, err)
	}

	return OpenArchiveFromBytes(data, filepath.Base(f.Name))
}

func (h *NestedArchiveHandler) readLargeInnerJar(f *zip.File, parentName string) (*ArchiveReader, error) {
	tmpFile, err := os.CreateTemp(h.tempDir, "inner-*.jar")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	rc, err := f.Open()
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("open inner jar %s: %w", f.Name, err)
	}
	defer rc.Close()

	_, err = io.Copy(tmpFile, rc)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("extract inner jar %s: %w", f.Name, err)
	}

	return OpenArchive(tmpFile.Name())
}

// IsInnerJarPath checks if a path is inside a known lib directory.
func IsInnerJarPath(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "web-inf/lib/") ||
		strings.Contains(lower, "boot-inf/lib/") ||
		strings.Contains(lower, "lib/") ||
		strings.HasPrefix(lower, "lib/")
}

// IsInnerWarPath checks if a path looks like a WAR inside another archive.
func IsInnerWarPath(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".war") && IsInnerJarPath(name)
}

// ExtractManifest extracts and parses META-INF/MANIFEST.MF from an archive reader.
func ExtractManifest(ar *ArchiveReader) (*Manifest, error) {
	for _, f := range ar.Entries() {
		if IsManifest(f.Name) {
			data, err := ar.ReadEntry(f)
			if err != nil {
				return nil, err
			}
			return ParseManifest(data)
		}
	}
	return nil, fmt.Errorf("no MANIFEST.MF found in %s", ar.Name())
}

// WalkFunc is called for each class file, XML file, and jar name found during scanning.
type WalkFunc func(entry ArchiveEntry, data []byte, err error) error

// WalkArchive recursively walks an archive and its nested jars, calling fn for each entry.
func (h *NestedArchiveHandler) WalkArchive(ar *ArchiveReader, depth int, fn WalkFunc) error {
	if depth > h.MaxDepth {
		return nil
	}

	for _, f := range ar.Entries() {
		name := f.Name

		// Skip directories.
		if f.FileInfo().IsDir() {
			continue
		}

		entry := ArchiveEntry{
			Name:    name,
			IsDir:   false,
			Size:    int64(f.UncompressedSize64),
			Archive: ar.Name(),
		}

		// Handle inner jars/wars recursively.
		if IsInnerJarPath(name) || IsInnerWarPath(name) {
			innerAr, err := h.ReadInnerJar(f, ar.Name())
			if err != nil {
				// Skip corrupted inner jars.
				continue
			}
			h.WalkArchive(innerAr, depth+1, fn)
			innerAr.Close()
			continue
		}

		// Skip non-interesting files.
		if !IsClassFile(name) && !IsXMLFile(name) && !IsManifest(name) {
			continue
		}

		data, err := ar.ReadEntry(f)
		if err != nil {
			if e := fn(entry, nil, err); e != nil {
				return e
			}
			continue
		}

		if err := fn(entry, data, nil); err != nil {
			return err
		}
	}

	return nil
}

const largeJarThreshold = 100 * 1024 * 1024 // 100MB
