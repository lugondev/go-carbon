package gen

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/codegen"
)

// FormatDocs formats documentation comments for code generation.
// It handles multi-line docs and ensures proper Go comment formatting.
func FormatDocs(docs []string) []string {
	if len(docs) == 0 {
		return nil
	}

	formatted := make([]string, 0, len(docs))
	for _, doc := range docs {
		// Trim whitespace
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		// Ensure it starts with proper comment format
		if !strings.HasPrefix(doc, "//") {
			doc = "// " + doc
		}
		formatted = append(formatted, doc)
	}
	return formatted
}

// AddDocs adds documentation comments to a Jennifer statement.
func AddDocs(stmt *jen.Statement, docs []string) *jen.Statement {
	formattedDocs := FormatDocs(docs)
	for _, doc := range formattedDocs {
		stmt.Comment(doc)
	}
	return stmt
}

// DiscriminatorToBytes converts discriminator byte array to Go byte slice literal.
func DiscriminatorToBytes(disc []byte) *jen.Statement {
	if len(disc) == 0 {
		return jen.Nil()
	}

	// Create byte slice literal: []byte{0x01, 0x02, ...}
	values := make([]jen.Code, len(disc))
	for i, b := range disc {
		values[i] = jen.Lit(b)
	}
	return jen.Index().Byte().Values(values...)
}

// DiscriminatorToHex converts discriminator to hex string.
func DiscriminatorToHex(disc []byte) string {
	if len(disc) == 0 {
		return ""
	}
	return hex.EncodeToString(disc)
}

// FormatConstantName formats a constant name to Go convention.
// Examples:
//   - "MAX_SIZE" -> "MaxSize"
//   - "default_value" -> "DefaultValue"
func FormatConstantName(name string) string {
	return ToPascalCase(name)
}

// FormatFieldName formats a struct field name to Go convention.
// Always exported (PascalCase).
func FormatFieldName(name string) string {
	return ToPascalCase(name)
}

// FormatVariableName formats a variable name to Go convention (camelCase).
func FormatVariableName(name string) string {
	return ToCamelCase(name)
}

// FormatTypeName formats a type name to Go convention (PascalCase).
func FormatTypeName(name string) string {
	return ToPascalCase(name)
}

// FormatFunctionName formats a function name to Go convention (PascalCase for exported).
func FormatFunctionName(name string) string {
	return ToPascalCase(name)
}

// IsOptionalType checks if an IDL type is an Option type.
func IsOptionalType(typ *codegen.IDLType) bool {
	return typ != nil && typ.Option != nil
}

// IsCOptionType checks if an IDL type is a COption type.
func IsCOptionType(typ *codegen.IDLType) bool {
	return typ != nil && typ.Coption != nil
}

// IsVecType checks if an IDL type is a Vec type.
func IsVecType(typ *codegen.IDLType) bool {
	return typ != nil && typ.Vec != nil
}

// IsArrayType checks if an IDL type is an Array type.
func IsArrayType(typ *codegen.IDLType) bool {
	return typ != nil && typ.Array != nil
}

// IsPrimitiveType checks if an IDL type is a primitive type.
func IsPrimitiveType(typ *codegen.IDLType) bool {
	return typ != nil && typ.Kind != ""
}

// IsPublicKeyType checks if an IDL type is a PublicKey.
func IsPublicKeyType(typ *codegen.IDLType) bool {
	if typ == nil {
		return false
	}
	return typ.Kind == "pubkey" || typ.Kind == "publicKey"
}

// GetInnerType gets the inner type of Option, COption, Vec, or Array.
func GetInnerType(typ *codegen.IDLType) *codegen.IDLType {
	if typ == nil {
		return nil
	}
	if typ.Option != nil {
		return typ.Option
	}
	if typ.Coption != nil {
		return typ.Coption
	}
	if typ.Vec != nil {
		return typ.Vec
	}
	if typ.Array != nil {
		return &typ.Array.Type
	}
	return typ
}

// GenerateDiscriminatorConstant generates a discriminator constant.
func GenerateDiscriminatorConstant(name string, disc []byte) *jen.Statement {
	constName := FormatConstantName(name) + "Discriminator"
	return jen.Var().Id(constName).Op("=").Add(DiscriminatorToBytes(disc))
}

// GenerateProgramIDConstant generates a ProgramID constant.
func GenerateProgramIDConstant(address string) (*jen.Statement, error) {
	// Validate address
	pubkey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("invalid program address: %w", err)
	}

	return jen.Var().Id("ProgramID").Op("=").Qual(
		"github.com/gagliardetto/solana-go",
		"MustPublicKeyFromBase58",
	).Call(jen.Lit(pubkey.String())), nil
}

