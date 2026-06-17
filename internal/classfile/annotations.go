package classfile

import "fmt"

// annotReader wraps byte data for annotation parsing.
type annotReader struct {
	data []byte
	pos  int
	cp   *ConstantPool
	err  error
}

func (r *annotReader) u1() uint8 {
	if r.err != nil || r.pos >= len(r.data) {
		if r.err == nil {
			r.err = fmt.Errorf("unexpected EOF in annotation at %d", r.pos)
		}
		return 0
	}
	v := r.data[r.pos]
	r.pos++
	return v
}

func (r *annotReader) u2() uint16 {
	if r.err != nil {
		return 0
	}
	if r.pos+1 >= len(r.data) {
		r.err = fmt.Errorf("unexpected EOF in annotation at %d", r.pos)
		return 0
	}
	v := uint16(r.data[r.pos])<<8 | uint16(r.data[r.pos+1])
	r.pos += 2
	return v
}

// ParseAnnotations parses RuntimeVisibleAnnotations attribute data.
// The data should be the raw bytes of the attribute (after name_index and length).
func ParseAnnotations(data []byte, cp *ConstantPool) ([]ParsedAnnotation, error) {
	r := &annotReader{data: data, cp: cp}
	numAnnotations := r.u2()
	annotations := make([]ParsedAnnotation, 0, numAnnotations)
	for i := uint16(0); i < numAnnotations; i++ {
		annotations = append(annotations, r.parseAnnotation())
	}
	if r.err != nil {
		return nil, r.err
	}
	return annotations, nil
}

// ParseParameterAnnotations parses RuntimeVisibleParameterAnnotations attribute data.
func ParseParameterAnnotations(data []byte, cp *ConstantPool) ([][]ParsedAnnotation, error) {
	r := &annotReader{data: data, cp: cp}
	numParams := r.u1()
	paramAnnotations := make([][]ParsedAnnotation, numParams)
	for i := uint8(0); i < numParams; i++ {
		numAnnotations := r.u2()
		annotations := make([]ParsedAnnotation, 0, numAnnotations)
		for j := uint16(0); j < numAnnotations; j++ {
			annotations = append(annotations, r.parseAnnotation())
		}
		paramAnnotations[i] = annotations
	}
	if r.err != nil {
		return nil, r.err
	}
	return paramAnnotations, nil
}

func (r *annotReader) parseAnnotation() ParsedAnnotation {
	typeIdx := r.u2()
	a := ParsedAnnotation{
		Type: r.cp.GetUTF8(typeIdx),
	}
	numPairs := r.u2()
	a.ElementValuePairs = make([]AnnotationElement, 0, numPairs)
	for i := uint16(0); i < numPairs; i++ {
		nameIdx := r.u2()
		name := r.cp.GetUTF8(nameIdx)
		value := r.parseElementValue()
		a.ElementValuePairs = append(a.ElementValuePairs, AnnotationElement{Name: name, Value: value})
	}
	return a
}

func (r *annotReader) parseElementValue() AnnotationValue {
	tag := r.u1()
	av := AnnotationValue{Tag: tag}
	switch tag {
	case 'B', 'C', 'D', 'F', 'I', 'J', 'S', 'Z', 's':
		// Primitives and strings: a const_value index into the constant pool.
		idx := r.u2()
		av.Str = r.cp.GetUTF8(idx)
	case 'e':
		// Enum: type_name_index + const_name_index.
		typeIdx := r.u2()
		nameIdx := r.u2()
		av.EnumType = r.cp.GetUTF8(typeIdx)
		av.EnumValue = r.cp.GetUTF8(nameIdx)
	case 'c':
		// Class: the UTF8 descriptor of the class.
		idx := r.u2()
		av.Str = r.cp.GetUTF8(idx)
	case '@':
		// Nested annotation.
		nested := r.parseAnnotation()
		av.Nested = &nested
	case '[':
		// Array: num_values, then each element_value.
		count := int(r.u2())
		av.Array = make([]AnnotationValue, 0, count)
		for i := 0; i < count; i++ {
			av.Array = append(av.Array, r.parseElementValue())
		}
	}
	return av
}

// GetElement returns the value of a named element, or nil if not found.
func (a *ParsedAnnotation) GetElement(name string) *AnnotationValue {
	for i := range a.ElementValuePairs {
		if a.ElementValuePairs[i].Name == name {
			return &a.ElementValuePairs[i].Value
		}
	}
	return nil
}

// AsString returns the string value of an annotation value.
// For 's' tags, returns the string. For arrays, returns the first element's string.
func (v *AnnotationValue) AsString() string {
	switch v.Tag {
	case 's':
		return v.Str
	case '[':
		if len(v.Array) > 0 {
			return v.Array[0].AsString()
		}
	}
	return ""
}

// AsStringArray returns all string values from an array annotation value.
func (v *AnnotationValue) AsStringArray() []string {
	switch v.Tag {
	case 's':
		return []string{v.Str}
	case '[':
		result := make([]string, 0, len(v.Array))
		for _, elem := range v.Array {
			if s := elem.AsString(); s != "" {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// AsEnumArray returns all enum values from an array annotation value.
func (v *AnnotationValue) AsEnumArray() []string {
	switch v.Tag {
	case 'e':
		return []string{v.EnumValue}
	case '[':
		result := make([]string, 0, len(v.Array))
		for _, elem := range v.Array {
			if elem.Tag == 'e' {
				result = append(result, elem.EnumValue)
			}
		}
		return result
	}
	return nil
}

// AnnotationSimpleName extracts the simple name from an annotation descriptor.
// "Lorg/springframework/web/bind/annotation/RequestMapping;" → "RequestMapping"
func AnnotationSimpleName(descriptor string) string {
	if len(descriptor) < 3 || descriptor[0] != 'L' {
		return descriptor
	}
	// Remove leading 'L' and trailing ';'.
	name := descriptor[1 : len(descriptor)-1]
	// Find last '/' or '.'.
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' || name[i] == '.' {
			return name[i+1:]
		}
	}
	return name
}
