package model

// RouteInfo represents a single HTTP route extracted from annotations or XML config.
type RouteInfo struct {
	URL         string   // Normalized URL path
	HTTPMethods []string // HTTP methods, empty means all
	ClassName   string   // Fully qualified class name
	MethodName  string   // Handler method name
	Framework   string   // "Spring MVC", "Struts2", "JAX-RS", "Servlet", "WebWork"
	SourceType  string   // "ANNOTATION" or "XML"
	SourceFile  string   // Path within archive
	ArchiveName string   // Source archive name
}

// ClassInfo summarizes a Java class.
type ClassInfo struct {
	FullName    string   // com.example.controller.UserController
	Package     string   // com.example.controller
	FilePath    string   // Actual file path within archive (e.g., com/example/controller/UserController.class)
	SuperClass  string   // java.lang.Object
	Interfaces  []string // Implemented interfaces
	Annotations []string // Class-level annotations
	SpringType  string   // Controller, RestController, Service, Repository, Component
	ArchiveName string   // Source archive name
}

// MethodSummary is a lightweight view of a method.
type MethodSummary struct {
	Name        string
	ReturnType  string
	Parameters  []string
	Annotations []string
}

// FrameworkInfo describes a detected application framework.
type FrameworkInfo struct {
	Name      string   // Spring Boot, Spring MVC, Struts2, etc.
	Version   string   // Version string or empty
	Confidence int     // 0-100
	Evidence  []string // Detection evidence
}

// ControllerInfo describes a controller class extracted from route analysis.
type ControllerInfo struct {
	FilePath  string // Controller file path relative to project root (archive!inner/path/Class.class)
	ClassName string // Fully qualified class name (e.g., com.example.controller.UserController)
	Framework string // Framework type: "Spring MVC", "Spring Boot", "Struts2", "JAX-RS", "Servlet", "WebWork"
}

// ComponentInfo describes a detected library/component.
type ComponentInfo struct {
	Name    string   // MyBatis, Log4j2, Shiro, etc.
	Type    string   // ORM, LOGGING, SECURITY, RPC, CONTAINER, etc.
	Version string   // Version string
	Source  string   // Evidence source (jar filename)
	Jars    []string // All jar files matching this component
}
