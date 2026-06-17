package classfile

import "fmt"

// Tag constants for constant pool entries.
const (
	CONSTANT_Utf8               = 1
	CONSTANT_Integer            = 3
	CONSTANT_Float              = 4
	CONSTANT_Long               = 5
	CONSTANT_Double             = 6
	CONSTANT_Class              = 7
	CONSTANT_String             = 8
	CONSTANT_Fieldref           = 9
	CONSTANT_Methodref          = 10
	CONSTANT_InterfaceMethodref = 11
	CONSTANT_NameAndType        = 12
	CONSTANT_MethodHandle       = 15
	CONSTANT_MethodType         = 16
	CONSTANT_Dynamic            = 17
	CONSTANT_InvokeDynamic      = 18
	CONSTANT_Module             = 19
	CONSTANT_Package            = 20
)

// GetUTF8 resolves a constant pool index to its UTF8 string.
func (cp *ConstantPool) GetUTF8(index uint16) string {
	if index == 0 || int(index) >= len(cp.entries) {
		return ""
	}
	e := &cp.entries[index]
	switch e.Tag {
	case CONSTANT_Utf8:
		return e.Utf8
	case CONSTANT_Class:
		// Class entry points to a Utf8 entry with the class name.
		if e.Utf8 != "" {
			return e.Utf8
		}
		name := cp.resolveClass(index)
		e.Utf8 = name
		return name
	case CONSTANT_String:
		// String entry points to a Utf8 entry.
		return cp.resolveString(index)
	case CONSTANT_Fieldref, CONSTANT_Methodref, CONSTANT_InterfaceMethodref:
		// Returns the name part of the referenced member.
		return cp.resolveMemberName(index)
	case CONSTANT_NameAndType:
		return cp.resolveNameAndType(index, true)
	}
	return ""
}

// GetClassName resolves a CONSTANT_Class index to the class name (slash-separated).
func (cp *ConstantPool) GetClassName(index uint16) string {
	if index == 0 || int(index) >= len(cp.entries) {
		return ""
	}
	e := &cp.entries[index]
	if e.Tag == CONSTANT_Class {
		return cp.resolveClass(index)
	}
	return ""
}

// resolveClass resolves a CONSTANT_Class entry to its name.
func (cp *ConstantPool) resolveClass(idx uint16) string {
	if int(idx) >= len(cp.entries) {
		return ""
	}
	e := &cp.entries[idx]
	utf8Idx := e.classIndex
	if utf8Idx == 0 {
		utf8Idx = e.nameTypeIdx // some entries store it differently
	}
	return cp.rawUTF8(utf8Idx)
}

// resolveString resolves a CONSTANT_String entry.
func (cp *ConstantPool) resolveString(idx uint16) string {
	if int(idx) >= len(cp.entries) {
		return ""
	}
	e := &cp.entries[idx]
	return cp.rawUTF8(e.stringIdx)
}

// resolveMemberName resolves the name part of a field/method ref.
func (cp *ConstantPool) resolveMemberName(idx uint16) string {
	if int(idx) >= len(cp.entries) {
		return ""
	}
	e := &cp.entries[idx]
	// nameTypeIdx points to a NameAndType entry.
	return cp.resolveNameAndType(e.nameTypeIdx, true)
}

// resolveMemberDescriptor resolves the descriptor part of a field/method ref.
func (cp *ConstantPool) resolveMemberDescriptor(idx uint16) string {
	if int(idx) >= len(cp.entries) {
		return ""
	}
	e := &cp.entries[idx]
	return cp.resolveNameAndType(e.nameTypeIdx, false)
}

// resolveNameAndType resolves a NameAndType entry.
func (cp *ConstantPool) resolveNameAndType(idx uint16, getName bool) string {
	if int(idx) >= len(cp.entries) {
		return ""
	}
	e := &cp.entries[idx]
	if getName {
		return cp.rawUTF8(e.classIndex) // name_index
	}
	return cp.rawUTF8(e.nameTypeIdx) // descriptor_index
}

// rawUTF8 reads a UTF8 string from the raw CP data at the given index.
func (cp *ConstantPool) rawUTF8(idx uint16) string {
	if idx == 0 || int(idx) >= len(cp.entries) {
		return ""
	}
	e := &cp.entries[idx]
	if e.Tag == CONSTANT_Utf8 {
		return e.Utf8
	}
	return ""
}

// ResolveRef resolves a Methodref/Fieldref/InterfaceMethodref index into its components.
func (cp *ConstantPool) ResolveRef(idx uint16) (className, name, descriptor string) {
	if int(idx) >= len(cp.entries) {
		return
	}
	e := &cp.entries[idx]
	switch e.Tag {
	case CONSTANT_Methodref, CONSTANT_Fieldref, CONSTANT_InterfaceMethodref:
		className = cp.GetClassName(e.classIndex)
		name = cp.resolveNameAndType(e.nameTypeIdx, true)
		descriptor = cp.resolveNameAndType(e.nameTypeIdx, false)
	}
	return
}

// parseConstantPool reads the constant pool from a class file reader.
func parseConstantPool(r *reader) (*ConstantPool, error) {
	count := int(r.u2())
	cp := &ConstantPool{
		entries: make([]CPEntry, count),
	}

	for i := 1; i < count; i++ {
		tag := r.u1()
		cp.entries[i].Tag = tag
		switch tag {
		case CONSTANT_Utf8:
			length := int(r.u2())
			cp.entries[i].Utf8 = string(r.bytes(length))
		case CONSTANT_Integer:
			r.skip(4)
		case CONSTANT_Float:
			r.skip(4)
		case CONSTANT_Long:
			r.skip(8)
			i++ // Long takes two slots
		case CONSTANT_Double:
			r.skip(8)
			i++ // Double takes two slots
		case CONSTANT_Class:
			cp.entries[i].classIndex = r.u2()
		case CONSTANT_String:
			cp.entries[i].stringIdx = r.u2()
		case CONSTANT_Fieldref:
			cp.entries[i].classIndex = r.u2()
			cp.entries[i].nameTypeIdx = r.u2()
		case CONSTANT_Methodref:
			cp.entries[i].classIndex = r.u2()
			cp.entries[i].nameTypeIdx = r.u2()
		case CONSTANT_InterfaceMethodref:
			cp.entries[i].classIndex = r.u2()
			cp.entries[i].nameTypeIdx = r.u2()
		case CONSTANT_NameAndType:
			cp.entries[i].classIndex = r.u2()  // name_index
			cp.entries[i].nameTypeIdx = r.u2() // descriptor_index
		case CONSTANT_MethodHandle:
			cp.entries[i].refKind = r.u1()
			cp.entries[i].refIdx = r.u2()
		case CONSTANT_MethodType:
			cp.entries[i].classIndex = 0
			r.skip(2) // descriptor_index
		case CONSTANT_Dynamic:
			r.skip(4) // bootstrap_method_attr_index + name_and_type_index
		case CONSTANT_InvokeDynamic:
			r.skip(4) // bootstrap_method_attr_index + name_and_type_index
		case CONSTANT_Module:
			r.skip(2) // name_index
		case CONSTANT_Package:
			r.skip(2) // name_index
		default:
			return nil, fmt.Errorf("unknown constant pool tag: %d at index %d", tag, i)
		}
		if r.err != nil {
			return nil, r.err
		}
	}
	return cp, nil
}
