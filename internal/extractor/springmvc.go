package extractor

import (
	"strings"

	"GetRoute/internal/classfile"
	"GetRoute/internal/model"
)

// Spring annotation descriptors.
const (
	springRequestMappingDesc = "Lorg/springframework/web/bind/annotation/RequestMapping;"
	springGetMappingDesc     = "Lorg/springframework/web/bind/annotation/GetMapping;"
	springPostMappingDesc    = "Lorg/springframework/web/bind/annotation/PostMapping;"
	springPutMappingDesc     = "Lorg/springframework/web/bind/annotation/PutMapping;"
	springDeleteMappingDesc  = "Lorg/springframework/web/bind/annotation/DeleteMapping;"
	springPatchMappingDesc   = "Lorg/springframework/web/bind/annotation/PatchMapping;"
	springControllerDesc     = "Lorg/springframework/stereotype/Controller;"
	springRestControllerDesc = "Lorg/springframework/web/bind/annotation/RestController;"
	springServiceDesc        = "Lorg/springframework/stereotype/Service;"
	springRepositoryDesc     = "Lorg/springframework/stereotype/Repository;"
	springComponentDesc      = "Lorg/springframework/stereotype/Component;"
	springResponseBodyDesc   = "Lorg/springframework/web/bind/annotation/ResponseBody;"
)

// SpringMVCExtractor extracts routes from Spring MVC annotated controllers.
type SpringMVCExtractor struct{}

func (e *SpringMVCExtractor) Name() string     { return "Spring MVC" }
func (e *SpringMVCExtractor) Priority() int    { return 90 }

func (e *SpringMVCExtractor) CanHandle(ctx *Context) bool {
	// Check if any class has @Controller or @RestController.
	for _, cf := range ctx.Classes {
		for _, ann := range cf.ClassAnnotations() {
			desc := ann.Type
			if desc == springControllerDesc || desc == springRestControllerDesc {
				return true
			}
		}
	}
	return false
}

func (e *SpringMVCExtractor) Extract(ctx *Context) ([]model.RouteInfo, []model.ClassInfo, error) {
	var routes []model.RouteInfo
	var classes []model.ClassInfo

	for _, cf := range ctx.Classes {
		classAnnotations := cf.ClassAnnotations()
		springType := e.springType(classAnnotations)

		if springType == "" {
			continue
		}

		// Build class-level base paths.
		basePaths := e.extractClassPaths(classAnnotations)

		// Use the ClassFile's archive name (tracks innermost jar for nested archives).
		archiveName := cf.ArchiveName
		if archiveName == "" {
			archiveName = ctx.ParentName
		}

		// Build ClassInfo.
		className := strings.ReplaceAll(cf.ThisClass, "/", ".")
		ci := model.ClassInfo{
			FullName:    className,
			Package:     e.extractPackage(className),
			FilePath:    cf.FilePath,
			SuperClass:  strings.ReplaceAll(cf.SuperClass, "/", "."),
			Annotations: e.annotationSimpleNames(classAnnotations),
			SpringType:  springType,
			ArchiveName: archiveName,
		}
		classes = append(classes, ci)

		// Extract routes from methods.
		for _, method := range cf.Methods {
			methodAnnotations := method.MethodAnnotations(cf.ConstantPool)

			// Skip methods without any HTTP mapping annotation.
			if !e.hasMappingAnnotation(methodAnnotations) {
				continue
			}

			httpMethods, methodPaths := e.extractMethodPaths(methodAnnotations)

			for _, basePath := range basePaths {
				for _, methodPath := range methodPaths {
					url := joinPath(basePath, methodPath)

					// If no HTTP method specified, default to GET (Spring MVC default for @RequestMapping).
					methods := httpMethods
					if len(methods) == 0 {
						methods = nil // nil means "all methods"
					}

					routes = append(routes, model.RouteInfo{
						URL:         url,
						HTTPMethods: methods,
						ClassName:   className,
						MethodName:  method.Name,
						Framework:   "Spring MVC",
						SourceType:  "ANNOTATION",
						SourceFile:  cf.FilePath,
						ArchiveName: archiveName,
					})
				}
			}
		}
	}

	return routes, classes, nil
}

func (e *SpringMVCExtractor) springType(annotations []classfile.ParsedAnnotation) string {
	for _, ann := range annotations {
		switch ann.Type {
		case springControllerDesc:
			return "Controller"
		case springRestControllerDesc:
			return "RestController"
		}
	}
	return ""
}

