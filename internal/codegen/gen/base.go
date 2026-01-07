// Package gen provides modular code generation using Jennifer.
package gen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/dave/jennifer/jen"
	"github.com/lugondev/go-carbon/internal/codegen"
)

// Generator is the base generator with shared utilities.
type Generator struct {
	IDL         *codegen.IDL
	PackageName string
	File        *jen.File
}

// NewGenerator creates a new base generator.
func NewGenerator(idl *codegen.IDL, packageName string) *Generator {
	return &Generator{
		IDL:         idl,
		PackageName: packageName,
		File:        jen.NewFile(packageName),
	}
}

// ToPascalCase converts a string to PascalCase.
// Examples:
//   - "user_account" -> "UserAccount"
//   - "my_program" -> "MyProgram"
//   - "USD" -> "USD"
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Handle all uppercase (acronyms)
	if isAllUpper(s) {
		return s
	}

	// Split by underscore
	parts := strings.Split(s, "_")
	var result strings.Builder

	for _, part := range parts {
		if part == "" {
			continue
		}
		// Capitalize first letter, lowercase the rest (unless all caps)
		if isAllUpper(part) {
			result.WriteString(part)
		} else {
			result.WriteString(strings.ToUpper(string(part[0])) + strings.ToLower(part[1:]))
		}
	}

	return result.String()
}

// ToCamelCase converts a string to camelCase.
// Examples:
//   - "user_account" -> "userAccount"
//   - "my_program" -> "myProgram"
func ToCamelCase(s string) string {
	if s == "" {
		return ""
	}

	pascal := ToPascalCase(s)
	if pascal == "" {
		return ""
	}

	// Make first character lowercase
	runes := []rune(pascal)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// isAllUpper checks if a string is all uppercase.
func isAllUpper(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

// ResolveType resolves an IDL type to a Jennifer code statement.
// This handles all primitive types, arrays, options, and custom types.
func (g *Generator) ResolveType(typ *codegen.IDLType) *jen.Statement {
	if typ == nil {
		return jen.Interface()
	}

	if typ.Defined != nil {
		typeName := ToPascalCase(typ.Defined.Name)
		if len(typ.Defined.Generics) > 0 {
			return jen.Id(typeName)
		}
		return jen.Id(typeName)
	}

	if typ.Option != nil {
		inner := g.ResolveType(typ.Option)
		return jen.Op("*").Add(inner)
	}

	if typ.Coption != nil {
		inner := g.ResolveType(typ.Coption)
		return jen.Op("*").Add(inner)
	}

	if typ.Vec != nil {
		inner := g.ResolveType(typ.Vec)
		return jen.Index().Add(inner)
	}

	if typ.Array != nil {
		inner := g.ResolveType(&typ.Array.Type)
		return jen.Index(jen.Lit(typ.Array.Len)).Add(inner)
	}

	if typ.Kind != "" {
		return g.resolvePrimitiveType(typ.Kind)
	}

	// Handle Tuple (not common in Solana, but supported)
	if len(typ.Tuple) > 0 {
		// Represent as struct with fields T0, T1, T2, etc.
		fields := make([]jen.Code, len(typ.Tuple))
		for i, t := range typ.Tuple {
			fields[i] = jen.Id(fmt.Sprintf("T%d", i)).Add(g.ResolveType(&t))
		}
		return jen.Struct(fields...)
	}

	// Handle inline Struct
	if typ.Struct != nil {
		fields := make([]jen.Code, len(typ.Struct.Fields))
		for i, field := range typ.Struct.Fields {
			fieldName := ToPascalCase(field.Name)
			fieldType := g.ResolveType(&field.Type)
			fields[i] = jen.Id(fieldName).Add(fieldType)
		}
		return jen.Struct(fields...)
	}

	// Handle inline Enum
	if typ.Enum != nil {
		// Inline enums are complex, treat as interface{} for now
		return jen.Interface()
	}

	// Fallback
	return jen.Interface()
}

// resolvePrimitiveType converts IDL primitive types to Go types.
func (g *Generator) resolvePrimitiveType(kind string) *jen.Statement {
	switch kind {
	// Unsigned integers
	case "u8":
		return jen.Qual("", "uint8")
	case "u16":
		return jen.Qual("", "uint16")
	case "u32":
		return jen.Qual("", "uint32")
	case "u64":
		return jen.Qual("", "uint64")
	case "u128":
		// Go doesn't have u128, use bin.Uint128
		return jen.Qual("github.com/gagliardetto/binary", "Uint128")

	// Signed integers
	case "i8":
		return jen.Qual("", "int8")
	case "i16":
		return jen.Qual("", "int16")
	case "i32":
		return jen.Qual("", "int32")
	case "i64":
		return jen.Qual("", "int64")
	case "i128":
		// Go doesn't have i128, use bin.Int128
		return jen.Qual("github.com/gagliardetto/binary", "Int128")

	// Floats
	case "f32":
		return jen.Qual("", "float32")
	case "f64":
		return jen.Qual("", "float64")

	// Boolean
	case "bool":
		return jen.Qual("", "bool")

	// String
	case "string":
		return jen.Qual("", "string")

	// Bytes
	case "bytes":
		return jen.Index().Byte()

	// Pubkey
	case "pubkey", "publicKey":
		return jen.Qual("github.com/gagliardetto/solana-go", "PublicKey")

	// Default
	default:
		// If not recognized, treat as custom type
		return jen.Id(ToPascalCase(kind))
	}
}

// AddComment adds documentation comments to the file.
func (g *Generator) AddComment(lines ...string) {
	for _, line := range lines {
		g.File.Comment(line)
	}
}

// AddImport adds an import statement to the file.
func (g *Generator) AddImport(path, alias string) {
	if alias != "" {
		g.File.ImportAlias(path, alias)
	} else {
		g.File.ImportName(path, "")
	}
}

// WriteToFile writes the generated code to a file.
func (g *Generator) WriteToFile(filepath string) error {
	return g.File.Save(filepath)
}

// Render renders the generated code to a string.
func (g *Generator) Render() (string, error) {
	return fmt.Sprintf("%#v", g.File), nil
}
