package detector

import (
	"path/filepath"
	"sort"
	"strings"

	"GetRoute/internal/model"
)

// ComponentResult holds a matched component with its version.
type ComponentResult struct {
	Name    string
	Type    string
	Version string
	Jars    []string
}

// DetectComponents scans jar filenames for known components.
func DetectComponents(jarNames []string) []model.ComponentInfo {
	// Deduplicate jar names by base name, keeping original path.
	baseToOriginal := make(map[string]string)
	var unique []string
	for _, name := range jarNames {
		base := filepath.Base(name)
		if _, ok := baseToOriginal[base]; !ok {
			baseToOriginal[base] = name
			unique = append(unique, base)
		}
	}

	// Match each jar against known component patterns.
	componentJars := make(map[string]*ComponentResult)

	for _, jarName := range unique {
		for _, cp := range KnownComponents {
			matches := cp.Regex.FindStringSubmatch(jarName)
			if matches == nil {
				continue
			}
			version := ""
			if len(matches) > 1 {
				version = matches[1]
				// Clean version suffix.
				version = cleanVersion(version)
			}

			key := cp.Name
			originalName := baseToOriginal[jarName]
			if existing, ok := componentJars[key]; ok {
				existing.Jars = append(existing.Jars, originalName)
				if existing.Version == "" && version != "" {
					existing.Version = version
				}
			} else {
				componentJars[key] = &ComponentResult{
					Name:    cp.Name,
					Type:    cp.Category,
					Version: version,
					Jars:    []string{originalName},
				}
			}
			break // One jar matches only one component (first match wins).
		}

		// Also check for unversioned framework jars that don't match the regex.
		checkUnversionedJars(componentJars, jarName, baseToOriginal[jarName])
	}

	// Convert to model.ComponentInfo, sorted by type then name.
	var results []model.ComponentInfo
	for _, cr := range componentJars {
		results = append(results, model.ComponentInfo{
			Name:    cr.Name,
			Type:    cr.Type,
			Version: cr.Version,
			Source:  strings.Join(cr.Jars, ", "),
			Jars:    cr.Jars,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Type != results[j].Type {
			return results[i].Type < results[j].Type
		}
		return results[i].Name < results[j].Name
	})

	return results
}

func checkUnversionedJars(componentJars map[string]*ComponentResult, jarName, originalName string) {
	lower := strings.ToLower(jarName)
	// Some jars don't follow the standard name-version.jar pattern.
	unversioned := map[string]string{
		"servlet-api":            "Servlet API",
		"jsp-api":                "JSP API",
		"jstl":                   "JSTL",
		"standard":               "JSTL Standard",
		"jta":                    "JTA",
		"activation":             "Java Activation",
		"mail":                   "JavaMail",
		"jaxb-api":               "JAXB",
		"websocket-api":          "WebSocket API",
		"el-api":                 "EL API",
		"jaxen":                  "Jaxen",
		"dom4j":                  "Dom4J",
		"xstream":                "XStream",
		"ognl":                   "OGNL",
		"cglib":                  "CGLib",
		"asm":                    "ASM",
		"javassist":              "Javassist",
		"antlr":                  "ANTLR",
		"snakeyaml":              "SnakeYAML",
		"jasypt":                 "Jasypt",
		"bcprov":                 "BouncyCastle",
		"bcpkix":                 "BouncyCastle PKIX",
	}

	for prefix, name := range unversioned {
		if strings.HasPrefix(lower, prefix) && strings.HasSuffix(lower, ".jar") {
			if _, exists := componentJars[name]; !exists {
				componentJars[name] = &ComponentResult{
					Name:    name,
					Type:    "Utility",
					Version: "",
					Jars:    []string{originalName},
				}
			}
		}
	}
}

func cleanVersion(v string) string {
	// Remove common qualifier suffixes.
	suffixes := []string{".RELEASE", ".Final", ".GA", "-SNAPSHOT", "-RELEASE"}
	for _, s := range suffixes {
		if strings.HasSuffix(strings.ToLower(v), strings.ToLower(s)) {
			v = v[:len(v)-len(s)]
			break
		}
	}
	return v
}
