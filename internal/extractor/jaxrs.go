package extractor

import (
	"strings"

	"GetRoute/internal/classfile"
	"GetRoute/internal/model"
)

// JAX-RS annotation descriptors (Java EE and Jakarta EE).
const (
	jaxrsPathDesc    = "Ljavax/ws/rs/Path;"
	jaxrsGETDesc     = "Ljavax/ws/rs/GET;"
	jaxrsPOSTDesc    = "Ljavax/ws/rs/POST;"
	jaxrsPUTDesc     = "Ljavax/ws/rs/PUT;"
	jaxrsDELETEDesc  = "Ljavax/ws/rs/DELETE;"
	jaxrsHEADDesc    = "Ljavax/ws/rs/HEAD;"
	jaxrsOPTIONSDesc = "Ljavax/ws/rs/OPTIONS;"
	jaxrsPATCHDesc   = "Ljavax/ws/rs/PATCH;"

	// Jakarta EE variants.
	jakartaPathDesc    = "Ljakarta/ws/rs/Path;"
	jakartaGETDesc     = "Ljakarta/ws/rs/GET;"
	jakartaPOSTDesc    = "Ljakarta/ws/rs/POST;"
	jakartaPUTDesc     = "Ljakarta/ws/rs/PUT;"
	jakartaDELETEDesc  = "Ljakarta/ws/rs/DELETE;"
	jakartaHEADDesc    = "Ljakarta/ws/rs/HEAD;"
	jakartaOPTIONSDesc = "Ljakarta/ws/rs/OPTIONS;"
	jakartaPATCHDesc   = "Ljakarta/ws/rs/PATCH;"

	jaxrsApplicationPathDesc = "Ljavax/ws/rs/ApplicationPath;"
	jakartaApplicationPathDesc = "Ljakarta/ws/rs/ApplicationPath;"
)

// JAXRSExtractor extracts routes from JAX-RS annotated resources.
type JAXRSExtractor struct{}

func (e *JAXRSExtractor) Name() string  { return "JAX-RS" }
func (e *JAXRSExtractor) Priority() int { return 70 }

func (e *JAXRSExtractor) CanHandle(ctx *Context) bool {
	for _, cf := range ctx.Classes {
		for _, ann := range cf.ClassAnnotations() {
			if ann.Type == jaxrsPathDesc || ann.Type == jakartaPathDesc {
				return true
			}
		}
	}
	return false
}

func (e *JAXRSExtractor) Extract(ctx *Context) ([]model.RouteInfo, []model.ClassInfo, error) {
	var routes []model.RouteInfo
	var classes []model.ClassInfo

	for _, cf := range ctx.Classes {
		classAnnotations := cf.ClassAnnotations()
		classPath := e.extractPath(classAnnotations)
		if classPath == "" {
			continue
		}

		archiveName := cf.ArchiveName
		if archiveName == "" {
			archiveName = ctx.ParentName
		}

		className := FQNFromSlash(cf.ThisClass)
		classes = append(classes, model.ClassInfo{
			FullName:    className,
			Package:     extractPackageName(className),
			SuperClass:  FQNFromSlash(cf.SuperClass),
			Annotations: annotationSimpleNames(classAnnotations),
			ArchiveName: archiveName,
		})

		for _, method := range cf.Methods {
			methodAnnotations := method.MethodAnnotations(cf.ConstantPool)
			httpMethod := e.extractHTTPMethod(methodAnnotations)
			if httpMethod == "" {
				continue
			}

			methodPath := e.extractPath(methodAnnotations)
			url := joinPath(classPath, methodPath)

			routes = append(routes, model.RouteInfo{
				URL:         url,
				HTTPMethods: []string{httpMethod},
				ClassName:   className,
				MethodName:  method.Name,
				Framework:   "JAX-RS",
				SourceType:  "ANNOTATION",
				SourceFile:  cf.FilePath,
				ArchiveName: archiveName,
			})
		}
	}

	return routes, classes, nil
}

func (e *JAXRSExtractor) extractPath(annotations []classfile.ParsedAnnotation) string {
	for _, ann := range annotations {
		if ann.Type == jaxrsPathDesc || ann.Type == jakartaPathDesc {
			if elem := ann.GetElement("value"); elem != nil {
				path := strings.TrimSpace(elem.AsString())
				if path != "" {
					return path
				}
			}
		}
	}
	return ""
}

func (e *JAXRSExtractor) extractHTTPMethod(annotations []classfile.ParsedAnnotation) string {
	methodMap := map[string]string{
		jaxrsGETDesc:     "GET",
		jaxrsPOSTDesc:    "POST",
		jaxrsPUTDesc:     "PUT",
		jaxrsDELETEDesc:  "DELETE",
		jaxrsHEADDesc:    "HEAD",
		jaxrsOPTIONSDesc: "OPTIONS",
		jaxrsPATCHDesc:   "PATCH",
		jakartaGETDesc:     "GET",
		jakartaPOSTDesc:    "POST",
		jakartaPUTDesc:     "PUT",
		jakartaDELETEDesc:  "DELETE",
		jakartaHEADDesc:    "HEAD",
		jakartaOPTIONSDesc: "OPTIONS",
		jakartaPATCHDesc:   "PATCH",
	}

	for _, ann := range annotations {
		if m, ok := methodMap[ann.Type]; ok {
			return m
		}
	}
	return ""
}

// IsJAXRSAnnotation checks if an annotation descriptor is a JAX-RS annotation.
func IsJAXRSAnnotation(desc string) bool {
	switch desc {
	case jaxrsPathDesc, jaxrsGETDesc, jaxrsPOSTDesc, jaxrsPUTDesc, jaxrsDELETEDesc,
		jaxrsHEADDesc, jaxrsOPTIONSDesc, jaxrsPATCHDesc, jaxrsApplicationPathDesc,
		jakartaPathDesc, jakartaGETDesc, jakartaPOSTDesc, jakartaPUTDesc, jakartaDELETEDesc,
		jakartaHEADDesc, jakartaOPTIONSDesc, jakartaPATCHDesc, jakartaApplicationPathDesc:
		return true
	}
	return false
}
