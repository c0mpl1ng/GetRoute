package detector

import (
	"strings"

	"GetRoute/internal/classfile"
	"GetRoute/internal/model"
	"GetRoute/internal/xmlconfig"
)

// FrameworkResult holds the result of framework detection.
type FrameworkResult struct {
	Name       string
	Version    string
	Confidence int
	Evidence   []string
}

// DetectFrameworks identifies web frameworks present in the scanned data.
func DetectFrameworks(
	classes map[string]*classfile.ClassFile,
	xmlFiles map[string][]byte,
	manifest map[string]string,
	bootInf bool,
	jarNames []string,
) []model.FrameworkInfo {
	var frameworks []model.FrameworkInfo

	// Spring Boot detection.
	if bootInf {
		version := detectSpringBootVersion(manifest, jarNames)
		frameworks = append(frameworks, model.FrameworkInfo{
			Name:       "Spring Boot",
			Version:    version,
			Confidence: 95,
			Evidence:   []string{"BOOT-INF directory found", "MANIFEST.MF Spring-Boot headers"},
		})
	}

	// Spring MVC detection.
	if detectSpringMVC(classes) {
		version := detectSpringVersion(jarNames)
		evidence := []string{"@Controller/@RestController annotations found"}
		frameworks = append(frameworks, model.FrameworkInfo{
			Name:       "Spring MVC",
			Version:    version,
			Confidence: 90,
			Evidence:   evidence,
		})
	}

	// Eway Framework / DHCC detection (东华医疗).
	if detectEway(xmlFiles, classes) {
		frameworks = append(frameworks, model.FrameworkInfo{
			Name:       "Eway Framework",
			Version:    "2.x",
			Confidence: 90,
			Evidence:   detectEwayEvidence(xmlFiles, classes),
		})
	}

	// Struts2 detection.
	if detectStruts2(xmlFiles, classes) {
		frameworks = append(frameworks, model.FrameworkInfo{
			Name:       "Struts2",
			Version:    detectStruts2Version(jarNames, xmlFiles),
			Confidence: 90,
			Evidence:   detectStruts2Evidence(xmlFiles, classes),
		})
	}

	// WebWork detection.
	if detectWebWork(xmlFiles, classes) {
		frameworks = append(frameworks, model.FrameworkInfo{
			Name:       "WebWork",
			Version:    "2.x",
			Confidence: 85,
			Evidence:   detectWebWorkEvidence(xmlFiles, classes),
		})
	} else if detectEway(xmlFiles, classes) {
		// Eway is built on WebWork, so detect WebWork as a secondary framework.
		frameworks = append(frameworks, model.FrameworkInfo{
			Name:       "WebWork",
			Version:    "2.x",
			Confidence: 85,
			Evidence:   []string{"xwork configuration files found (Eway Framework base)"},
		})
	}

	// JAX-RS detection.
	if detectJAXRS(classes) {
		version := detectJAXRSVersion(jarNames)
		frameworks = append(frameworks, model.FrameworkInfo{
			Name:       "JAX-RS",
			Version:    version,
			Confidence: 85,
			Evidence:   []string{"@Path annotations found on resource classes"},
		})
	}

	// Servlet detection.
	if detectServlet(xmlFiles, classes) {
		frameworks = append(frameworks, model.FrameworkInfo{
			Name:       "Servlet",
			Version:    detectServletVersion(xmlFiles, jarNames),
			Confidence: 80,
			Evidence:   detectServletEvidence(xmlFiles, classes),
		})
	}

	// If no framework detected.
	if len(frameworks) == 0 {
		frameworks = append(frameworks, model.FrameworkInfo{
			Name:       "Unknown",
			Version:    "",
			Confidence: 0,
			Evidence:   []string{"No supported framework markers found"},
		})
	}

	return frameworks
}

