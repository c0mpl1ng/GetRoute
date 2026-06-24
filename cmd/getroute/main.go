package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"strings"

	"GetRoute/internal/classfile"
	"GetRoute/internal/detector"
	"GetRoute/internal/exporter"
	"GetRoute/internal/extractor"
	"GetRoute/internal/indexer"
	"GetRoute/internal/model"
	"GetRoute/internal/scanner"
)

func main() {
	var (
		input   string
		output  string
		threads int
		verbose bool
	)

	flag.StringVar(&input, "input", "", "Input file (jar/war/zip) or directory path")
	flag.StringVar(&output, "output", ".", "Output directory")
	flag.IntVar(&threads, "threads", runtime.NumCPU(), "Number of concurrent workers")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	// Support -input flag or positional arg.
	inputs := flag.Args()
	if input != "" {
		inputs = append([]string{input}, inputs...)
	}

	if len(inputs) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: GetRoute [flags] <file.jar|file.war|file.zip|directory>...\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  GetRoute -input app.jar\n")
		fmt.Fprintf(os.Stderr, "  GetRoute -input app.war -output ./result\n")
		fmt.Fprintf(os.Stderr, "  GetRoute -input ./target/classes -verbose\n")
		fmt.Fprintf(os.Stderr, "  GetRoute -input app.jar -threads 20\n")
		os.Exit(1)
	}

	// Separate inputs into files and directories.
	var fileInputs []string
	var dirInputs []string
	for _, f := range inputs {
		info, err := os.Stat(f)
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: input not found: %s\n", f)
			os.Exit(1)
		}
		if info.IsDir() {
			dirInputs = append(dirInputs, f)
		} else {
			fileInputs = append(fileInputs, f)
		}
	}

	// Create output directory if needed.
	if output != "." {
		if err := os.MkdirAll(output, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot create output directory: %v\n", err)
			os.Exit(1)
		}
	}

	if verbose {
		log.SetFlags(log.LstdFlags | log.Lmsgprefix)
		log.SetPrefix("[GETROUTE] ")
		log.Printf("Starting analysis with %d threads", threads)
		log.Printf("Input files: %v", fileInputs)
		log.Printf("Input directories: %v", dirInputs)
	}

	// Phase 1: Scan inputs.
	scan, err := scanner.NewScanner(2, threads, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer scan.Cleanup()

	// Phase 1a: Scan archives.
	results, err := scan.ScanAll(fileInputs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning archives: %v\n", err)
		os.Exit(1)
	}

	// Phase 1b: Scan directories.
	for _, d := range dirInputs {
		dirResult, err := scan.ScanDir(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning directory %s: %v\n", d, err)
			os.Exit(1)
		}
		results = append(results, dirResult)
	}

	if verbose {
		totalClasses := 0
		for _, r := range results {
			totalClasses += len(r.Classes)
		}
		log.Printf("Scan complete: %d inputs, %d classes", len(results), totalClasses)
	}

	// Phase 2: Build extraction context.
	idx := indexer.NewIndexer()
	reg := extractor.NewRegistry()

	for _, result := range results {
		// Merge class files from all scans.
		allClasses := make(map[string]*classfile.ClassFile)
		for k, v := range result.Classes {
			allClasses[k] = v
		}

		// Merge XML files.
		allXML := make(map[string][]byte)
		for k, v := range result.XMLFiles {
			allXML[k] = v
		}

		// Merge jar names.
		allJars := append([]string{}, result.JarNames...)

		// Build manifest map.
		manifestMap := make(map[string]string)
		if result.Manifest != nil {
			for k, v := range result.Manifest.Main {
				manifestMap[k] = v
			}
		}

		// Determine if BOOT-INF is present.
		bootInf := result.Manifest != nil && result.Manifest.HasBootInf()

		ctx := &extractor.Context{
			Classes:    allClasses,
			XMLFiles:   allXML,
			Manifest:   manifestMap,
			BootInf:    bootInf,
			JarNames:   allJars,
			ParentName: result.ArchiveName,
		}

		// Phase 3: Extract routes and classes.
		routes, classes, err := reg.ExtractAll(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting routes: %v\n", err)
			os.Exit(1)
		}
		idx.AddRoutes(routes)
		idx.AddClasses(classes)

		// Phase 3.5: Add ALL scanned classes to the index (not just extractor matches).
		for _, cf := range allClasses {
			className := strings.ReplaceAll(cf.ThisClass, "/", ".")
			idx.AddClasses([]model.ClassInfo{{
				FullName:    className,
				Package:     pkgName(className),
				FilePath:    cf.FilePath,
				SuperClass:  strings.ReplaceAll(cf.SuperClass, "/", "."),
				ArchiveName: cf.ArchiveName,
			}})
		}

		// Phase 4: Detect frameworks.
		frameworks := detector.DetectFrameworks(
			allClasses, allXML, manifestMap, bootInf, allJars,
		)
		idx.AddFrameworks(frameworks)

		// Phase 5: Detect components.
		components := detector.DetectComponents(allJars)
		idx.AddComponents(components)

		if verbose {
			log.Printf("[%s] %d routes, %d classes, %d frameworks, %d components",
				result.ArchiveName, len(routes), len(classes), len(frameworks), len(components))
		}
	}

	// Phase 6: Build index (dedup, sort, normalize).
	idx.Build()

	// Check if any supported framework was detected.
	frameworkNames := detector.GetSupportedFrameworks(idx.Frameworks())
	if frameworkNames == "Unknown" {
		fmt.Fprintf(os.Stderr, "Unsupported framework: no supported web framework detected in the input.\n")
		os.Exit(1)
	}

	// Phase 7: Export to Excel.
	outputFile := output + "/GetRoute.xlsx"
	if err := exporter.Export(idx.Routes(), idx.Classes(), idx.Controllers(), idx.Frameworks(), idx.Components(), outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error exporting to Excel: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Analysis complete!\n")
	fmt.Printf("  Framework: %s\n", frameworkNames)
	fmt.Printf("  Routes: %d\n", len(idx.Routes()))
	fmt.Printf("  Controllers: %d\n", len(idx.Controllers()))
	fmt.Printf("  Classes: %d\n", len(idx.Classes()))
	fmt.Printf("  Components: %d\n", len(idx.Components()))
	fmt.Printf("  Output: %s\n", outputFile)
}

func pkgName(className string) string {
	lastDot := strings.LastIndex(className, ".")
	if lastDot < 0 {
		return ""
	}
	return className[:lastDot]
}
