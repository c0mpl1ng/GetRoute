package extractor

import (
	"strings"

	"GetRoute/internal/classfile"
	"GetRoute/internal/model"
	"GetRoute/internal/xmlconfig"
)

// WebWork class markers.
const (
	webworkActionContextDesc      = "Lcom/opensymphony/webwork/ServletActionContext;"
	webworkActionSupportDesc      = "Lcom/opensymphony/webwork/ActionSupport;"
	webworkXWorkActionSupportDesc = "Lcom/opensymphony/xwork/ActionSupport;"
)

// WebWorkExtractor extracts routes from xwork.xml files for WebWork applications.
type WebWorkExtractor struct{}

func (e *WebWorkExtractor) Name() string  { return "WebWork" }
func (e *WebWorkExtractor) Priority() int { return 75 }

func (e *WebWorkExtractor) CanHandle(ctx *Context) bool {
	// Check for any xwork*.xml file.
	for name := range ctx.XMLFiles {
		if xmlconfig.IsXWorkFile(name) {
			return true
		}
	}

	// Check for WebWork-specific classes.
	for _, cf := range ctx.Classes {
		if strings.Contains(cf.ThisClass, "com/opensymphony/webwork") ||
			strings.Contains(cf.ThisClass, "com/opensymphony/xwork") {
			return true
		}
		for _, ann := range cf.ClassAnnotations() {
			switch ann.Type {
			case webworkActionContextDesc, webworkActionSupportDesc, webworkXWorkActionSupportDesc:
				return true
			}
		}
	}

	return false
}

func (e *WebWorkExtractor) Extract(ctx *Context) ([]model.RouteInfo, []model.ClassInfo, error) {
	var routes []model.RouteInfo
	var classes []model.ClassInfo
	archiveName := ctx.ParentName

	// Collect all packages to resolve namespaces across files.
	allPackages := make(map[string]string) // package name -> namespace

	// First pass: collect package namespaces from all xwork files (including root xwork.xml).
	for name, data := range ctx.XMLFiles {
		if !xmlconfig.IsXWorkFile(name) {
			continue
		}
		xw, err := xmlconfig.ParseXWorkOrStrutsXML(data)
		if err != nil {
			continue
		}
		for _, pkg := range xw.Packages {
			if pkg.Namespace != "" {
				allPackages[pkg.Name] = pkg.Namespace
			}
		}
	}

	// Second pass: extract actions from all xwork files.
	for name, data := range ctx.XMLFiles {
		if !xmlconfig.IsXWorkFile(name) {
			continue
		}
		xw, err := xmlconfig.ParseXWorkOrStrutsXML(data)
		if err != nil {
			continue
		}
		for _, action := range xw.GetAllActions() {
			routes = append(routes, model.RouteInfo{
				URL:         action.URL(),
				HTTPMethods: nil, // All methods
				ClassName:   action.ActionClass,
				MethodName:  action.ActionMethod,
				Framework:   "WebWork",
				SourceType:  "XML",
				SourceFile:  name,
				ArchiveName: archiveName,
			})
		}
	}

	// Also check for struts*.xml files that may contain xwork-compatible config.
	for name, data := range ctx.XMLFiles {
		if !xmlconfig.IsStrutsFile(name) {
			continue
		}
		// Skip if we already parsed it as xwork.
		if xmlconfig.IsXWorkFile(name) {
			continue
		}
		sx, err := xmlconfig.ParseStrutsXML(data)
		if err != nil {
			continue
		}
		for _, action := range sx.GetAllActions() {
			routes = append(routes, model.RouteInfo{
				URL:         action.URL(),
				HTTPMethods: nil,
				ClassName:   action.ActionClass,
				MethodName:  action.ActionMethod,
				Framework:   "WebWork",
				SourceType:  "XML",
				SourceFile:  name,
				ArchiveName: archiveName,
			})
		}
	}

	// Record WebWork/Eway classes.
	for _, cf := range ctx.Classes {
		if !strings.Contains(cf.ThisClass, "com/opensymphony/webwork") &&
			!strings.Contains(cf.ThisClass, "com/opensymphony/xwork") &&
			!strings.Contains(cf.ThisClass, "com/eway") {
			continue
		}

		className := FQNFromSlash(cf.ThisClass)
		classAnnotations := cf.ClassAnnotations()
		classes = append(classes, model.ClassInfo{
			FullName:    className,
			Package:     extractPackageName(className),
			SuperClass:  FQNFromSlash(cf.SuperClass),
			Annotations: annotationSimpleNames(classAnnotations),
			ArchiveName: archiveName,
		})
	}

	return routes, classes, nil
}

// Helper functions shared across extractors.

func extractPackageName(className string) string {
	lastDot := strings.LastIndex(className, ".")
	if lastDot < 0 {
		return ""
	}
	return className[:lastDot]
}

func annotationSimpleNames(annotations []classfile.ParsedAnnotation) []string {
	var names []string
	for _, ann := range annotations {
		names = append(names, "@"+classfile.AnnotationSimpleName(ann.Type))
	}
	return names
}
