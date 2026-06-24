package extractor

import (
	"strings"

	"GetRoute/internal/classfile"
	"GetRoute/internal/model"
	"GetRoute/internal/xmlconfig"
)

// Struts annotation descriptors.
const (
	strutsActionDesc        = "Lorg/apache/struts2/convention/annotation/Action;"
	strutsNamespaceDesc     = "Lorg/apache/struts2/convention/annotation/Namespace;"
	strutsResultDesc        = "Lorg/apache/struts2/convention/annotation/Result;"
	strutsActionsDesc       = "Lorg/apache/struts2/convention/annotation/Actions;"
	strutsParentPackageDesc = "Lorg/apache/struts2/convention/annotation/ParentPackage;"
)

// Struts2Extractor extracts routes from struts.xml and Struts2 annotations.
type Struts2Extractor struct{}

func (e *Struts2Extractor) Name() string  { return "Struts2" }
func (e *Struts2Extractor) Priority() int { return 80 }

func (e *Struts2Extractor) CanHandle(ctx *Context) bool {
	// Check for any struts*.xml file.
	for name := range ctx.XMLFiles {
		if xmlconfig.IsStrutsFile(name) {
			return true
		}
	}

	// Check web.xml for Struts2 filter.
	if data, ok := e.findWebXML(ctx.XMLFiles); ok {
		w, err := xmlconfig.ParseWebXML(data)
		if err == nil && w.HasStrutsFilter() {
			return true
		}
	}

	// Check classes for Struts2 annotations.
	for _, cf := range ctx.Classes {
		for _, ann := range cf.ClassAnnotations() {
			switch ann.Type {
			case strutsActionDesc, strutsNamespaceDesc, strutsActionsDesc:
				return true
			}
		}
		// Check if extends ActionSupport.
		if strings.Contains(cf.SuperClass, "ActionSupport") {
			return true
		}
	}

	return false
}

func (e *Struts2Extractor) Extract(ctx *Context) ([]model.RouteInfo, []model.ClassInfo, error) {
	var routes []model.RouteInfo
	var classes []model.ClassInfo
	archiveName := ctx.ParentName

	// 1. Extract from all struts*.xml files.
	for name, data := range ctx.XMLFiles {
		if !xmlconfig.IsStrutsFile(name) {
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
				Framework:   "Struts2",
				SourceType:  "XML",
				SourceFile:  ctx.FindClassFile(action.ActionClass),
				ArchiveName: archiveName,
			})
		}
	}

	// 2. Extract from web.xml Struts filter if present.
	if data, ok := e.findWebXML(ctx.XMLFiles); ok {
		w, err := xmlconfig.ParseWebXML(data)
		if err == nil && w.HasStrutsFilter() {
			// Routes already extracted via struts.xml above; web.xml confirms Struts2.
			_ = w
		}
	}

	// 3. Extract from Struts2 annotations on classes.
	for _, cf := range ctx.Classes {
		classAnnotations := cf.ClassAnnotations()
		hasAction := false
		for _, ann := range classAnnotations {
			if ann.Type == strutsActionDesc || ann.Type == strutsActionsDesc {
				hasAction = true
				break
			}
		}
		if !hasAction {
			continue
		}

		className := FQNFromSlash(cf.ThisClass)
		classNS := e.extractNamespace(classAnnotations)

		srcJar := cf.ArchiveName
		if srcJar == "" {
			srcJar = archiveName
		}

		classes = append(classes, model.ClassInfo{
			FullName:    className,
			Package:     extractPackageName(className),
			FilePath:    cf.FilePath,
			SuperClass:  FQNFromSlash(cf.SuperClass),
			Annotations: annotationSimpleNames(classAnnotations),
			SpringType:  "",
			ArchiveName: srcJar,
		})

		for _, method := range cf.Methods {
			methodAnnotations := method.MethodAnnotations(cf.ConstantPool)
			for _, ann := range methodAnnotations {
				if ann.Type == strutsActionDesc {
					actionPath := e.extractAnnotationValue(ann)
					url := joinPath(classNS, actionPath)
					routes = append(routes, model.RouteInfo{
						URL:         url,
						HTTPMethods: nil,
						ClassName:   className,
						MethodName:  method.Name,
						Framework:   "Struts2",
						SourceType:  "ANNOTATION",
						SourceFile:  cf.FilePath,
						ArchiveName: srcJar,
					})
				}
			}
		}
	}

	return routes, classes, nil
}

func (e *Struts2Extractor) findWebXML(xmlFiles map[string][]byte) ([]byte, bool) {
	if data, ok := xmlFiles["WEB-INF/web.xml"]; ok {
		return data, true
	}
	for name, data := range xmlFiles {
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, "web.xml") {
			return data, true
		}
	}
	return nil, false
}

func (e *Struts2Extractor) extractNamespace(annotations []classfile.ParsedAnnotation) string {
	for _, ann := range annotations {
		if ann.Type == strutsNamespaceDesc {
			if elem := ann.GetElement("value"); elem != nil {
				ns := elem.AsString()
				if ns != "" {
					return ns
				}
			}
		}
	}
	return ""
}

func (e *Struts2Extractor) extractAnnotationValue(ann classfile.ParsedAnnotation) string {
	if elem := ann.GetElement("value"); elem != nil {
		return elem.AsString()
	}
	return ""
}
