package classfile

import (
	"regexp"
	"strings"
)

// ReadJavaFile parses a Java source file and extracts class-level information
// including annotations, package, class name, superclass, and methods with their
// annotations. Returns a ClassFile compatible with the existing extractor pipeline.
func ReadJavaFile(data []byte, archiveName, filePath string) (*ClassFile, error) {
	src := string(data)

	// Remove comments only (preserve string literals so annotation values survive).
	clean := stripComments(src)

	// Extract package.
	pkg := extractPackage(clean)

	// Parse imports: simple name → fully qualified name.
	imports := parseImports(clean)

	// Find class/interface/enum declaration block and annotations preceding it.
	classAnnotations, className, superClass, interfaces, bodyStart, bodyEnd := extractClassDecl(clean, imports)

	// Compute ThisClass in slash format.
	thisClass := className
	if pkg != "" {
		thisClass = pkg + "/" + className
	}
	thisClass = strings.ReplaceAll(thisClass, ".", "/")

	superClassName := strings.ReplaceAll(superClass, ".", "/")

	// Find all method declarations within the class body only.
	classBody := clean[bodyStart:bodyEnd]
	methods, methodAnns := extractMethods(classBody, imports)

	cf := &ClassFile{
		ThisClass:      thisClass,
		SuperClass:     superClassName,
		InterfaceNames: interfaces,
		Methods:        methods,
		ArchiveName:    archiveName,
		FilePath:       filePath,
	}

	// Build synthetic constant pool and annotation attributes.
	return newJavaSourceClassFile(cf, classAnnotations, methodAnns), nil
}

// ---------------------------------------------------------------------------
// Comment stripping (preserves string/char literal content)
// ---------------------------------------------------------------------------

// stripComments removes Java comments while preserving string and char literals.
// We track strings/chars so that comment markers (//, /*) inside them are not
// treated as real comments. The literal content is kept so annotation string
// values survive.
func stripComments(src string) string {
	var out strings.Builder
	out.Grow(len(src))

	i := 0
	n := len(src)
	for i < n {
		// String literal — keep content.
		if src[i] == '"' {
			out.WriteByte('"')
			i = copyString(src, i, &out)
			continue
		}
		// Char literal — keep content.
		if src[i] == '\'' {
			out.WriteByte('\'')
			i = copyChar(src, i, &out)
			continue
		}
		// Line comment
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			i = skipLineComment(src, i)
			out.WriteByte(' ')
			continue
		}
		// Block comment
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i = skipBlockComment(src, i)
			out.WriteByte(' ')
			continue
		}
		out.WriteByte(src[i])
		i++
	}
	return out.String()
}

// copyString copies a Java string literal from src[i] to out, returns new position.
func copyString(src string, i int, out *strings.Builder) int {
	i++ // skip opening "
	for i < len(src) {
		if src[i] == '\\' {
			out.WriteByte(src[i])
			i++
			if i < len(src) {
				out.WriteByte(src[i])
				i++
			}
			continue
		}
		if src[i] == '"' {
			out.WriteByte('"')
			return i + 1
		}
		out.WriteByte(src[i])
		i++
	}
	return i
}

// copyChar copies a Java char literal from src[i] to out, returns new position.
func copyChar(src string, i int, out *strings.Builder) int {
	i++ // skip opening '
	for i < len(src) {
		if src[i] == '\\' {
			out.WriteByte(src[i])
			i++
			if i < len(src) {
				out.WriteByte(src[i])
				i++
			}
			continue
		}
		if src[i] == '\'' {
			out.WriteByte('\'')
			return i + 1
		}
		out.WriteByte(src[i])
		i++
	}
	return i
}

func skipString(src string, i int) int {
	i++
	for i < len(src) {
		if src[i] == '\\' {
			i += 2
			continue
		}
		if src[i] == '"' {
			return i + 1
		}
		i++
	}
	return i
}