// GenerateFieldTag generates struct field tags for JSON and Borsh.
func GenerateFieldTag(fieldName string) map[string]string {
	tags := make(map[string]string)
	tags["json"] = fieldName
	tags["borsh"] = fieldName
	return tags
}

// GenerateStructField generates a struct field with proper tags.
func GenerateStructField(field codegen.IDLField, gen *Generator) *jen.Statement {
	fieldName := FormatFieldName(field.Name)
	fieldType := gen.ResolveType(&field.Type)
	tags := GenerateFieldTag(field.Name)

	stmt := jen.Id(fieldName).Add(fieldType).Tag(tags)

	// Add docs if present
	if len(field.Docs) > 0 {
		return AddDocs(stmt, field.Docs)
	}

	return stmt
}

// NeedsPointerForOptional checks if a type needs pointer wrapping for Option.
func NeedsPointerForOptional(typ *codegen.IDLType) bool {
	if typ == nil {
		return false
	}

	// Primitive types and PublicKey need pointers
	if IsPrimitiveType(typ) || IsPublicKeyType(typ) {
		return true
	}

	// Defined types need pointers
	if typ.Defined != nil {
		return true
	}

	// Slices and arrays already handle nil
	if IsVecType(typ) || IsArrayType(typ) {
		return false
	}

	return true
}

// GenerateImports generates common imports for generated files.
func GenerateImports(file *jen.File, needsBinary, needsSolana bool) {
	if needsBinary {
		file.ImportName("github.com/gagliardetto/binary", "bin")
	}
	if needsSolana {
		file.ImportName("github.com/gagliardetto/solana-go", "")
	}
}

// IsStructType checks if an IDL type is a struct type.
func IsStructType(typ *codegen.IDLType) bool {
	return typ != nil && typ.Struct != nil
}

// IsEnumType checks if an IDL type is an enum type.
func IsEnumType(typ *codegen.IDLType) bool {
	return typ != nil && typ.Enum != nil
}

// IsDefinedType checks if an IDL type is a defined (custom) type.
func IsDefinedType(typ *codegen.IDLType) bool {
	return typ != nil && typ.Defined != nil
}

// IsTupleType checks if an IDL type is a tuple type.
func IsTupleType(typ *codegen.IDLType) bool {
	return typ != nil && len(typ.Tuple) > 0
}

// GetTypeSize returns the size in bytes of a type if it's fixed-size.
// Returns -1 if the type is variable-size.
func GetTypeSize(typ *codegen.IDLType) int {
	if typ == nil {
		return -1
	}

	// Primitive types
	switch typ.Kind {
	case "bool", "u8", "i8":
		return 1
	case "u16", "i16":
		return 2
	case "u32", "i32", "f32":
		return 4
	case "u64", "i64", "f64":
		return 8
	case "u128", "i128":
		return 16
	case "pubkey", "publicKey":
		return 32
	case "string", "bytes":
		return -1 // Variable size
	}

	// Array has fixed size
	if typ.Array != nil {
		innerSize := GetTypeSize(&typ.Array.Type)
		if innerSize > 0 {
			return innerSize * typ.Array.Len
		}
	}

	// Vec, Option, COption are variable size
	if typ.Vec != nil || typ.Option != nil || typ.Coption != nil {
		return -1
	}

	// Can't determine size for other types
	return -1
}

// ShouldGenerateBorshMethods checks if we should generate Borsh marshal/unmarshal.
func ShouldGenerateBorshMethods(typeDef *codegen.IDLTypeDef) bool {
	// Generate for all struct types
	if IsStructType(&typeDef.Type) {
		return true
	}

	// Generate for all enum types
	if IsEnumType(&typeDef.Type) {
		return true
	}

	// Don't generate for simple type aliases
	if IsPrimitiveType(&typeDef.Type) {
		return false
	}

	return true
}

// HasVariantFields checks if an enum has any variants with fields.
func HasVariantFields(enum *codegen.IDLEnumType) bool {
	if enum == nil {
		return false
	}
	for _, variant := range enum.Variants {
		if len(variant.Fields) > 0 {
			return true
		}
	}
	return false
}

// IsSimpleEnum checks if an enum is simple (no variant fields).
func IsSimpleEnum(enum *codegen.IDLEnumType) bool {
	return !HasVariantFields(enum)
}
