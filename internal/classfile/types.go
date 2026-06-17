package classfile

// CPEntry is a single resolved constant pool entry.
type CPEntry struct {
	Tag           uint8
	Utf8          string
	RefClassName  string
	RefName       string
	RefDescriptor string
	// Raw indices stored during first pass, resolved lazily.
	classIndex  uint16
	nameTypeIdx uint16
	stringIdx   uint16
	refKind     uint8
	refIdx      uint16
}

// ConstantPool holds resolved constant pool entries, indexed by 1-based index.
type ConstantPool struct {
	entries []CPEntry
	raw     []byte // Raw CP data for lazy resolution
}

// ClassFile represents a parsed Java .class file.
type ClassFile struct {
	MinorVersion   uint16
	MajorVersion   uint16 // 52=Java8, 55=Java11, 61=Java17, 65=Java21
	ConstantPool   *ConstantPool
	AccessFlags    uint16
	ThisClass      string // Slash-separated: "com/example/MyClass"
	SuperClass     string
	InterfaceNames []string
	Fields         []FieldInfo
	Methods        []MethodInfo
	Attributes     []AttributeInfo // Class-level attributes
	ArchiveName    string
	FilePath       string // Path within archive
}

// FieldInfo represents a Java class field.
type FieldInfo struct {
	AccessFlags uint16
	Name        string
	Descriptor  string
	Attributes  []AttributeInfo
}

// MethodInfo represents a Java class method.
type MethodInfo struct {
	AccessFlags uint16
	Name        string
	Descriptor  string
	ReturnType  string
	ParamTypes  []string
	Attributes  []AttributeInfo
}

// AttributeInfo represents a generic attribute in a class file.
type AttributeInfo struct {
	Name string
	Data []byte
}

// ParsedAnnotation represents a parsed Java annotation.
type ParsedAnnotation struct {
	Type              string // "Lorg/springframework/web/bind/annotation/RequestMapping;"
	ElementValuePairs []AnnotationElement
}

// AnnotationElement is a name-value pair within an annotation.
type AnnotationElement struct {
	Name  string
	Value AnnotationValue
}

// AnnotationValue is a union type for annotation element values.
type AnnotationValue struct {
	Tag       uint8              // 's','e','c','@','[','B','C','D','F','I','J','S','Z'
	Str       string             // For 's' tag
	EnumType  string             // For 'e' tag
	EnumValue string             // For 'e' tag
	Array     []AnnotationValue  // For '[' tag
	Nested    *ParsedAnnotation  // For '@' tag
}