func skipChar(src string, i int) int {
	i++
	for i < len(src) {
		if src[i] == '\\' {
			i += 2
			continue
		}
		if src[i] == '\'' {
			return i + 1
		}
		i++
	}
	return i
}

func skipLineComment(src string, i int) int {
	for i < len(src) && src[i] != '\n' {
		i++
	}
	return i
}

func skipBlockComment(src string, i int) int {
	i += 2
	for i+1 < len(src) {
		if src[i] == '*' && src[i+1] == '/' {
			return i + 2
		}
		i++
	}
	return i
}

// ---------------------------------------------------------------------------
// Package extraction
// ---------------------------------------------------------------------------

var pkgRe = regexp.MustCompile(`\bpackage\s+([\w.]+)\s*;`)

func extractPackage(src string) string {
	m := pkgRe.FindStringSubmatch(src)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// ---------------------------------------------------------------------------
// Import parsing
// ---------------------------------------------------------------------------

var importRe = regexp.MustCompile(`\bimport\s+(static\s+)?([\w.*]+)\s*;`)

func parseImports(src string) map[string]string {
	imports := make(map[string]string)
	matches := importRe.FindAllStringSubmatch(src, -1)
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		fqn := m[2]
		lastDot := strings.LastIndex(fqn, ".")
		if lastDot >= 0 {
			simple := fqn[lastDot+1:]
			if simple != "*" {
				imports[simple] = fqn
			}
		}
	}
	return imports
}

// ---------------------------------------------------------------------------
// Class declaration extraction
// ---------------------------------------------------------------------------

var classDeclRe = regexp.MustCompile(
	`(?s)((?:@[\w.]+\s*(?:\([^)]*\))?\s*)*)` +
		`(?:(?:public|protected|private|abstract|static|final|strictfp)\s+)*` +
		`(class|interface|enum)\s+` +
		`(\w+)` +
		`(?:\s+extends\s+([\w.<>,?\s]+?))?` +
		`(?:\s+implements\s+([\w.<>,?\s]+?))?` +
		`\s*[{;]`,
)

func extractClassDecl(src string, imports map[string]string) ([]ParsedAnnotation, string, string, []string, int, int) {
	m := classDeclRe.FindStringSubmatchIndex(src)
	if len(m) < 10 {
		return nil, "Unknown", "java/lang/Object", nil, 0, len(src)
	}

	// FindStringSubmatchIndex indices: m[0],m[1]=full, m[2],m[3]=annotations,
	// m[4],m[5]=type, m[6],m[7]=className, m[8],m[9]=extends, m[10],m[11]=implements
	annText := strings.TrimSpace(src[m[2]:m[3]])
	className := src[m[6]:m[7]]

	superClass := "java.lang.Object"
	if m[8] >= 0 {
		superClass = strings.TrimSpace(src[m[8]:m[9]])
	}

	ifaceText := ""
	if len(m) >= 12 && m[10] >= 0 {
		ifaceText = strings.TrimSpace(src[m[10]:m[11]])
	}

	// Find class body: the regex ends with [{;], so the opening '{' may
	// already be consumed. Check the last matched character first.
	bodyStart := -1
	if m[1] > 0 && m[1] <= len(src) && src[m[1]-1] == '{' {
		bodyStart = m[1] - 1
	} else {
		bodyStart = findOpeningBrace(src, m[1])
	}
	bodyEnd := findMatchingBrace(src, bodyStart)
	if bodyStart < 0 || bodyEnd < 0 || bodyEnd <= bodyStart {
		// No body found (interface with ';', etc.) — fallback to entire source.
		bodyStart = 0
		bodyEnd = len(src)
	}

	var annotations []ParsedAnnotation
	if annText != "" {
		annotations = parseAnnotationBlock(annText, imports)
	}

	var interfaces []string
	if ifaceText != "" {
		for _, name := range strings.Split(ifaceText, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				interfaces = append(interfaces, name)
			}
		}
	}

	if idx := strings.Index(superClass, "<"); idx >= 0 {
		superClass = superClass[:idx]
	}

	return annotations, className, superClass, interfaces, bodyStart, bodyEnd
}

