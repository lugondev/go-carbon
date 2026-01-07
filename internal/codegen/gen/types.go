package gen

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/lugondev/go-carbon/internal/codegen"
)

// TypesGenerator generates custom type definitions (structs and enums).
type TypesGenerator struct {
	*Generator
}

// NewTypesGenerator creates a new types generator.
func NewTypesGenerator(gen *Generator) *TypesGenerator {
	return &TypesGenerator{Generator: gen}
}

// Generate generates all custom types from the IDL.
func (g *TypesGenerator) Generate() error {
	if len(g.IDL.Types) == 0 {
		return nil
	}

	// Generate each type definition
	for _, typeDef := range g.IDL.Types {
		if err := g.generateType(typeDef); err != nil {
			return fmt.Errorf("failed to generate type %s: %w", typeDef.Name, err)
		}
	}

	return nil
}

// generateType generates a single type definition.
func (g *TypesGenerator) generateType(typeDef codegen.IDLTypeDef) error {
	// Add documentation
	if len(typeDef.Docs) > 0 {
		for _, doc := range FormatDocs(typeDef.Docs) {
			g.File.Comment(doc)
		}
	}

	// Handle different type kinds
	if typeDef.Type.Struct != nil {
		return g.generateStruct(typeDef)
	} else if typeDef.Type.Enum != nil {
		return g.generateEnum(typeDef)
	} else {
		// Type alias (e.g., type MyU64 = u64)
		return g.generateTypeAlias(typeDef)
	}
}

// generateStruct generates a struct type definition.
func (g *TypesGenerator) generateStruct(typeDef codegen.IDLTypeDef) error {
	typeName := FormatTypeName(typeDef.Name)

	// Generate struct fields
	fields := make([]jen.Code, 0, len(typeDef.Type.Struct.Fields))
	for _, field := range typeDef.Type.Struct.Fields {
		fieldStmt := GenerateStructField(field, g.Generator)
		fields = append(fields, fieldStmt)
	}

	// Generate the struct
	g.File.Type().Id(typeName).Struct(fields...)

	return nil
}

// generateEnum generates an enum type definition.
// Solana/Anchor enums can be:
// 1. Simple enums (no fields) - represented as uint8 constants
// 2. Complex enums (with fields) - represented as interface + variant structs
func (g *TypesGenerator) generateEnum(typeDef codegen.IDLTypeDef) error {
	if IsSimpleEnum(typeDef.Type.Enum) {
		return g.generateSimpleEnum(typeDef)
	}
	return g.generateComplexEnum(typeDef)
}

// generateSimpleEnum generates a simple enum (no variant fields).
// Example:
//
//	type Status uint8
//	const (
//	    StatusPending Status = 0
//	    StatusActive  Status = 1
//	    StatusClosed  Status = 2
//	)
func (g *TypesGenerator) generateSimpleEnum(typeDef codegen.IDLTypeDef) error {
	typeName := FormatTypeName(typeDef.Name)

	// Generate enum type as uint8
	g.File.Type().Id(typeName).Uint8()
	g.File.Line()

	// Generate enum constants
	constValues := make([]jen.Code, 0, len(typeDef.Type.Enum.Variants))
	for i, variant := range typeDef.Type.Enum.Variants {
		variantName := FormatVariantName(typeName, variant.Name)
		constValues = append(constValues, jen.Id(variantName).Id(typeName).Op("=").Lit(i))
	}

	g.File.Const().Defs(constValues...)
	g.File.Line()

	// Generate String() method
	g.generateSimpleEnumStringMethod(typeName, typeDef.Type.Enum.Variants)

	return nil
}

