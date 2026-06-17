package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ArchiveEntry represents a single entry in an archive.
type ArchiveEntry struct {
	Name    string // Relative path within the archive
	IsDir   bool
	Size    int64
	Archive string // Name of the archive containing this entry
}

// ArchiveReader provides read access to a ZIP/JAR/WAR file.
type ArchiveReader struct {
	path    string
	name    string
	zipRC   *zip.ReadCloser
	zipR    *zip.Reader // For in-memory archives
	data    []byte      // Raw data for in-memory archives
	tempDir string      // Temp directory for large inner archives
}

// OpenArchive opens a ZIP/JAR/WAR file from disk.
func OpenArchive(path string) (*ArchiveReader, error) {
	rc, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open archive %s: %w", path, err)
	}
	return &ArchiveReader{
		path:  path,
		name:  filepath.Base(path),
		zipRC: rc,
	}, nil
}

// OpenArchiveFromBytes opens an archive from in-memory bytes (for nested jars).
func OpenArchiveFromBytes(data []byte, name string) (*ArchiveReader, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open archive %s: %w", name, err)
	}
	return &ArchiveReader{
		name: name,
		zipR: r,
		data: data,
	}, nil
}

// Name returns the archive filename.
func (ar *ArchiveReader) Name() string {
	return ar.name
}

// Path returns the full path to the archive on disk.
func (ar *ArchiveReader) Path() string {
	return ar.path
}

// Entries returns all entries in the archive.
func (ar *ArchiveReader) Entries() []*zip.File {
	if ar.zipRC != nil {
		return ar.zipRC.File
	}
	if ar.zipR != nil {
		return ar.zipR.File
	}
	return nil
}

// ReadEntry reads the full content of an entry.
func (ar *ArchiveReader) ReadEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("read entry %s: %w", f.Name, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, maxEntrySize))
	if err != nil {
		return nil, fmt.Errorf("read entry %s: %w", f.Name, err)
	}
	return data, nil
}

// ReadEntryLimited reads up to maxSize bytes of an entry.
func (ar *ArchiveReader) ReadEntryLimited(f *zip.File, maxSize int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("read entry %s: %w", f.Name, err)
	}
	defer rc.Close()

	if maxSize <= 0 {
		maxSize = maxEntrySize
	}
	data, err := io.ReadAll(io.LimitReader(rc, maxSize))
	if err != nil {
		return nil, fmt.Errorf("read entry %s: %w", f.Name, err)
	}
	return data, nil
}

// IsInnerJar returns true if the entry is a jar file inside lib directories.
func (ar *ArchiveReader) IsInnerJar(name string) bool {
	if !strings.HasSuffix(strings.ToLower(name), ".jar") {
		return false
	}
	// Check if it's in a lib directory.
	lower := strings.ToLower(name)
	return strings.Contains(lower, "web-inf/lib/") ||
		strings.Contains(lower, "boot-inf/lib/") ||
		strings.Contains(lower, "lib/")
}

// IsClassFile returns true if the entry is a .class file.
func IsClassFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".class")
}

// IsXMLFile returns true if the entry is a relevant XML config file.
func IsXMLFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".xml") &&
		!strings.Contains(lower, "pom.xml")
}

// IsManifest returns true if the entry is META-INF/MANIFEST.MF.
func IsManifest(name string) bool {
	upper := strings.ToUpper(name)
	return upper == "META-INF/MANIFEST.MF"
}

// IsWebXML returns true if the entry is WEB-INF/web.xml.
func IsWebXML(name string) bool {
	upper := strings.ToUpper(name)
	return upper == "WEB-INF/WEB.XML"
}

// IsStrutsXML returns true if the entry is struts.xml.
func IsStrutsXML(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), "struts.xml")
}

// IsXWorkXML returns true if the entry is xwork.xml.
func IsXWorkXML(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), "xwork.xml")
}

// IsSpringXML returns true if the entry is a Spring configuration file.
func IsSpringXML(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, "spring.factories") ||
		strings.Contains(lower, "application") && strings.HasSuffix(lower, ".xml")
}

// HasBootInf returns true if the archive contains BOOT-INF/ entries (Spring Boot).
func (ar *ArchiveReader) HasBootInf() bool {
	for _, f := range ar.Entries() {
		if strings.HasPrefix(strings.ToUpper(f.Name), "BOOT-INF/") {
			return true
		}
	}
	return false
}

// Close closes the archive and cleans up temp files.
func (ar *ArchiveReader) Close() error {
	if ar.zipRC != nil {
		ar.zipRC.Close()
	}
	if ar.tempDir != "" {
		os.RemoveAll(ar.tempDir)
	}
	return nil
}

// Max entry size to read into memory (500MB).
const maxEntrySize = 500 * 1024 * 1024