// ---------------------------------------------------------------------------
// Method extraction
// ---------------------------------------------------------------------------

// methodHeaderRe matches annotations, modifiers, return type, method name, and
// the opening paren of the parameter list. We then use findMatchingParen to
// locate the closing paren, which correctly handles nested parens inside
// annotation parameters (e.g. @RequestParam(required = false)).
var methodHeaderRe = regexp.MustCompile(
	`(?s)((?:@[\w.]+\s*(?:\([^)]*\))?\s*)*)` +
		`(?:(?:public|protected|private|abstract|static|final|synchronized|native|strictfp|default)\s+)*` +
		`(?:<[\w\s,?]+>\s*)?` +
		`([\w\[\]<>,.?$\s]+?)` +
		`\s+` +
		`(\w+)` +
		`\s*\(`,
)

var skipWords = map[string]bool{
	"class": true, "interface": true, "enum": true,
	"if": true, "else": true, "for": true, "while": true,
	"switch": true, "try": true, "catch": true, "finally": true,
	"return": true, "throw": true, "new": true, "package": true,
	"import": true, "super": true, "this": true, "synchronized": true,
	"volatile": true, "transient": true, "native": true, "strictfp": true,
}

func extractMethods(src string, imports map[string]string) ([]MethodInfo, map[string][]ParsedAnnotation) {
	var methods []MethodInfo
	methodAnns := make(map[string][]ParsedAnnotation)
	allMatches := methodHeaderRe.FindAllStringSubmatchIndex(src, -1)

	for _, idx := range allMatches {
		if len(idx) < 8 {
			continue
		}
		// The regex now ends at the opening paren. idx[0] = start, idx[1] = end (after the '(').
		openParenPos := idx[1] - 1 // position of '('
		annText := strings.TrimSpace(src[idx[2]:idx[3]])
		returnType := strings.TrimSpace(src[idx[4]:idx[5]])
		methodName := src[idx[6]:idx[7]]

		if skipWords[methodName] {
			continue
		}

		// Skip constructor calls: "new ClassName(" — the return type is "new".
		if returnType == "new" {
			continue
		}

		// Skip if the matched text ends with "new" (constructor in weird formatting).
		// Also check that the return type isn't a line-noise artifact.
		if isNoiseReturnType(returnType) {
			continue
		}

		// Find matching closing paren for the parameter list.
		closeParenPos := findMatchingParen(src, openParenPos)
		if closeParenPos < 0 {
			continue
		}
		paramsText := src[openParenPos+1 : closeParenPos]

		var annotations []ParsedAnnotation
		if annText != "" {
			annotations = parseAnnotationBlock(annText, imports)
		}

		returnTypeSlash := strings.ReplaceAll(returnType, "[]", "")
		returnTypeSlash = strings.TrimSpace(returnTypeSlash)
		if idx2 := strings.Index(returnTypeSlash, "<"); idx2 >= 0 {
			returnTypeSlash = returnTypeSlash[:idx2]
		}

		paramTypes := parseParamTypesFromSource(paramsText)

		methods = append(methods, MethodInfo{
			Name:       methodName,
			ReturnType: returnTypeSlash,
			ParamTypes: paramTypes,
		})

		if len(annotations) > 0 {
			methodAnns[methodName] = annotations
		}
	}

	return methods, methodAnns
}

// isNoiseReturnType returns true if the return type captured by the method regex
// looks like it came from a false-positive match (e.g., whitespace-heavy or
// containing line noise from comment stripping).
func isNoiseReturnType(rt string) bool {
	// Return types that are just whitespace/newlines are noise.
	trimmed := strings.TrimSpace(rt)
	if trimmed == "" {
		return true
	}
	// If the "return type" contains a newline, it's likely a false match
	// from cross-line noise.
	if strings.Contains(rt, "\n") {
		return true
	}
	// Single punctuation or symbol characters.
	if len(trimmed) <= 1 && !isLetter(trimmed) {
		return true
	}
	return false
}

