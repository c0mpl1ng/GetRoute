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
	wd := getwd()

	for _, input := range inputs {
		ar, err := archive.OpenArchive(input)
		if err != nil {
			return nil, err
		}

		relPath := relPathFromCWD(wd, input)

		result := &ScanResult{
			Classes:     make(map[string]*classfile.ClassFile),
			XMLFiles:    make(map[string][]byte),
			ArchiveName: relPath,
		}

		if s.Verbose {
			log.Printf("[SCAN] Processing archive: %s", relPath)
		}

		// Scan the archive to collect class files, XML, manifest, jar names.
		s.scanArchive(ar, 0, result, relPath)
		ar.Close()

		results = append(results, result)

		if s.Verbose {
			log.Printf("[SCAN] %s: %d classes, %d XML files, %d jars",
				relPath, len(result.Classes), len(result.XMLFiles), len(result.JarNames))
		}
	}

	return results, nil
}

func (s *Scanner) scanArchive(ar *archive.ArchiveReader, depth int, result *ScanResult, sourcePrefix string) {
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
			jarBase := normalizeJarName(filepath.Base(name))
			result.JarNames = append(result.JarNames, sourcePrefix+"!"+jarBase)
			s.scanArchive(innerAr, depth+1, result, sourcePrefix+"!"+jarBase)
			innerAr.Close()
			continue
		}

		// Track jar names for component detection.
		if strings.HasSuffix(strings.ToLower(name), ".jar") {
			jarBase := normalizeJarName(filepath.Base(name))
			result.JarNames = append(result.JarNames, sourcePrefix+"!"+jarBase)
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

		// Handle Java source files.
		if archive.IsJavaFile(name) {
			data, err := ar.ReadEntry(f)
			if err != nil {
				if s.Verbose {
					log.Printf("[WARN] Failed to read java source %s: %v", name, err)
				}
				continue
			}
			classTasks = append(classTasks, classTask{name: name, data: data})
		}
	}

	// Parse class files using worker pool.
	if len(classTasks) > 0 {
		s.parseClasses(classTasks, result, sourcePrefix)
	}
}

// ScanDir scans a directory tree and returns a ScanResult with all discovered artifacts.
// It walks the directory for .class files, XML configs, MANIFEST.MF, and nested .jar/.war files.
func (s *Scanner) ScanDir(dirPath string) (*ScanResult, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("resolve directory path: %w", err)
	}

	wd := getwd()
	relDirPath := relPathFromCWD(wd, absPath)

	if s.Verbose {
		log.Printf("[SCAN] Processing directory: %s", absPath)
	}

	result := &ScanResult{
		Classes:     make(map[string]*classfile.ClassFile),
		XMLFiles:    make(map[string][]byte),
		ArchiveName: relDirPath,
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

		// Handle nested jar/war/zip files recursively.
		if strings.HasSuffix(lowerName, ".jar") || strings.HasSuffix(lowerName, ".war") || strings.HasSuffix(lowerName, ".zip") {
			jarRelPath := relPathFromCWD(wd, path)
			jarRelPath = normalizeJarPath(jarRelPath)
			result.JarNames = append(result.JarNames, jarRelPath)
			innerAr, err := archive.OpenArchive(path)
			if err != nil {
				if s.Verbose {
					log.Printf("[WARN] Skip archive %s: %v", relPath, err)
				}
				return nil
			}
			s.scanArchive(innerAr, 1, result, jarRelPath)
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

		// Handle Java source files.
		if archive.IsJavaFile(relPath) {
			data, err := os.ReadFile(path)
			if err != nil {
				if s.Verbose {
					log.Printf("[WARN] Failed to read java source %s: %v", relPath, err)
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
		s.parseClasses(classTasks, result, relDirPath)
	}

	if s.Verbose {
		log.Printf("[SCAN] %s: %d classes, %d XML files, %d jars",
			relDirPath, len(result.Classes), len(result.XMLFiles), len(result.JarNames))
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
				var cf *classfile.ClassFile
				var err error
				if strings.HasSuffix(strings.ToLower(t.name), ".java") {
					cf, err = classfile.ReadJavaFile(t.data, archiveName, t.name)
				} else {
					cf, err = classfile.ReadClassFile(t.data, archiveName, t.name)
				}
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

// normalizeJarName strips known source/source-zip suffixes so that
// component detection matches against standard jar names.
// "spring-webmvc-5.1.18.RELEASE.jar.src.zip" → "spring-webmvc-5.1.18.RELEASE.jar"
// "spring-webmvc-5.1.18.RELEASE.jar.src"     → "spring-webmvc-5.1.18.RELEASE.jar"
func normalizeJarName(name string) string {
	lower := strings.ToLower(name)
	// Strip .src.zip suffix first.
	if strings.HasSuffix(lower, ".jar.src.zip") {
		return name[:len(name)-len(".src.zip")]
	}
	// Then strip .src suffix.
	if strings.HasSuffix(lower, ".jar.src") {
		return name[:len(name)-len(".src")]
	}
	return name
}

// normalizeJarPath applies normalizeJarName to the base name of a full path,
// preserving any archive prefix (!-separated).
func normalizeJarPath(path string) string {
	// Handle archive prefix: "target/app.war!spring-webmvc.jar.src.zip"
	if idx := strings.LastIndex(path, "!"); idx >= 0 {
		prefix := path[:idx+1]
		base := normalizeJarName(path[idx+1:])
		return prefix + base
	}
	// Direct filesystem path.
	dir := filepath.Dir(path)
	base := normalizeJarName(filepath.Base(path))
	if dir == "." {
		return base
	}
	return dir + "/" + base
}

func getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func relPathFromCWD(wd, absPath string) string {
	rel, err := filepath.Rel(wd, absPath)
	if err != nil {
		return filepath.Base(absPath)
	}
	return filepath.ToSlash(rel)
}
