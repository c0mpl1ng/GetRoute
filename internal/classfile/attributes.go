package classfile

// Attribute name constants.
const (
	AttrRuntimeVisibleAnnotations    = "RuntimeVisibleAnnotations"
	AttrRuntimeInvisibleAnnotations  = "RuntimeInvisibleAnnotations"
	AttrRuntimeVisibleParameterAnnotations = "RuntimeVisibleParameterAnnotations"
	AttrSignature                    = "Signature"
	AttrCode                         = "Code"
	AttrExceptions                   = "Exceptions"
	AttrSourceFile                   = "SourceFile"
	AttrLineNumberTable              = "LineNumberTable"
	AttrLocalVariableTable           = "LocalVariableTable"
	AttrDeprecated                   = "Deprecated"
)

// knownAnnotationAttrs are the attribute names that contain annotations.
var knownAnnotationAttrs = map[string]bool{
	AttrRuntimeVisibleAnnotations:    true,
	AttrRuntimeInvisibleAnnotations:  true,
	AttrRuntimeVisibleParameterAnnotations: true,
}

func parseAttributes(r *reader, cp *ConstantPool) []AttributeInfo {
	count := r.u2()
	attrs := make([]AttributeInfo, 0, count)
	for i := uint16(0); i < count; i++ {
		nameIdx := r.u2()
		length := int(r.u4())
		name := cp.GetUTF8(nameIdx)
		data := r.bytes(length)
		attrs = append(attrs, AttributeInfo{
			Name: name,
			Data: data,
		})
	}
	return attrs
}

// GetAnnotationAttrs filters attributes to only those containing annotations.
func GetAnnotationAttrs(attrs []AttributeInfo) []AttributeInfo {
	var result []AttributeInfo
	for _, a := range attrs {
		if knownAnnotationAttrs[a.Name] {
			result = append(result, a)
		}
	}
	return result
}

// FindAttribute finds an attribute by name.
func FindAttribute(attrs []AttributeInfo, name string) *AttributeInfo {
	for i := range attrs {
		if attrs[i].Name == name {
			return &attrs[i]
		}
	}
	return nil
}

// ClassAnnotations returns parsed class-level annotations.
// Checks RuntimeVisibleAnnotations first, then falls back to RuntimeInvisibleAnnotations.
func (cf *ClassFile) ClassAnnotations() []ParsedAnnotation {
	return parseAnnotationAttrs(cf.Attributes, cf.ConstantPool)
}

// MethodAnnotations returns parsed method-level annotations for a method.
// Checks RuntimeVisibleAnnotations first, then falls back to RuntimeInvisibleAnnotations.
func (m *MethodInfo) MethodAnnotations(cp *ConstantPool) []ParsedAnnotation {
	return parseAnnotationAttrs(m.Attributes, cp)
}

// parseAnnotationAttrs searches a list of attributes for annotation data.
// Prefers RuntimeVisibleAnnotations, falls back to RuntimeInvisibleAnnotations.
func parseAnnotationAttrs(attrs []AttributeInfo, cp *ConstantPool) []ParsedAnnotation {
	var invisible []byte
	for _, attr := range attrs {
		switch attr.Name {
		case AttrRuntimeVisibleAnnotations:
			annotations, err := ParseAnnotations(attr.Data, cp)
			if err == nil {
				return annotations
			}
		case AttrRuntimeInvisibleAnnotations:
			invisible = attr.Data
		}
	}
	if invisible != nil {
		annotations, err := ParseAnnotations(invisible, cp)
		if err == nil {
			return annotations
		}
	}
	return nil
}