func isLetter(s string) bool {
	if len(s) == 0 {
		return false
	}
	r := s[0]
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// findMatchingParen finds the closing ')' that matches the opening '(' at openPos.
// Handles nested parentheses, strings, and comments.
func findMatchingParen(src string, openPos int) int {
	if openPos < 0 || openPos >= len(src) || src[openPos] != '(' {
		return -1
	}
	depth := 1
	i := openPos + 1
	for i < len(src) && depth > 0 {
		switch src[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		case '"':
			i = skipString(src, i)
			continue
		case '\'':
			i = skipChar(src, i)
			continue
		case '/':
			if i+1 < len(src) && src[i+1] == '/' {
				i = skipLineComment(src, i)
				continue
			}
			if i+1 < len(src) && src[i+1] == '*' {
				i = skipBlockComment(src, i)
				continue
			}
		}
		i++
	}
	return -1
}

// findOpeningBrace finds the first '{' at or after pos, skipping whitespace, strings, and comments.
func findOpeningBrace(src string, pos int) int {
	for i := pos; i < len(src); i++ {
		switch src[i] {
		case '{':
			return i
		case '"':
			i = skipString(src, i)
		case '\'':
			i = skipChar(src, i)
		case '/':
			if i+1 < len(src) && src[i+1] == '/' {
				i = skipLineComment(src, i)
			} else if i+1 < len(src) && src[i+1] == '*' {
				i = skipBlockComment(src, i)
			}
		}
	}
	return -1
}

// findMatchingBrace finds the closing '}' that matches the opening '{' at openPos.
func findMatchingBrace(src string, openPos int) int {
	if openPos < 0 || openPos >= len(src) || src[openPos] != '{' {
		return -1
	}
	depth := 1
	i := openPos + 1
	for i < len(src) && depth > 0 {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1 // include the closing brace
			}
		case '"':
			i = skipString(src, i)
			continue
		case '\'':
			i = skipChar(src, i)
			continue
		case '/':
			if i+1 < len(src) && src[i+1] == '/' {
				i = skipLineComment(src, i)
				continue
			}
			if i+1 < len(src) && src[i+1] == '*' {
				i = skipBlockComment(src, i)
				continue
			}
		}
		i++
	}
	return -1
}

func parseParamTypesFromSource(paramsText string) []string {
	if strings.TrimSpace(paramsText) == "" {
		return nil
	}
	var types []string
	for _, param := range strings.Split(paramsText, ",") {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}
		param = stripAnnotations(param)
		parts := strings.Fields(param)
		if len(parts) >= 1 {
			typeName := parts[0]
			if typeName == "final" && len(parts) >= 2 {
				typeName = parts[1]
			}
			if idx := strings.Index(typeName, "<"); idx >= 0 {
				typeName = typeName[:idx]
			}
			typeName = strings.TrimSuffix(typeName, "...")
			types = append(types, typeName)
		}
	}
	return types
}

func stripAnnotations(text string) string {
	for {
		text = strings.TrimSpace(text)
		if strings.HasPrefix(text, "@") {
			parenCount := 0
			searchStart := 0
			for searchStart < len(text) {
				pos := strings.IndexAny(text[searchStart:], "() ")
				if pos < 0 {
					return ""
				}
				pos += searchStart
				switch text[pos] {
				case '(':
					parenCount++
				case ')':
					parenCount--
				case ' ':
					if parenCount == 0 {
						text = text[pos:]
						goto next
					}
				}
				searchStart = pos + 1
			}
			return ""
		next:
		} else {
			return text
		}
	}
}

// ---------------------------------------------------------------------------
// Annotation parsing from source text
// ---------------------------------------------------------------------------

