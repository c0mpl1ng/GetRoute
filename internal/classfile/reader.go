package classfile

import (
	"encoding/binary"
	"fmt"
)

// reader wraps a byte slice with a position cursor for binary reading.
type reader struct {
	data        []byte
	pos         int
	err         error
	archiveName string
}

func newReader(data []byte, archiveName string) *reader {
	return &reader{data: data, archiveName: archiveName}
}

func (r *reader) u1() uint8 {
	if r.err != nil || r.pos >= len(r.data) {
		if r.err == nil {
			r.err = fmt.Errorf("unexpected EOF at position %d", r.pos)
		}
		return 0
	}
	v := r.data[r.pos]
	r.pos++
	return v
}

func (r *reader) u2() uint16 {
	if r.err != nil || r.pos+1 >= len(r.data) {
		return 0
	}
	v := binary.BigEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return v
}

func (r *reader) u4() uint32 {
	if r.err != nil || r.pos+3 >= len(r.data) {
		return 0
	}
	v := binary.BigEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v
}

func (r *reader) bytes(n int) []byte {
	if r.err != nil || r.pos+n > len(r.data) {
		if r.err == nil {
			r.err = fmt.Errorf("unexpected EOF at position %d, need %d bytes", r.pos, n)
		}
		return nil
	}
	v := r.data[r.pos : r.pos+n]
	r.pos += n
	return v
}

func (r *reader) skip(n int) {
	if r.err != nil {
		return
	}
	if r.pos+n > len(r.data) {
		r.err = fmt.Errorf("skip past EOF: pos=%d, skip=%d, len=%d", r.pos, n, len(r.data))
		return
	}
	r.pos += n
}

// ReadClassFile parses a complete .class file from raw bytes.
func ReadClassFile(data []byte, archiveName, filePath string) (*ClassFile, error) {
	r := newReader(data, archiveName)

	magic := r.u4()
	if magic != 0xCAFEBABE {
		return nil, fmt.Errorf("%s: invalid class file magic: 0x%X", filePath, magic)
	}

	cf := &ClassFile{
		ArchiveName: archiveName,
		FilePath:    filePath,
	}

	cf.MinorVersion = r.u2()
	cf.MajorVersion = r.u2()

	cp, err := parseConstantPool(r)
	if err != nil {
		return nil, fmt.Errorf("%s: constant pool: %w", filePath, err)
	}
	cf.ConstantPool = cp

	cf.AccessFlags = r.u2()

	cf.ThisClass = cp.GetClassName(r.u2())
	cf.SuperClass = cp.GetClassName(r.u2())

	// Interfaces
	ifaceCount := r.u2()
	cf.InterfaceNames = make([]string, ifaceCount)
	for i := uint16(0); i < ifaceCount; i++ {
		cf.InterfaceNames[i] = cp.GetClassName(r.u2())
	}

	// Fields
	cf.Fields = parseFields(r, cp)

	// Methods
	cf.Methods = parseMethods(r, cp)

	// Class attributes
	cf.Attributes = parseAttributes(r, cp)

	if r.err != nil {
		return nil, r.err
	}

	return cf, nil
}

func parseFields(r *reader, cp *ConstantPool) []FieldInfo {
	count := r.u2()
	fields := make([]FieldInfo, count)
	for i := uint16(0); i < count; i++ {
		fields[i] = FieldInfo{
			AccessFlags: r.u2(),
			Name:        cp.GetUTF8(r.u2()),
			Descriptor:  cp.GetUTF8(r.u2()),
			Attributes:  parseAttributes(r, cp),
		}
	}
	return fields
}

func parseMethods(r *reader, cp *ConstantPool) []MethodInfo {
	count := r.u2()
	methods := make([]MethodInfo, count)
	for i := uint16(0); i < count; i++ {
		accessFlags := r.u2()
		name := cp.GetUTF8(r.u2())
		descriptor := cp.GetUTF8(r.u2())
		returnType, paramTypes := parseMethodDescriptor(descriptor)
		attrs := parseAttributes(r, cp)
		methods[i] = MethodInfo{
			AccessFlags: accessFlags,
			Name:        name,
			Descriptor:  descriptor,
			ReturnType:  returnType,
			ParamTypes:  paramTypes,
			Attributes:  attrs,
		}
	}
	return methods
}

// parseMethodDescriptor extracts return type and parameter types from a JVM descriptor.
// Example: "(Ljava/lang/String;I)Ljava/util/List;" → "List", ["String", "int"]
func parseMethodDescriptor(desc string) (string, []string) {
	if len(desc) == 0 || desc[0] != '(' {
		return "", nil
	}
	// Find param/return boundary.
	closeParen := 0
	depth := 0
	for i, c := range desc {
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth == 0 {
				closeParen = i
				break
			}
		}
	}

	params := parseFieldDescriptors(desc[1:closeParen])
	returnType := parseFieldType(desc[closeParen+1:])
	return returnType, params
}

func parseFieldDescriptors(desc string) []string {
	var types []string
	i := 0
	for i < len(desc) {
		t, n := parseFieldTypeWithLen(desc[i:])
		types = append(types, t)
		i += n
	}
	return types
}

func parseFieldType(desc string) string {
	t, _ := parseFieldTypeWithLen(desc)
	return t
}

func parseFieldTypeWithLen(desc string) (string, int) {
	if len(desc) == 0 {
		return "", 0
	}
	switch desc[0] {
	case 'B':
		return "byte", 1
	case 'C':
		return "char", 1
	case 'D':
		return "double", 1
	case 'F':
		return "float", 1
	case 'I':
		return "int", 1
	case 'J':
		return "long", 1
	case 'S':
		return "short", 1
	case 'Z':
		return "boolean", 1
	case 'V':
		return "void", 1
	case 'L':
		// Find the semicolon.
		end := 1
		for end < len(desc) && desc[end] != ';' {
			end++
		}
		name := desc[1:end]
		// Extract simple name (after last /).
		lastSlash := 0
		for i, c := range name {
			if c == '/' {
				lastSlash = i + 1
			}
		}
		return name[lastSlash:], end + 1
	case '[':
		inner, n := parseFieldTypeWithLen(desc[1:])
		return inner + "[]", n + 1
	}
	return desc, len(desc)
}

// JavaVersion returns a human-readable Java version string.
func JavaVersion(major uint16) string {
	versions := map[uint16]string{
		45: "1.1", 46: "1.2", 47: "1.3", 48: "1.4",
		49: "5", 50: "6", 51: "7", 52: "8",
		53: "9", 54: "10", 55: "11", 56: "12",
		57: "13", 58: "14", 59: "15", 60: "16",
		61: "17", 62: "18", 63: "19", 64: "20",
		65: "21", 66: "22", 67: "23", 68: "24",
	}
	if v, ok := versions[major]; ok {
		return v
	}
	return fmt.Sprintf("unknown(%d)", major)
}