func detectEway(xmlFiles map[string][]byte, classes map[string]*classfile.ClassFile) bool {
	// Check for Eway xwork configuration files.
	for name := range xmlFiles {
		if xmlconfig.IsXWorkFile(name) {
			// Check if it contains Eway-specific paths.
			if strings.Contains(name, "com/eway") {
				return true
			}
		}
	}
	// Check for webwork-default.xml (WebWork base config used by Eway).
	for name := range xmlFiles {
		if strings.HasSuffix(strings.ToLower(name), "webwork-default.xml") {
			return true
		}
	}
	// Check for Eway class packages.
	for _, cf := range classes {
		if strings.Contains(cf.ThisClass, "com/eway") {
			return true
		}
	}
	return false
}

func detectEwayEvidence(xmlFiles map[string][]byte, classes map[string]*classfile.ClassFile) []string {
	var evidence []string
	for name := range xmlFiles {
		if xmlconfig.IsXWorkFile(name) && strings.Contains(name, "com/eway") {
			evidence = append(evidence, name)
			if len(evidence) >= 3 {
				break
			}
		}
	}
	if len(evidence) == 0 {
		for name := range xmlFiles {
			if strings.HasSuffix(strings.ToLower(name), "webwork-default.xml") {
				evidence = append(evidence, name)
				break
			}
		}
	}
	for _, cf := range classes {
		if strings.Contains(cf.ThisClass, "com/eway/framework") {
			evidence = append(evidence, "Eway base class: "+cf.ThisClass)
			break
		}
	}
	return evidence
}

func detectSpringBootVersion(manifest map[string]string, jarNames []string) string {
	if v, ok := manifest["Spring-Boot-Version"]; ok && v != "" {
		return v
	}
	for _, name := range jarNames {
		if strings.HasPrefix(strings.ToLower(name), "spring-boot-") {
			for _, cp := range KnownComponents {
				if cp.Name == "Spring Boot" {
					if m := cp.Regex.FindStringSubmatch(name); len(m) > 1 {
						return m[1]
					}
				}
			}
		}
	}
	return ""
}

func detectSpringMVC(classes map[string]*classfile.ClassFile) bool {
	controllerDescs := []string{
		"Lorg/springframework/stereotype/Controller;",
		"Lorg/springframework/web/bind/annotation/RestController;",
	}
	for _, cf := range classes {
		for _, ann := range cf.ClassAnnotations() {
			for _, desc := range controllerDescs {
				if ann.Type == desc {
					return true
				}
			}
		}
	}
	return false
}

func detectSpringVersion(jarNames []string) string {
	for _, name := range jarNames {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "spring-webmvc-") || strings.HasPrefix(lower, "spring-web-") {
			for _, cp := range KnownComponents {
				if cp.Name == "Spring MVC" || cp.Name == "Spring Web" {
					if m := cp.Regex.FindStringSubmatch(name); len(m) > 1 {
						return m[1]
					}
				}
			}
		}
	}
	return ""
}

func detectStruts2(xmlFiles map[string][]byte, classes map[string]*classfile.ClassFile) bool {
	// Check for struts*.xml files.
	for name := range xmlFiles {
		if xmlconfig.IsStrutsFile(name) {
			return true
		}
	}

	return false
}

func detectStruts2Version(jarNames []string, xmlFiles map[string][]byte) string {
	for _, name := range jarNames {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "struts2-core-") {
			for _, cp := range KnownComponents {
				if cp.Name == "Struts2" {
					if m := cp.Regex.FindStringSubmatch(name); len(m) > 1 {
						return m[1]
					}
				}
			}
		}
	}
	return "2.x"
}

func detectStruts2Evidence(xmlFiles map[string][]byte, classes map[string]*classfile.ClassFile) []string {
	var evidence []string
	for name := range xmlFiles {
		if xmlconfig.IsStrutsFile(name) {
			evidence = append(evidence, name)
			if len(evidence) >= 3 {
				break
			}
		}
	}
	return evidence
}

func detectWebWork(xmlFiles map[string][]byte, classes map[string]*classfile.ClassFile) bool {
	for name := range xmlFiles {
		if xmlconfig.IsXWorkFile(name) {
			return true
		}
	}
	for _, cf := range classes {
		if strings.Contains(cf.ThisClass, "com/opensymphony/webwork") ||
			strings.Contains(cf.ThisClass, "com/opensymphony/xwork") {
			return true
		}
	}
	return false
}

