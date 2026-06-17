package indexer

import (
	"sort"
	"strings"

	"GetRoute/internal/model"
)

// Indexer deduplicates, sorts, and normalizes route and class data.
type Indexer struct {
	routes     []model.RouteInfo
	classes    []model.ClassInfo
	frameworks []model.FrameworkInfo
	components []model.ComponentInfo
}

// NewIndexer creates a new Indexer.
func NewIndexer() *Indexer {
	return &Indexer{}
}

// AddRoutes adds routes to the indexer.
func (idx *Indexer) AddRoutes(routes []model.RouteInfo) {
	idx.routes = append(idx.routes, routes...)
}

// AddClasses adds classes to the indexer.
func (idx *Indexer) AddClasses(classes []model.ClassInfo) {
	idx.classes = append(idx.classes, classes...)
}

// AddFrameworks adds framework info.
func (idx *Indexer) AddFrameworks(frameworks []model.FrameworkInfo) {
	idx.frameworks = append(idx.frameworks, frameworks...)
}

// AddComponents adds component info.
func (idx *Indexer) AddComponents(components []model.ComponentInfo) {
	idx.components = append(idx.components, components...)
}

// Build deduplicates, normalizes, and sorts all data.
func (idx *Indexer) Build() {
	idx.routes = deduplicateRoutes(idx.routes)
	idx.classes = deduplicateClasses(idx.classes)
	idx.components = deduplicateComponents(idx.components)

	sortRoutes(idx.routes)
	sortClasses(idx.classes)
	sortComponents(idx.components)
}

// Routes returns the indexed routes.
func (idx *Indexer) Routes() []model.RouteInfo {
	return idx.routes
}

// Classes returns the indexed classes.
func (idx *Indexer) Classes() []model.ClassInfo {
	return idx.classes
}

// Frameworks returns the framework info.
func (idx *Indexer) Frameworks() []model.FrameworkInfo {
	return idx.frameworks
}

// Components returns the component info.
func (idx *Indexer) Components() []model.ComponentInfo {
	return idx.components
}

func deduplicateRoutes(routes []model.RouteInfo) []model.RouteInfo {
	seen := make(map[string]bool)
	var result []model.RouteInfo
	for _, r := range routes {
		key := routeKey(r)
		if seen[key] {
			continue
		}
		seen[key] = true
		r.URL = normalizeURL(r.URL)
		result = append(result, r)
	}
	return result
}

func routeKey(r model.RouteInfo) string {
	methods := strings.Join(r.HTTPMethods, ",")
	if methods == "" {
		methods = "*"
	}
	return methods + " " + r.URL + " " + r.ClassName + " " + r.MethodName + " " + r.Framework
}

func deduplicateClasses(classes []model.ClassInfo) []model.ClassInfo {
	seen := make(map[string]bool)
	var result []model.ClassInfo
	for _, c := range classes {
		key := c.FullName + "@" + c.ArchiveName
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, c)
	}
	return result
}

func deduplicateComponents(components []model.ComponentInfo) []model.ComponentInfo {
	seen := make(map[string]bool)
	var result []model.ComponentInfo
	for _, c := range components {
		if seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		result = append(result, c)
	}
	return result
}

func sortRoutes(routes []model.RouteInfo) {
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].URL != routes[j].URL {
			return routes[i].URL < routes[j].URL
		}
		mi := strings.Join(routes[i].HTTPMethods, ",")
		mj := strings.Join(routes[j].HTTPMethods, ",")
		if mi != mj {
			return mi < mj
		}
		return routes[i].ClassName < routes[j].ClassName
	})
}

func sortClasses(classes []model.ClassInfo) {
	sort.Slice(classes, func(i, j int) bool {
		if classes[i].Package != classes[j].Package {
			return classes[i].Package < classes[j].Package
		}
		return classes[i].FullName < classes[j].FullName
	})
}

func sortComponents(components []model.ComponentInfo) {
	sort.Slice(components, func(i, j int) bool {
		if components[i].Type != components[j].Type {
			return components[i].Type < components[j].Type
		}
		return components[i].Name < components[j].Name
	})
}

func normalizeURL(url string) string {
	// Ensure leading /.
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	// Collapse multiple /.
	for strings.Contains(url, "//") {
		url = strings.ReplaceAll(url, "//", "/")
	}
	// Remove trailing / unless root.
	if len(url) > 1 && strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}
	return url
}