func parseAnnotationBlock(annText string, imports map[string]string) []ParsedAnnotation {
	var annotations []ParsedAnnotation

	annMarkerRe := regexp.MustCompile(`@([\w.]+)\s*(\([^)]*\))?`)
	matches := annMarkerRe.FindAllStringSubmatch(annText, -1)

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		annName := m[1]
		annParams := ""
		if len(m) >= 3 {
			annParams = m[2]
		}

		desc := resolveAnnotationDesc(annName, imports)
		if desc == "" {
			continue
		}

		ann := ParsedAnnotation{
			Type:              desc,
			ElementValuePairs: parseAnnotationParams(annParams),
		}
		annotations = append(annotations, ann)
	}

	return annotations
}

func resolveAnnotationDesc(name string, imports map[string]string) string {
	name = strings.TrimPrefix(name, "@")

	if strings.Contains(name, ".") {
		return "L" + strings.ReplaceAll(name, ".", "/") + ";"
	}

	if fqn, ok := imports[name]; ok {
		return "L" + strings.ReplaceAll(fqn, ".", "/") + ";"
	}

	commonAnnotations := map[string]string{
		"Controller":              "Lorg/springframework/stereotype/Controller;",
		"RestController":          "Lorg/springframework/web/bind/annotation/RestController;",
		"RequestMapping":          "Lorg/springframework/web/bind/annotation/RequestMapping;",
		"GetMapping":              "Lorg/springframework/web/bind/annotation/GetMapping;",
		"PostMapping":             "Lorg/springframework/web/bind/annotation/PostMapping;",
		"PutMapping":              "Lorg/springframework/web/bind/annotation/PutMapping;",
		"DeleteMapping":           "Lorg/springframework/web/bind/annotation/DeleteMapping;",
		"PatchMapping":            "Lorg/springframework/web/bind/annotation/PatchMapping;",
		"ResponseBody":            "Lorg/springframework/web/bind/annotation/ResponseBody;",
		"RequestBody":             "Lorg/springframework/web/bind/annotation/RequestBody;",
		"RequestParam":            "Lorg/springframework/web/bind/annotation/RequestParam;",
		"PathVariable":            "Lorg/springframework/web/bind/annotation/PathVariable;",
		"Service":                 "Lorg/springframework/stereotype/Service;",
		"Repository":              "Lorg/springframework/stereotype/Repository;",
		"Component":               "Lorg/springframework/stereotype/Component;",
		"Configuration":           "Lorg/springframework/context/annotation/Configuration;",
		"Bean":                    "Lorg/springframework/context/annotation/Bean;",
		"Autowired":               "Lorg/springframework/beans/factory/annotation/Autowired;",
		"Qualifier":               "Lorg/springframework/beans/factory/annotation/Qualifier;",
		"Value":                   "Lorg/springframework/beans/factory/annotation/Value;",
		"WebServlet":              "Ljavax/servlet/annotation/WebServlet;",
		"WebFilter":               "Ljavax/servlet/annotation/WebFilter;",
		"WebListener":             "Ljavax/servlet/annotation/WebListener;",
		"Path":                    "Ljavax/ws/rs/Path;",
		"GET":                     "Ljavax/ws/rs/GET;",
		"POST":                    "Ljavax/ws/rs/POST;",
		"PUT":                     "Ljavax/ws/rs/PUT;",
		"DELETE":                  "Ljavax/ws/rs/DELETE;",
		"PATCH":                   "Ljavax/ws/rs/PATCH;",
		"HEAD":                    "Ljavax/ws/rs/HEAD;",
		"OPTIONS":                 "Ljavax/ws/rs/OPTIONS;",
		"Produces":                "Ljavax/ws/rs/Produces;",
		"Consumes":                "Ljavax/ws/rs/Consumes;",
		"ApplicationPath":         "Ljavax/ws/rs/ApplicationPath;",
		"Action":                  "Lorg/apache/struts2/convention/annotation/Action;",
		"Actions":                 "Lorg/apache/struts2/convention/annotation/Actions;",
		"Namespace":               "Lorg/apache/struts2/convention/annotation/Namespace;",
		"Override":                "Ljava/lang/Override;",
		"Deprecated":              "Ljava/lang/Deprecated;",
		"SuppressWarnings":        "Ljava/lang/SuppressWarnings;",
		"FunctionalInterface":     "Ljava/lang/FunctionalInterface;",
		"SafeVarargs":             "Ljava/lang/SafeVarargs;",
	}

	if desc, ok := commonAnnotations[name]; ok {
		return desc
	}

	return ""
}

