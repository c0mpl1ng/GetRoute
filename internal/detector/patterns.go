package detector

import "regexp"

// ComponentPattern defines how to detect a component from jar filenames.
type ComponentPattern struct {
	Name     string
	Category string
	Regex    *regexp.Regexp
	Aliases  []string
}

// versionPattern matches common Maven version suffixes.
const versionPattern = `(\d+\.\d+(?:\.\d+)?(?:[.\-_][\w.\-]+)?(?:-SNAPSHOT|\.RELEASE|\.Final|\.GA|\.SR\d*)?(?:-M\d+)?(?:-RC\d+)?(?:-alpha\d+)?(?:-beta\d+)?)`

// KnownComponents is the database of known component patterns.
var KnownComponents = []ComponentPattern{
	// Web Frameworks
	{Name: "Spring MVC", Category: "Framework", Regex: compile(`(?i)^spring-webmvc-` + versionPattern + `\.jar$`)},
	{Name: "Spring Web", Category: "Framework", Regex: compile(`(?i)^spring-web-` + versionPattern + `\.jar$`)},
	{Name: "Spring Boot", Category: "Framework", Regex: compile(`(?i)^spring-boot-` + versionPattern + `\.jar$`)},
	{Name: "Spring Boot Starter", Category: "Framework", Regex: compile(`(?i)^spring-boot-starter-web-` + versionPattern + `\.jar$`)},
	{Name: "Spring Cloud", Category: "Framework", Regex: compile(`(?i)^spring-cloud-` + versionPattern + `\.jar$`)},
	{Name: "Struts2", Category: "Framework", Regex: compile(`(?i)^struts2-core-` + versionPattern + `\.jar$`)},
	{Name: "JAX-RS", Category: "Framework", Regex: compile(`(?i)^(?:javax\.ws\.rs|jakarta\.ws\.rs)-api-` + versionPattern + `\.jar$`)},

	// ORM / Database
	{Name: "MyBatis", Category: "ORM", Regex: compile(`(?i)^mybatis-` + versionPattern + `\.jar$`)},
	{Name: "MyBatis Spring", Category: "ORM", Regex: compile(`(?i)^mybatis-spring-` + versionPattern + `\.jar$`)},
	{Name: "MyBatis Plus", Category: "ORM", Regex: compile(`(?i)^mybatis-plus-` + versionPattern + `\.jar$`)},
	{Name: "Hibernate", Category: "ORM", Regex: compile(`(?i)^hibernate-core-` + versionPattern + `\.jar$`)},
	{Name: "JPA", Category: "ORM", Regex: compile(`(?i)^hibernate-jpa-` + versionPattern + `\.jar$`)},
	{Name: "Druid", Category: "Database", Regex: compile(`(?i)^druid-` + versionPattern + `\.jar$`)},
	{Name: "HikariCP", Category: "Database", Regex: compile(`(?i)^HikariCP-` + versionPattern + `\.jar$`)},
	{Name: "C3P0", Category: "Database", Regex: compile(`(?i)^c3p0-` + versionPattern + `\.jar$`)},

	// Security
	{Name: "Shiro", Category: "Security", Regex: compile(`(?i)^shiro-core-` + versionPattern + `\.jar$`)},
	{Name: "Shiro Web", Category: "Security", Regex: compile(`(?i)^shiro-web-` + versionPattern + `\.jar$`)},
	{Name: "Spring Security", Category: "Security", Regex: compile(`(?i)^spring-security-core-` + versionPattern + `\.jar$`)},
	{Name: "Spring Security Web", Category: "Security", Regex: compile(`(?i)^spring-security-web-` + versionPattern + `\.jar$`)},
	{Name: "SaToken", Category: "Security", Regex: compile(`(?i)^sa-token-` + versionPattern + `\.jar$`)},

	// Logging
	{Name: "Log4j2", Category: "Logging", Regex: compile(`(?i)^log4j-core-` + versionPattern + `\.jar$`)},
	{Name: "Log4j", Category: "Logging", Regex: compile(`(?i)^log4j-` + versionPattern + `\.jar$`)},
	{Name: "Logback", Category: "Logging", Regex: compile(`(?i)^logback-classic-` + versionPattern + `\.jar$`)},
	{Name: "Logback Core", Category: "Logging", Regex: compile(`(?i)^logback-core-` + versionPattern + `\.jar$`)},
	{Name: "SLF4J", Category: "Logging", Regex: compile(`(?i)^slf4j-api-` + versionPattern + `\.jar$`)},

	// RPC
	{Name: "Dubbo", Category: "RPC", Regex: compile(`(?i)^dubbo-` + versionPattern + `\.jar$`)},
	{Name: "gRPC", Category: "RPC", Regex: compile(`(?i)^grpc-core-` + versionPattern + `\.jar$`)},

	// Containers
	{Name: "Tomcat", Category: "Container", Regex: compile(`(?i)^tomcat-embed-core-` + versionPattern + `\.jar$`)},
	{Name: "Tomcat Jasper", Category: "Container", Regex: compile(`(?i)^tomcat-jasper-` + versionPattern + `\.jar$`)},
	{Name: "Jetty", Category: "Container", Regex: compile(`(?i)^jetty-server-` + versionPattern + `\.jar$`)},
	{Name: "Undertow", Category: "Container", Regex: compile(`(?i)^undertow-core-` + versionPattern + `\.jar$`)},

	// JSON
	{Name: "Jackson", Category: "JSON", Regex: compile(`(?i)^jackson-databind-` + versionPattern + `\.jar$`)},
	{Name: "Jackson Core", Category: "JSON", Regex: compile(`(?i)^jackson-core-` + versionPattern + `\.jar$`)},
	{Name: "Jackson Annotations", Category: "JSON", Regex: compile(`(?i)^jackson-annotations-` + versionPattern + `\.jar$`)},
	{Name: "Fastjson", Category: "JSON", Regex: compile(`(?i)^fastjson-` + versionPattern + `\.jar$`)},
	{Name: "Fastjson2", Category: "JSON", Regex: compile(`(?i)^fastjson2-` + versionPattern + `\.jar$`)},
	{Name: "Gson", Category: "JSON", Regex: compile(`(?i)^gson-` + versionPattern + `\.jar$`)},

	// Template Engines
	{Name: "Thymeleaf", Category: "Template", Regex: compile(`(?i)^thymeleaf-` + versionPattern + `\.jar$`)},
	{Name: "FreeMarker", Category: "Template", Regex: compile(`(?i)^freemarker-` + versionPattern + `\.jar$`)},
	{Name: "Velocity", Category: "Template", Regex: compile(`(?i)^velocity-` + versionPattern + `\.jar$`)},
	{Name: "Beetl", Category: "Template", Regex: compile(`(?i)^beetl-` + versionPattern + `\.jar$`)},

	// HTTP Clients
	{Name: "HttpClient", Category: "HTTP", Regex: compile(`(?i)^httpclient-` + versionPattern + `\.jar$`)},
	{Name: "OkHttp", Category: "HTTP", Regex: compile(`(?i)^okhttp-` + versionPattern + `\.jar$`)},

	// Chinese Frameworks
	{Name: "JReap", Category: "Framework", Regex: compile(`(?i)^jreap-` + versionPattern + `\.jar$`)},
	{Name: "RuoYi", Category: "Framework", Regex: compile(`(?i)^ruoyi-` + versionPattern + `\.jar$`)},
	{Name: "Jeecg", Category: "Framework", Regex: compile(`(?i)^jeecg-` + versionPattern + `\.jar$`)},
	{Name: "DHCC", Category: "Framework", Regex: compile(`(?i)^dhcc-` + versionPattern + `\.jar$`)},
	{Name: "Eway Framework", Category: "Framework", Regex: compile(`(?i)^eway-` + versionPattern + `\.jar$`)},

	// Apache Commons
	{Name: "Commons IO", Category: "Utility", Regex: compile(`(?i)^commons-io-` + versionPattern + `\.jar$`)},
	{Name: "Commons Lang", Category: "Utility", Regex: compile(`(?i)^commons-lang3?-` + versionPattern + `\.jar$`)},
	{Name: "Commons BeanUtils", Category: "Utility", Regex: compile(`(?i)^commons-beanutils-` + versionPattern + `\.jar$`)},
	{Name: "Commons Collections", Category: "Utility", Regex: compile(`(?i)^commons-collections` + versionPattern + `\.jar$`)},
	{Name: "Commons FileUpload", Category: "Utility", Regex: compile(`(?i)^commons-fileupload-` + versionPattern + `\.jar$`)},

	// Google
	{Name: "Guava", Category: "Utility", Regex: compile(`(?i)^guava-` + versionPattern + `\.jar$`)},

	// Others
	{Name: "Ehcache", Category: "Cache", Regex: compile(`(?i)^ehcache-` + versionPattern + `\.jar$`)},
	{Name: "Redis", Category: "Cache", Regex: compile(`(?i)^jedis-` + versionPattern + `\.jar$`)},
	{Name: "Lettuce", Category: "Cache", Regex: compile(`(?i)^lettuce-core-` + versionPattern + `\.jar$`)},
	{Name: "CXF", Category: "Web Service", Regex: compile(`(?i)^cxf-` + versionPattern + `\.jar$`)},
	{Name: "Axis2", Category: "Web Service", Regex: compile(`(?i)^axis2-` + versionPattern + `\.jar$`)},
	{Name: "Swagger", Category: "API Doc", Regex: compile(`(?i)^swagger-` + versionPattern + `\.jar$`)},
	{Name: "Knife4j", Category: "API Doc", Regex: compile(`(?i)^knife4j-` + versionPattern + `\.jar$`)},

	// Testing
	{Name: "JUnit", Category: "Testing", Regex: compile(`(?i)^junit-` + versionPattern + `\.jar$`)},
	{Name: "TestNG", Category: "Testing", Regex: compile(`(?i)^testng-` + versionPattern + `\.jar$`)},
}

func compile(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}
