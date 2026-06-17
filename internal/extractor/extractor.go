package extractor

import (
	"sort"

	"GetRoute/internal/classfile"
	"GetRoute/internal/model"
)

// Context carries all parsed data an extractor may need.
type Context struct {
	Classes    map[string]*classfile.ClassFile // Key: slash-separated class path
	XMLFiles   map[string][]byte               // Key: filename within archive
	Manifest   map[string]string               // Main manifest attributes
	BootInf    bool                            // Has BOOT-INF/ directory
	JarNames   []string                        // All jar filenames
	ParentName string                          // Parent archive name for nested archives
}

// RouteExtractor is the interface for all framework-specific route extractors.
type RouteExtractor interface {
	Name() string
	Priority() int // Higher = checked first
	CanHandle(ctx *Context) bool
	Extract(ctx *Context) ([]model.RouteInfo, []model.ClassInfo, error)
}

// Registry holds all extractors and runs them in priority order.
type Registry struct {
	extractors []RouteExtractor
}

// NewRegistry creates a new extractor registry with all framework extractors.
func NewRegistry() *Registry {
	return &Registry{
		extractors: []RouteExtractor{
			&SpringBootExtractor{},
			&SpringMVCExtractor{},
			&Struts2Extractor{},
			&WebWorkExtractor{},
			&JAXRSExtractor{},
			&ServletExtractor{},
		},
	}
}

// ExtractAll runs all capable extractors and aggregates results.
func (reg *Registry) ExtractAll(ctx *Context) ([]model.RouteInfo, []model.ClassInfo, error) {
	sort.Slice(reg.extractors, func(i, j int) bool {
		return reg.extractors[i].Priority() > reg.extractors[j].Priority()
	})

	var allRoutes []model.RouteInfo
	var allClasses []model.ClassInfo

	for _, ext := range reg.extractors {
		if !ext.CanHandle(ctx) {
			continue
		}
		routes, classes, err := ext.Extract(ctx)
		if err != nil {
			return nil, nil, err
		}
		allRoutes = append(allRoutes, routes...)
		allClasses = append(allClasses, classes...)
	}

	return allRoutes, allClasses, nil
}