func (e *SpringMVCExtractor) extractClassPaths(annotations []classfile.ParsedAnnotation) []string {
	var paths []string
	for _, ann := range annotations {
		if ann.Type != springRequestMappingDesc {
			continue
		}
		// Check "value" and "path" elements.
		for _, elemName := range []string{"value", "path"} {
			if elem := ann.GetElement(elemName); elem != nil {
				for _, p := range elem.AsStringArray() {
					p = strings.TrimSpace(p)
					if p != "" && p != "/" {
						paths = append(paths, p)
					} else {
						paths = append(paths, "")
					}
				}
			}
		}
	}
	if len(paths) == 0 {
		paths = []string{""}
	}
	return paths
}

func (e *SpringMVCExtractor) extractMethodPaths(annotations []classfile.ParsedAnnotation) ([]string, []string) {
	var httpMethods []string
	var paths []string

	for _, ann := range annotations {
		switch ann.Type {
		case springRequestMappingDesc:
			// Check "method" element for HTTP methods.
			if elem := ann.GetElement("method"); elem != nil {
				httpMethods = append(httpMethods, elem.AsEnumArray()...)
			}
			// Check "value" and "path" for paths.
			for _, elemName := range []string{"value", "path"} {
				if elem := ann.GetElement(elemName); elem != nil {
					for _, p := range elem.AsStringArray() {
						p = strings.TrimSpace(p)
						if p != "" {
							paths = append(paths, p)
						}
					}
				}
			}
		case springGetMappingDesc:
			httpMethods = []string{"GET"}
			paths = append(paths, e.extractPathValues(ann)...)
		case springPostMappingDesc:
			httpMethods = []string{"POST"}
			paths = append(paths, e.extractPathValues(ann)...)
		case springPutMappingDesc:
			httpMethods = []string{"PUT"}
			paths = append(paths, e.extractPathValues(ann)...)
		case springDeleteMappingDesc:
			httpMethods = []string{"DELETE"}
			paths = append(paths, e.extractPathValues(ann)...)
		case springPatchMappingDesc:
			httpMethods = []string{"PATCH"}
			paths = append(paths, e.extractPathValues(ann)...)
		}
	}

	if len(paths) == 0 {
		paths = []string{""}
	}

	return httpMethods, paths
}

func (e *SpringMVCExtractor) extractPathValues(ann classfile.ParsedAnnotation) []string {
	var paths []string
	for _, elemName := range []string{"value", "path"} {
		if elem := ann.GetElement(elemName); elem != nil {
			for _, p := range elem.AsStringArray() {
				p = strings.TrimSpace(p)
				if p != "" {
					paths = append(paths, p)
				}
			}
		}
	}
	if len(paths) == 0 {
		paths = []string{""}
	}
	return paths
}

// hasMappingAnnotation returns true if the method has any HTTP mapping annotation.
func (e *SpringMVCExtractor) hasMappingAnnotation(annotations []classfile.ParsedAnnotation) bool {
	for _, ann := range annotations {
		switch ann.Type {
		case springRequestMappingDesc, springGetMappingDesc, springPostMappingDesc,
			springPutMappingDesc, springDeleteMappingDesc, springPatchMappingDesc:
			return true
		}
	}
	return false
}

func (e *SpringMVCExtractor) extractPackage(className string) string {
	lastDot := strings.LastIndex(className, ".")
	if lastDot < 0 {
		return ""
	}
	return className[:lastDot]
}

func (e *SpringMVCExtractor) annotationSimpleNames(annotations []classfile.ParsedAnnotation) []string {
	var names []string
	for _, ann := range annotations {
		names = append(names, "@"+classfile.AnnotationSimpleName(ann.Type))
	}
	return names
}

// ---------------------------------------------------------------------------
// SpringBootExtractor
// ---------------------------------------------------------------------------

// SpringBootExtractor extends Spring MVC extraction for Spring Boot apps.
type SpringBootExtractor struct {
	springMVCExtractor SpringMVCExtractor
}

func (e *SpringBootExtractor) Name() string  { return "Spring Boot" }
func (e *SpringBootExtractor) Priority() int { return 100 }

func (e *SpringBootExtractor) CanHandle(ctx *Context) bool {
	return ctx.BootInf
}

func (e *SpringBootExtractor) Extract(ctx *Context) ([]model.RouteInfo, []model.ClassInfo, error) {
	routes, classes, err := e.springMVCExtractor.Extract(ctx)
	if err != nil {
		return nil, nil, err
	}
	// Tag all routes as Spring Boot.
	for i := range routes {
		routes[i].Framework = "Spring Boot"
	}
	return routes, classes, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func joinPath(base, sub string) string {
	base = strings.TrimRight(base, "/")
	sub = strings.TrimLeft(sub, "/")
	if base == "" && sub == "" {
		return "/"
	}
	if base == "" {
		return "/" + sub
	}
	if sub == "" {
		return "/" + base
	}
	return "/" + base + "/" + sub
}

// FQNFromSlash converts slash-separated class name to dot-separated.
func FQNFromSlash(slash string) string {
	return strings.ReplaceAll(slash, "/", ".")
}