// generateComplexEnum generates a complex enum (with variant fields).
// Example:
//
//	type Action interface {
//	    isAction()
//	}
//
//	type ActionTransfer struct {
//	    Amount uint64
//	    To     solana.PublicKey
//	}
//	func (*ActionTransfer) isAction() {}
//
//	type ActionClose struct{}
//	func (*ActionClose) isAction() {}
func (g *TypesGenerator) generateComplexEnum(typeDef codegen.IDLTypeDef) error {
	typeName := FormatTypeName(typeDef.Name)
	interfaceMethodName := fmt.Sprintf("is%s", typeName)

	// Generate interface
	g.File.Type().Id(typeName).Interface(
		jen.Id(interfaceMethodName).Params(),
	)
	g.File.Line()

	// Generate variant structs
	for i, variant := range typeDef.Type.Enum.Variants {
		if err := g.generateEnumVariant(typeName, variant, i, interfaceMethodName); err != nil {
			return fmt.Errorf("failed to generate variant %s: %w", variant.Name, err)
		}
	}

	return nil
}

// generateEnumVariant generates a single enum variant struct.
func (g *TypesGenerator) generateEnumVariant(enumName string, variant codegen.IDLEnumVariant, index int, interfaceMethod string) error {
	variantTypeName := FormatVariantName(enumName, variant.Name)

	// Generate variant struct
	if len(variant.Fields) > 0 {
		// Variant with fields
		fields := make([]jen.Code, 0, len(variant.Fields))
		for _, field := range variant.Fields {
			fieldStmt := GenerateStructField(field, g.Generator)
			fields = append(fields, fieldStmt)
		}
		g.File.Type().Id(variantTypeName).Struct(fields...)
	} else {
		// Empty variant
		g.File.Type().Id(variantTypeName).Struct()
	}

	// Generate interface implementation method
	g.File.Func().Params(jen.Op("*").Id(variantTypeName)).Id(interfaceMethod).Params().Block()
	g.File.Line()

	return nil
}

// generateSimpleEnumStringMethod generates a String() method for simple enums.
func (g *TypesGenerator) generateSimpleEnumStringMethod(typeName string, variants []codegen.IDLEnumVariant) {
	cases := make([]jen.Code, 0, len(variants))
	for _, variant := range variants {
		variantName := FormatVariantName(typeName, variant.Name)
		cases = append(cases,
			jen.Case(jen.Id(variantName)).Block(
				jen.Return(jen.Lit(variant.Name)),
			),
		)
	}

	// Add default case
	cases = append(cases,
		jen.Default().Block(
			jen.Return(jen.Qual("fmt", "Sprintf").Call(
				jen.Lit("%s(%d)"),
				jen.Lit(typeName),
				jen.Id("e"),
			)),
		),
	)

	g.File.Func().Params(jen.Id("e").Id(typeName)).Id("String").Params().String().Block(
		jen.Switch(jen.Id("e")).Block(cases...),
	)
	g.File.Line()
}

// generateTypeAlias generates a type alias.
// Example: type MyU64 = uint64
func (g *TypesGenerator) generateTypeAlias(typeDef codegen.IDLTypeDef) error {
	typeName := FormatTypeName(typeDef.Name)
	aliasType := g.ResolveType(&typeDef.Type)

	g.File.Type().Id(typeName).Op("=").Add(aliasType)
	g.File.Line()

	return nil
}

// FormatVariantName formats an enum variant name.
// Examples:
//   - Status + Pending -> StatusPending
//   - Action + Transfer -> ActionTransfer
func FormatVariantName(enumName, variantName string) string {
	return enumName + FormatTypeName(variantName)
}

// GenerateTypesFile generates the types.go file.
func GenerateTypesFile(idl *codegen.IDL, packageName, outputDir string) error {
	gen := NewGenerator(idl, packageName)
	typesGen := NewTypesGenerator(gen)

	// Add file header comment
	gen.File.Comment("Code generated by go-carbon. DO NOT EDIT.")
	gen.File.Line()

	// Generate all types
	if err := typesGen.Generate(); err != nil {
		return err
	}

	// Save to file
	filename := fmt.Sprintf("%s/types.go", outputDir)
	return gen.WriteToFile(filename)
}
