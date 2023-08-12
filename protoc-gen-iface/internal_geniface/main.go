package internal_geniface

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ref https://github.com/protocolbuffers/protobuf-go/blob/6d0a5dbd95005b70501b4cc2c5124dab07a1f4a0/cmd/protoc-gen-go/internal_gengo/main.go#L68C2-L68C2

func GenerateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	filename := file.GeneratedFilenamePrefix + ".iface.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// Code generated by protoc-gen-iface. DO NOT EDIT.")
	g.P("package ", file.GoPackageName)
	g.P()

	f := newFileInfo(file)
	for _, message := range f.allMessages {
		genMessageInterface(g, f, message)
	}

	return g
}

// fieldGoType returns the Go type used for a field.
//
// If it returns pointer=true, the struct field is a pointer to the type.
func fieldGoType(g *protogen.GeneratedFile, f *fileInfo, field *protogen.Field) (goType string, pointer bool) {
	if field.Desc.IsWeak() {
		return "struct{}", false
	}

	pointer = field.Desc.HasPresence()
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		goType = "bool"
	case protoreflect.EnumKind:
		goType = g.QualifiedGoIdent(field.Enum.GoIdent)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		goType = "int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		goType = "uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		goType = "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		goType = "uint64"
	case protoreflect.FloatKind:
		goType = "float32"
	case protoreflect.DoubleKind:
		goType = "float64"
	case protoreflect.StringKind:
		goType = "string"
	case protoreflect.BytesKind:
		goType = "[]byte"
		pointer = false // rely on nullability of slices for presence
	case protoreflect.MessageKind, protoreflect.GroupKind:
		goType = "*" + g.QualifiedGoIdent(field.Message.GoIdent)
		pointer = false // pointer captured as part of the type
	}
	switch {
	case field.Desc.IsList():
		return "[]" + goType, false
	case field.Desc.IsMap():
		keyType, _ := fieldGoType(g, f, field.Message.Fields[0])
		valType, _ := fieldGoType(g, f, field.Message.Fields[1])
		return fmt.Sprintf("map[%v]%v", keyType, valType), false
	}
	return goType, pointer
}

func genMessageInterface(g *protogen.GeneratedFile, f *fileInfo, m *messageInfo) {
	// ref https://github.com/protocolbuffers/protobuf-go/blob/6d0a5dbd95005b70501b4cc2c5124dab07a1f4a0/cmd/protoc-gen-go/internal_gengo/main.go#L547
	if m.Desc.IsMapEntry() {
		return
	}

	g.P("type ", m.GoIdent, "Iface interface {")
	genMessageInterfaceMethod(g, f, m)
	g.P("}")
	g.P()

}

func genMessageInterfaceMethod(g *protogen.GeneratedFile, f *fileInfo, m *messageInfo) {
	for _, field := range m.Fields {
		// Getter for parent oneof.
		if oneof := field.Oneof; oneof != nil && oneof.Fields[0] == field && !oneof.Desc.IsSynthetic() {
			g.P("Get", oneof.GoName, "() ", oneofInterfaceName(oneof))
		}

		// Getter for message field.
		goType, _ := fieldGoType(g, f, field)

		g.Annotate(m.GoIdent.GoName+".Get"+field.GoName, field.Location)
		switch {
		case field.Oneof != nil && !field.Oneof.Desc.IsSynthetic():
			g.P("Get", field.GoName, "() ", goType)
		default:
			g.P("Get", field.GoName, "() ", goType)
		}
	}
}

// oneofInterfaceName returns the name of the interface type implemented by
// the oneof field value types.
func oneofInterfaceName(oneof *protogen.Oneof) string {
	return "is" + oneof.GoIdent.GoName
}

// genMessageOneofWrapperTypes generates the oneof wrapper types and
// associates the types with the parent message type.
func genMessageOneofWrapperTypes(g *protogen.GeneratedFile, f *fileInfo, m *messageInfo) {
	for _, oneof := range m.Oneofs {
		if oneof.Desc.IsSynthetic() {
			continue
		}
		ifName := oneofInterfaceName(oneof)
		g.P("type ", ifName, " interface {")
		g.P(ifName, "()")
		g.P("}")
	}
}