func parseAnnotationParams(params string) []AnnotationElement {
	params = strings.TrimSpace(params)
	if params == "" {
		return nil
	}

	if strings.HasPrefix(params, "(") && strings.HasSuffix(params, ")") {
		params = params[1 : len(params)-1]
	}
	params = strings.TrimSpace(params)
	if params == "" {
		return nil
	}

	var elements []AnnotationElement

	if !containsTopLevel(params, '=') {
		val := parseAnnotationValue(params)
		elements = append(elements, AnnotationElement{Name: "value", Value: val})
		return elements
	}

	pairs := splitTopLevel(params, ',')
	for _, pair := range pairs {
		eqIdx := strings.Index(pair, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:eqIdx])
		valText := strings.TrimSpace(pair[eqIdx+1:])
		val := parseAnnotationValue(valText)
		elements = append(elements, AnnotationElement{Name: key, Value: val})
	}

	return elements
}

func parseAnnotationValue(text string) AnnotationValue {
	text = strings.TrimSpace(text)

	if text == "" {
		return AnnotationValue{Tag: 's', Str: ""}
	}

	// Array: {val1, val2, val3}
	if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
		inner := text[1 : len(text)-1]
		parts := splitTopLevel(inner, ',')
		var arr []AnnotationValue
		for _, p := range parts {
			arr = append(arr, parseAnnotationValue(p))
		}
		return AnnotationValue{Tag: '[', Array: arr}
	}

	// String literal.
	if strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"") {
		s := text[1 : len(text)-1]
		return AnnotationValue{Tag: 's', Str: s}
	}

	// Enum value: RequestMethod.GET
	if strings.Contains(text, ".") {
		lastDot := strings.LastIndex(text, ".")
		enumType := text[:lastDot]
		enumValue := text[lastDot+1:]
		return AnnotationValue{
			Tag:       'e',
			EnumType:  "L" + strings.ReplaceAll(enumType, ".", "/") + ";",
			EnumValue: enumValue,
		}
	}

	// Class literal: MyClass.class
	if strings.HasSuffix(text, ".class") {
		className := text[:len(text)-6]
		return AnnotationValue{Tag: 'c', Str: className}
	}

	// Default: treat as string.
	return AnnotationValue{Tag: 's', Str: text}
}

func containsTopLevel(s string, ch byte) bool {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '{':
			depth++
		case ')', '}':
			depth--
		case ch:
			if depth == 0 {
				return true
			}
		}
	}
	return false
}

func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth := 0
	inString := false
	lastStart := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			if !inString {
				inString = true
			} else if i > 0 && s[i-1] != '\\' {
				inString = false
			}
		case '(', '{':
			if !inString {
				depth++
			}
		case ')', '}':
			if !inString {
				depth--
			}
		case sep:
			if depth == 0 && !inString {
				part := strings.TrimSpace(s[lastStart:i])
				if part != "" {
					parts = append(parts, part)
				}
				lastStart = i + 1
			}
		}
	}
	part := strings.TrimSpace(s[lastStart:])
	if part != "" {
		parts = append(parts, part)
	}

	return parts
}

// ---------------------------------------------------------------------------
// Synthetic constant pool and binary annotation encoding
// ---------------------------------------------------------------------------

