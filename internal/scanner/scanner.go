package scanner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"GetRoute/internal/archive"
	"GetRoute/internal/classfile"
)

type classTask struct {
	name string
	data []byte
}

// Scanner orchestrates the archive scanning process.
type Scanner struct {
	MaxDepth   int
	MaxWorkers int
	Verbose    bool
	handler    *archive.NestedArchiveHandler
}

// NewScanner creates a new Scanner.
func NewScanner(maxDepth, maxWorkers int, verbose bool) (*Scanner, error) {
	handler, err := archive.NewNestedArchiveHandler(maxDepth)
	if err != nil {
		return nil, err
	}
	return &Scanner{
		MaxDepth:   maxDepth,
		MaxWorkers: maxWorkers,
		Verbose:    verbose,
		handler:    handler,
	}, nil
}

// Cleanup removes temporary files created during scanning.
func (s *Scanner) Cleanup() {
	s.handler.Cleanup()
}

// ScanAll scans all input archives and returns aggregated results.
func (s *Scanner) ScanAll(inputs []string) ([]*ScanResult, error) {
	var results []*ScanResult

	for _, input := range inputs {
		ar, err := archive.OpenArchive(input)
		if err != nil {
			return nil, err
		}

		result := &ScanResult{
			Classes:     make(map[string]*classfile.ClassFile),
			XMLFiles:    make(map[string][]byte),
			ArchiveName: ar.Name(),
		}

		if s.Verbose {
			log.Printf("[SCAN] Processing archive: %s", ar.Name())
		}

		// Scan the archive to collect class files, XML, manifest, jar names.
		s.scanArchive(ar, 0, result)
		ar.Close()

		results = append(results, result)

		if s.Verbose {
			log.Printf("[SCAN] %s: %d classes, %d XML files, %d jars",
				ar.Name(), len(result.Classes), len(result.XMLFiles), len(result.JarNames))
		}
	}

	return results, nil
}

func (s *Scanner) scanArchive(ar *archive.ArchiveReader, depth int, result *ScanResult) {
	if depth > s.MaxDepth {
		return
	}

	entries := ar.Entries()
	var classTasks []classTask

	for _, f := range entries {
		name := f.Name

		if f.FileInfo().IsDir() {
			continue
		}

		// Handle inner jars recursively.
		if ar.IsInnerJar(name) || archive.IsInnerWarPath(name) {
			innerAr, err := s.handler.ReadInnerJar(f, ar.Name())
			if err != nil {
				if s.Verbose {
					log.Printf("[WARN] Skip inner jar %s: %v", name, err)
				}
				continue
			}
			result.JarNames = append(result.JarNames, filepath.Base(name))
			s.scanArchive(innerAr, depth+1, result)
			innerAr.Close()
			continue
		}

		// Track jar names for component detection.
		if strings.HasSuffix(strings.ToLower(name), ".jar") {
			result.JarNames = append(result.JarNames, filepath.Base(name))
		}

		// Handle manifest.
		if archive.IsManifest(name) {
			data, err := ar.ReadEntry(f)
			if err != nil {
				continue
			}
			mf, err := archive.ParseManifest(data)
			if err == nil {
				result.Manifest = mf
			}
			continue
		}

		// Handle XML files.
		if archive.IsXMLFile(name) {
			data, err := ar.ReadEntryLimited(f, 10*1024*1024) // 10MB max for XML
			if err != nil {
				if s.Verbose {
					log.Printf("[WARN] Failed to read XML %s: %v", name, err)
				}
				continue
			}
			result.XMLFiles[name] = data
			continue
		}

		// Handle class files.
		if archive.IsClassFile(name) {
			data, err := ar.ReadEntry(f)
			if err != nil {
				if s.Verbose {
					log.Printf("[WARN] Failed to read class %s: %v", name, err)
				}
				continue
			}
			classTasks = append(classTasks, classTask{name: name, data: data})
		}
	}

	// Parse class files using worker pool.
	if len(classTasks) > 0 {
		s.parseClasses(classTasks, result, ar.Name())
	}
}

// ScanDir scans a directory tree and returns a ScanResult with all discovered artifacts.
// It walks the directory for .class files, XML configs, MANIFEST.MF, and nested .jar/.war files.
func (s *Scanner) ScanDir(dirPath string) (*ScanResult, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("resolve directory path: %w", err)
	}

	if s.Verbose {
		log.Printf("[SCAN] Processing directory: %s", absPath)
	}

	result := &ScanResult{
		Classes:     make(map[string]*classfile.ClassFile),
		XMLFiles:    make(map[string][]byte),
		ArchiveName: filepath.Base(absPath),
	}

	var classTasks []classTask

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(absPath, path)
		relPath = filepath.ToSlash(relPath)

		lowerName := strings.ToLower(info.Name())

		// Handle nested jar/war files recursively.
		if strings.HasSuffix(lowerName, ".jar") || strings.HasSuffix(lowerName, ".war") {
			result.JarNames = append(result.JarNames, info.Name())
			innerAr, err := archive.OpenArchive(path)
			if err != nil {
				if s.Verbose {
					log.Printf("[WARN] Skip archive %s: %v", relPath, err)
				}
				return nil
			}
			s.scanArchive(innerAr, 1, result)
			innerAr.Close()
			return nil
		}

		// Handle manifest.
		if archive.IsManifest(relPath) {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			mf, err := archive.ParseManifest(data)
			if err == nil {
				result.Manifest = mf
			}
			return nil
		}

		// Handle XML files.
		if archive.IsXMLFile(relPath) {
			data, err := os.ReadFile(path)
			if err != nil {
				if s.Verbose {
					log.Printf("[WARN] Failed to read XML %s: %v", relPath, err)
				}
				return nil
			}
			result.XMLFiles[relPath] = data
			return nil
		}

		// Handle class files.
		if archive.IsClassFile(relPath) {
			data, err := os.ReadFile(path)
			if err != nil {
				if s.Verbose {
					log.Printf("[WARN] Failed to read class %s: %v", relPath, err)
				}
				return nil
			}
			classTasks = append(classTasks, classTask{name: relPath, data: data})
			return nil
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", absPath, err)
	}

	// Parse class files using worker pool.
	if len(classTasks) > 0 {
		s.parseClasses(classTasks, result, filepath.Base(absPath))
	}

	if s.Verbose {
		log.Printf("[SCAN] %s: %d classes, %d XML files, %d jars",
			absPath, len(result.Classes), len(result.XMLFiles), len(result.JarNames))
	}

	return result, nil
}

func (s *Scanner) parseClasses(tasks []classTask, result *ScanResult, archiveName string) {
	if s.MaxWorkers <= 0 {
		s.MaxWorkers = 1
	}

	taskCh := make(chan classTask, len(tasks))
	resultCh := make(chan *classfile.ClassFile, len(tasks))

	var wg sync.WaitGroup
	for i := 0; i < s.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskCh {
				cf, err := classfile.ReadClassFile(t.data, archiveName, t.name)
				if err != nil {
					if s.Verbose {
						log.Printf("[WARN] Parse class %s: %v", t.name, err)
					}
					continue
				}
				resultCh <- cf
			}
		}()
	}

	// Feed tasks.
	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	wg.Wait()
	close(resultCh)

	for cf := range resultCh {
		result.Classes[cf.FilePath] = cf
	}
}