func detectWebWorkEvidence(xmlFiles map[string][]byte, classes map[string]*classfile.ClassFile) []string {
	var evidence []string
	for name := range xmlFiles {
		if xmlconfig.IsXWorkFile(name) {
			evidence = append(evidence, name)
			if len(evidence) >= 3 {
				break
			}
		}
	}
	for _, cf := range classes {
		if strings.Contains(cf.SuperClass, "ActionSupport") {
			evidence = append(evidence, "ActionSupport subclass: "+cf.ThisClass)
			break
		}
	}
	return evidence
}

func detectJAXRS(classes map[string]*classfile.ClassFile) bool {
	pathDescs := []string{
		"Ljavax/ws/rs/Path;",
		"Ljakarta/ws/rs/Path;",
	}
	for _, cf := range classes {
		for _, ann := range cf.ClassAnnotations() {
			for _, desc := range pathDescs {
				if ann.Type == desc {
					return true
				}
			}
		}
	}
	return false
}

func detectJAXRSVersion(jarNames []string) string {
	for _, name := range jarNames {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "javax.ws.rs-api-") || strings.HasPrefix(lower, "jakarta.ws.rs-api-") {
			for _, cp := range KnownComponents {
				if cp.Name == "JAX-RS" {
					if m := cp.Regex.FindStringSubmatch(name); len(m) > 1 {
						return m[1]
					}
				}
			}
		}
	}
	return ""
}

func detectServlet(xmlFiles map[string][]byte, classes map[string]*classfile.ClassFile) bool {
	for name := range xmlFiles {
		if isWebXMLFile(name) {
			return true
		}
	}
	servletDescs := []string{
		"Ljavax/servlet/annotation/WebServlet;",
		"Ljakarta/servlet/annotation/WebServlet;",
	}
	for _, cf := range classes {
		for _, ann := range cf.ClassAnnotations() {
			for _, desc := range servletDescs {
				if ann.Type == desc {
					return true
				}
			}
		}
	}
	return false
}

func detectServletVersion(xmlFiles map[string][]byte, jarNames []string) string {
	for _, name := range jarNames {
		if strings.HasPrefix(strings.ToLower(name), "tomcat-embed-core-") {
			for _, cp := range KnownComponents {
				if cp.Name == "Tomcat" {
					if m := cp.Regex.FindStringSubmatch(name); len(m) > 1 {
						return "Tomcat " + m[1]
					}
				}
			}
		}
	}
	return ""
}

func detectServletEvidence(xmlFiles map[string][]byte, classes map[string]*classfile.ClassFile) []string {
	var evidence []string
	for name := range xmlFiles {
		if isWebXMLFile(name) {
			evidence = append(evidence, name)
		}
	}
	for _, cf := range classes {
		for _, ann := range cf.ClassAnnotations() {
			if ann.Type == "Ljavax/servlet/annotation/WebServlet;" ||
				ann.Type == "Ljakarta/servlet/annotation/WebServlet;" {
				evidence = append(evidence, "@WebServlet on "+cf.ThisClass)
				break
			}
		}
	}
	return evidence
}

func isWebXMLFile(name string) bool {
	lower := strings.ToLower(name)
	if lower == "web-inf/web.xml" {
		return true
	}
	base := lower
	if idx := strings.LastIndex(lower, "/"); idx >= 0 {
		base = lower[idx+1:]
	}
	return strings.HasSuffix(base, "web.xml")
}

// GetSupportedFrameworks returns the list of framework names that were detected.
func GetSupportedFrameworks(frameworks []model.FrameworkInfo) string {
	var names []string
	for _, f := range frameworks {
		if f.Name != "Unknown" {
			names = append(names, f.Name)
		}
	}
	if len(names) == 0 {
		return "Unknown"
	}
	return strings.Join(names, ", ")
}