// newJavaSourceClassFile builds a ClassFile from Java source parsing results
// with a synthetic constant pool so that ClassAnnotations() and MethodAnnotations()
// work with the existing ParseAnnotations infrastructure.
func newJavaSourceClassFile(cf *ClassFile, classAnns []ParsedAnnotation, methodAnns map[string][]ParsedAnnotation) *ClassFile {
	cp := &ConstantPool{
		entries: make([]CPEntry, 1), // 1-based, index 0 unused
	}
	strIdx := make(map[string]int)

	addUTF8 := func(s string) int {
		if idx, ok := strIdx[s]; ok {
			return idx
		}
		idx := len(cp.entries)
		cp.entries = append(cp.entries, CPEntry{Tag: CONSTANT_Utf8, Utf8: s})
		strIdx[s] = idx
		return idx
	}

	// Attach class annotations.
	if len(classAnns) > 0 {
		cf.Attributes = append(cf.Attributes, AttributeInfo{
			Name: AttrRuntimeVisibleAnnotations,
			Data: buildAnnotationAttributeData(classAnns, addUTF8),
		})
	}

	// Attach method annotations.
	for i := range cf.Methods {
		methodName := cf.Methods[i].Name
		if anns, ok := methodAnns[methodName]; ok && len(anns) > 0 {
			cf.Methods[i].Attributes = append(cf.Methods[i].Attributes, AttributeInfo{
				Name: AttrRuntimeVisibleAnnotations,
				Data: buildAnnotationAttributeData(anns, addUTF8),
			})
		}
	}

	cf.ConstantPool = cp
	return cf
}

func buildAnnotationAttributeData(annotations []ParsedAnnotation, addUTF8 func(string) int) []byte {
	var buf []byte

	// u2 num_annotations
	buf = append(buf, byte(len(annotations)>>8), byte(len(annotations)&0xFF))

	for _, ann := range annotations {
		buf = append(buf, encodeAnnotation(ann, addUTF8)...)
	}

	return buf
}

// encodeAnnotation encodes a single annotation (type_index + num_pairs + pairs).
func encodeAnnotation(ann ParsedAnnotation, addUTF8 func(string) int) []byte {
	var buf []byte

	// u2 type_index
	typeIdx := addUTF8(ann.Type)
	buf = append(buf, byte(typeIdx>>8), byte(typeIdx&0xFF))

	// u2 num_element_value_pairs
	buf = append(buf, byte(len(ann.ElementValuePairs)>>8), byte(len(ann.ElementValuePairs)&0xFF))

	for _, pair := range ann.ElementValuePairs {
		// u2 element_name_index
		nameIdx := addUTF8(pair.Name)
		buf = append(buf, byte(nameIdx>>8), byte(nameIdx&0xFF))

		// element_value
		buf = append(buf, encodeElementValue(pair.Value, addUTF8)...)
	}

	return buf
}

func encodeElementValue(v AnnotationValue, addUTF8 func(string) int) []byte {
	var buf []byte
	buf = append(buf, v.Tag)

	switch v.Tag {
	case 'B', 'C', 'D', 'F', 'I', 'J', 'S', 'Z', 's':
		idx := addUTF8(v.Str)
		buf = append(buf, byte(idx>>8), byte(idx&0xFF))
	case 'e':
		typeIdx := addUTF8(v.EnumType)
		buf = append(buf, byte(typeIdx>>8), byte(typeIdx&0xFF))
		nameIdx := addUTF8(v.EnumValue)
		buf = append(buf, byte(nameIdx>>8), byte(nameIdx&0xFF))
	case 'c':
		idx := addUTF8(v.Str)
		buf = append(buf, byte(idx>>8), byte(idx&0xFF))
	case '@':
		// Nested annotation — encode just the annotation, without num_annotations prefix.
		nestedData := encodeAnnotation(*v.Nested, addUTF8)
		buf = append(buf, nestedData...)
	case '[':
		buf = append(buf, byte(len(v.Array)>>8), byte(len(v.Array)&0xFF))
		for _, elem := range v.Array {
			buf = append(buf, encodeElementValue(elem, addUTF8)...)
		}
	}

	return buf
}
