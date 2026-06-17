package extractor

import (
	"strings"

	"GetRoute/internal/classfile"
	"GetRoute/internal/model"
	"GetRoute/internal/xmlconfig"
)

// Servlet annotation descriptors.
const (
	servletWebServletDesc     = "Ljavax/servlet/annotation/WebServlet;"
	servletJakartaServletDesc = "Ljakarta/servlet/annotation/WebServlet;"
	servletWebFilterDesc      = "Ljavax/servlet/annotation/WebFilter;"
	servletJakartaFilterDesc  = "Ljakarta/servlet/annotation/WebFilter;"
)

// ServletExtractor extracts routes from web.xml and @WebServlet annotations.
type ServletExtractor struct{}

func (e *ServletExtractor) Name() string  { return "Servlet" }
func (e *ServletExtractor) Priority() int { return 60 }

func (e *ServletExtractor) CanHandle(ctx *Context) bool {
	// Check for any web.xml file (standard or non-standard paths).
	for name := range ctx.XMLFiles {
		if isWebXMLFile(name) {
			return true
		}
	}

	// Check for @WebServlet annotations.
	for _, cf := range ctx.Classes {
		for _, ann := range cf.ClassAnnotations() {
			if ann.Type == servletWebServletDesc || ann.Type == servletJakartaServletDesc {
				return true
			}
		}
	}

	return false
}

func (e *ServletExtractor) Extract(ctx *Context) ([]model.RouteInfo, []model.ClassInfo, error) {
	var routes []model.RouteInfo
	var classes []model.ClassInfo
	archiveName := ctx.ParentName

	// 1. Extract from ALL web.xml files (standard and non-standard paths).
	for name, data := range ctx.XMLFiles {
		if !isWebXMLFile(name) {
			continue
		}

		w, err := xmlconfig.ParseWebXML(data)
		if err != nil {
			continue
		}

		// Build servlet-name → servlet-class map.
		servletClasses := make(map[string]string)
		for _, s := range w.Servlets {
			servletClasses[s.Name] = s.Class
		}

		for _, sm := range w.ServletMaps {
			className := servletClasses[sm.Name]
			routes = append(routes, model.RouteInfo{
				URL:         normalizeServletPattern(sm.Pattern),
				HTTPMethods: nil, // All methods
				ClassName:   className,
				MethodName:  "", // XML-defined
				Framework:   "Servlet",
				SourceType:  "XML",
				SourceFile:  name,
				ArchiveName: archiveName,
			})
		}

		// Also extract filter mappings.
		filterClasses := make(map[string]string)
		for _, f := range w.Filters {
			filterClasses[f.Name] = f.Class
		}
		for _, fm := range w.FilterMaps {
			className := filterClasses[fm.Name]
			if className == "" {
				className = fm.Name
			}
			routes = append(routes, model.RouteInfo{
				URL:         normalizeServletPattern(fm.Pattern),
				HTTPMethods: nil,
				ClassName:   className,
				MethodName:  "",
				Framework:   "Servlet",
				SourceType:  "XML",
				SourceFile:  name,
				ArchiveName: archiveName,
			})
		}
	}

	// 2. Extract from @WebServlet annotations.
	for _, cf := range ctx.Classes {
		classAnnotations := cf.ClassAnnotations()
		var webServlet *classfile.ParsedAnnotation
		for i := range classAnnotations {
			if classAnnotations[i].Type == servletWebServletDesc || classAnnotations[i].Type == servletJakartaServletDesc {
				webServlet = &classAnnotations[i]
				break
			}
		}
		if webServlet == nil {
			continue
		}

		className := FQNFromSlash(cf.ThisClass)
		srcJar := cf.ArchiveName
		if srcJar == "" {
			srcJar = archiveName
		}
		classes = append(classes, model.ClassInfo{
			FullName:    className,
			Package:     extractPackageName(className),
			SuperClass:  FQNFromSlash(cf.SuperClass),
			Annotations: annotationSimpleNames(classAnnotations),
			ArchiveName: srcJar,
		})

		// Extract URL patterns from "value" or "urlPatterns" element.
		patterns := e.extractServletPatterns(webServlet)
		for _, pattern := range patterns {
			routes = append(routes, model.RouteInfo{
				URL:         normalizeServletPattern(pattern),
				HTTPMethods: nil,
				ClassName:   className,
				MethodName:  "",
				Framework:   "Servlet",
				SourceType:  "ANNOTATION",
				SourceFile:  cf.FilePath,
				ArchiveName: srcJar,
			})
		}
	}

	return routes, classes, nil
}

func (e *ServletExtractor) extractServletPatterns(ann *classfile.ParsedAnnotation) []string {
	var patterns []string

	if elem := ann.GetElement("value"); elem != nil {
		for _, p := range elem.AsStringArray() {
			p = strings.TrimSpace(p)
			if p != "" {
				patterns = append(patterns, p)
			}
		}
	}

	if elem := ann.GetElement("urlPatterns"); elem != nil {
		for _, p := range elem.AsStringArray() {
			p = strings.TrimSpace(p)
			if p != "" {
				patterns = append(patterns, p)
			}
		}
	}

	return patterns
}

func normalizeServletPattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "/"
	}
	if !strings.HasPrefix(pattern, "/") && !strings.HasPrefix(pattern, "*") {
		pattern = "/" + pattern
	}
	return pattern
}

// isWebXMLFile checks if a filename is a web.xml file (standard or non-standard paths).
func isWebXMLFile(name string) bool {
	lower := strings.ToLower(name)
	// Standard path.
	if lower == "web-inf/web.xml" {
		return true
	}
	// Non-standard paths like withoutcas_web.xml, withcas_web.xml, or any *web.xml.
	base := lower
	if idx := strings.LastIndex(lower, "/"); idx >= 0 {
		base = lower[idx+1:]
	}
	return strings.HasSuffix(base, "web.xml")
}
