package gen

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/lugondev/go-carbon/internal/codegen"
)

// InstructionsGenerator generates instruction types and builders.
type InstructionsGenerator struct {
	*Generator
}

// NewInstructionsGenerator creates a new instructions generator.
func NewInstructionsGenerator(gen *Generator) *InstructionsGenerator {
	return &InstructionsGenerator{Generator: gen}
}

// Generate generates all instructions from the IDL.
func (g *InstructionsGenerator) Generate() error {
	if len(g.IDL.Instructions) == 0 {
		return nil
	}

	// Generate each instruction
	for _, instr := range g.IDL.Instructions {
		if err := g.generateInstruction(instr); err != nil {
			return fmt.Errorf("failed to generate instruction %s: %w", instr.Name, err)
		}
	}

	// Generate instruction parser (unified decoder)
	if err := g.generateInstructionParser(); err != nil {
		return fmt.Errorf("failed to generate instruction parser: %w", err)
	}

	return nil
}

// generateInstruction generates a single instruction type and builder.
func (g *InstructionsGenerator) generateInstruction(instr codegen.IDLInstruction) error {
	instrName := FormatTypeName(instr.Name)

	// Add documentation
	if len(instr.Docs) > 0 {
		for _, doc := range FormatDocs(instr.Docs) {
			g.File.Comment(doc)
		}
	}

	// Generate instruction struct
	if err := g.generateInstructionStruct(instrName, instr); err != nil {
		return err
	}

	// Generate discriminator constant
	g.generateInstructionDiscriminator(instrName, instr.Discriminator)

	// Generate NewXxxInstruction builder function
	if err := g.generateInstructionBuilder(instrName, instr); err != nil {
		return err
	}

	// Generate Build() method
	if err := g.generateInstructionBuild(instrName, instr); err != nil {
		return err
	}

	// Generate ValidateAccounts() method
	if err := g.generateInstructionValidate(instrName, instr); err != nil {
		return err
	}

	g.File.Line()
	return nil
}

// generateInstructionStruct generates the instruction struct type.
func (g *InstructionsGenerator) generateInstructionStruct(instrName string, instr codegen.IDLInstruction) error {
	fields := []jen.Code{}

	// Add args fields
	for _, arg := range instr.Args {
		fieldName := FormatFieldName(arg.Name)
		fieldType := g.ResolveType(&arg.Type)
		tags := GenerateFieldTag(arg.Name)

		field := jen.Id(fieldName).Add(fieldType).Tag(tags)
		if len(arg.Docs) > 0 {
			field = AddDocs(field, arg.Docs)
		}
		fields = append(fields, field)
	}

	// Add accounts as a nested struct
	if len(instr.Accounts) > 0 {
		accountFields := []jen.Code{}
		for _, acc := range instr.Accounts {
			accFieldName := FormatFieldName(acc.Name)
			accType := jen.Qual("github.com/gagliardetto/solana-go", "PublicKey")

			// Optional accounts are pointers
			if acc.Optional {
				accType = jen.Op("*").Add(accType)
			}

			accField := jen.Id(accFieldName).Add(accType).Tag(map[string]string{"json": acc.Name})
			if len(acc.Docs) > 0 {
				accField = AddDocs(accField, acc.Docs)
			}
			accountFields = append(accountFields, accField)
		}

		fields = append(fields, jen.Id("Accounts").Struct(accountFields...))
	}

	g.File.Type().Id(instrName + "Instruction").Struct(fields...)
	g.File.Line()

	return nil
}

// generateInstructionDiscriminator generates the discriminator constant.
func (g *InstructionsGenerator) generateInstructionDiscriminator(instrName string, disc []byte) {
	constName := instrName + "InstructionDiscriminator"
	g.File.Var().Id(constName).Op("=").Add(DiscriminatorToBytes(disc))
	g.File.Line()
}

// generateInstructionBuilder generates the NewXxxInstruction builder function.
func (g *InstructionsGenerator) generateInstructionBuilder(instrName string, instr codegen.IDLInstruction) error {
	builderName := "New" + instrName + "Instruction"
	instrTypeName := instrName + "Instruction"

	// Collect parameters for builder
	params := []jen.Code{}

	// Add account parameters
	for _, acc := range instr.Accounts {
		accParamName := FormatVariableName(acc.Name)
		accType := jen.Qual("github.com/gagliardetto/solana-go", "PublicKey")
		if acc.Optional {
			accType = jen.Op("*").Add(accType)
		}
		params = append(params, jen.Id(accParamName).Add(accType))
	}

	// Add args parameters
	for _, arg := range instr.Args {
		argParamName := FormatVariableName(arg.Name)
		argType := g.ResolveType(&arg.Type)
		params = append(params, jen.Id(argParamName).Add(argType))
	}

	// Generate function body
	g.File.Func().Id(builderName).Params(params...).Op("*").Id(instrTypeName).Block(
		jen.Return(jen.Op("&").Id(instrTypeName).Values(
			g.generateBuilderInitializer(instr)...,
		)),
	)
	g.File.Line()

	return nil
}

// generateBuilderInitializer generates the struct initializer for builder.
func (g *InstructionsGenerator) generateBuilderInitializer(instr codegen.IDLInstruction) []jen.Code {
	values := []jen.Code{}

	// Initialize args
	for _, arg := range instr.Args {
		fieldName := FormatFieldName(arg.Name)
		paramName := FormatVariableName(arg.Name)
		values = append(values, jen.Id(fieldName).Op(":").Id(paramName))
	}

	// Initialize accounts
	if len(instr.Accounts) > 0 {
		// Build struct type fields with tags
		structFields := []jen.Code{}
		for _, acc := range instr.Accounts {
			accFieldName := FormatFieldName(acc.Name)
			accType := jen.Qual("github.com/gagliardetto/solana-go", "PublicKey")
			if acc.Optional {
				accType = jen.Op("*").Add(accType)
			}
			structFields = append(structFields,
				jen.Id(accFieldName).Add(accType).Tag(map[string]string{"json": acc.Name}),
			)
		}

		// Build struct values
		structValues := []jen.Code{}
		for _, acc := range instr.Accounts {
			accFieldName := FormatFieldName(acc.Name)
			accParamName := FormatVariableName(acc.Name)
			structValues = append(structValues, jen.Id(accFieldName).Op(":").Id(accParamName))
		}

		// Combine: Accounts: struct{...fields...}{...values...}
		values = append(values,
			jen.Id("Accounts").Op(":").Struct(structFields...).Values(structValues...),
		)
	}

	return values
}

// generateInstructionBuild generates the Build() method.
func (g *InstructionsGenerator) generateInstructionBuild(instrName string, instr codegen.IDLInstruction) error {
	instrTypeName := instrName + "Instruction"
	discConst := instrName + "InstructionDiscriminator"

	g.File.Comment("Build creates a Solana instruction from this instruction data.")
	g.File.Func().Params(jen.Id("ix").Op("*").Id(instrTypeName)).Id("Build").Params().Params(
		jen.Op("*").Qual("github.com/gagliardetto/solana-go", "GenericInstruction"),
		jen.Error(),
	).Block(
		// Validate accounts
		jen.If(jen.Err().Op(":=").Id("ix").Dot("ValidateAccounts").Call(), jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Err()),
		),
		jen.Line(),

		// Serialize instruction data with discriminator
		jen.Comment("Serialize instruction data"),
		jen.List(jen.Id("data"), jen.Err()).Op(":=").Qual("github.com/gagliardetto/binary", "MarshalBorsh").Call(jen.Id("ix")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode instruction: %w"), jen.Err())),
		),
		jen.Line(),

		// Prepend discriminator
		jen.Id("fullData").Op(":=").Append(jen.Id(discConst), jen.Id("data").Op("...")),
		jen.Line(),

		// Build account metas
		jen.Id("accounts").Op(":=").Index().Op("*").Qual("github.com/gagliardetto/solana-go", "AccountMeta").Values(
			g.generateAccountMetas(instr.Accounts)...,
		),
		jen.Line(),

		// Create instruction using NewInstruction
		jen.Return(
			jen.Qual("github.com/gagliardetto/solana-go", "NewInstruction").Call(
				jen.Id("ProgramID"),
				jen.Id("accounts"),
				jen.Id("fullData"),
			),
			jen.Nil(),
		),
	)
	g.File.Line()

	return nil
}

// generateAccountMetas generates account metas for Build method.
func (g *InstructionsGenerator) generateAccountMetas(accounts []codegen.IDLAccountMeta) []jen.Code {
	metas := []jen.Code{}

	for _, acc := range accounts {
		accField := "ix.Accounts." + FormatFieldName(acc.Name)

		if acc.Optional {
			// Optional account: check if not nil
			metas = append(metas,
				jen.If(jen.Id(accField).Op("!=").Nil()).Block(
					jen.Op("&").Qual("github.com/gagliardetto/solana-go", "AccountMeta").Values(
						jen.Id("PublicKey").Op(":").Op("*").Id(accField),
						jen.Id("IsWritable").Op(":").Lit(acc.Writable),
						jen.Id("IsSigner").Op(":").Lit(acc.Signer),
					),
				),
			)
		} else {
			// Required account
			metas = append(metas,
				jen.Op("&").Qual("github.com/gagliardetto/solana-go", "AccountMeta").Values(
					jen.Id("PublicKey").Op(":").Id(accField),
					jen.Id("IsWritable").Op(":").Lit(acc.Writable),
					jen.Id("IsSigner").Op(":").Lit(acc.Signer),
				),
			)
		}
	}

	return metas
}

// generateInstructionValidate generates the ValidateAccounts() method.
func (g *InstructionsGenerator) generateInstructionValidate(instrName string, instr codegen.IDLInstruction) error {
	instrTypeName := instrName + "Instruction"

	g.File.Func().Params(jen.Id("ix").Op("*").Id(instrTypeName)).Id("ValidateAccounts").Params().Error().Block(
		jen.Return(jen.Nil()),
	)
	g.File.Line()

	return nil
}

// generateInstructionParser generates a unified instruction parser.
func (g *InstructionsGenerator) generateInstructionParser() error {
	g.File.Comment("ParseInstruction parses an instruction from raw data.")
	g.File.Comment("It uses the discriminator (first 8 bytes) to identify the instruction type.")
	g.File.Func().Id("ParseInstruction").Params(
		jen.Id("data").Index().Byte(),
	).Params(
		jen.Interface(),
		jen.Error(),
	).Block(
		// Check minimum length
		jen.If(jen.Len(jen.Id("data")).Op("<").Lit(8)).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("instruction data too short"))),
		),
		jen.Line(),

		// Extract discriminator
		jen.Id("discriminator").Op(":=").Id("data").Index(jen.Lit(0).Op(":").Lit(8)),
		jen.Id("instructionData").Op(":=").Id("data").Index(jen.Lit(8).Op(":")),
		jen.Line(),

		// Switch on discriminator
		jen.Switch(jen.Qual("encoding/hex", "EncodeToString").Call(jen.Id("discriminator"))).Block(
			g.generateParserCases()...,
		),
	)
	g.File.Line()

	return nil
}

// generateParserCases generates switch cases for instruction parser.
func (g *InstructionsGenerator) generateParserCases() []jen.Code {
	cases := []jen.Code{}

	for _, instr := range g.IDL.Instructions {
		instrName := FormatTypeName(instr.Name)
		instrTypeName := instrName + "Instruction"
		discHex := DiscriminatorToHex(instr.Discriminator)

		cases = append(cases,
			jen.Case(jen.Lit(discHex)).Block(
				jen.Var().Id("ix").Id(instrTypeName),
				jen.If(
					jen.Err().Op(":=").Qual("github.com/gagliardetto/binary", "UnmarshalBorsh").Call(
						jen.Op("&").Id("ix"),
						jen.Id("instructionData"),
					),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.Return(jen.Nil(), jen.Err()),
				),
				jen.Return(jen.Op("&").Id("ix"), jen.Nil()),
			),
		)
	}

	// Default case
	cases = append(cases,
		jen.Default().Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
				jen.Lit("unknown instruction discriminator: %s"),
				jen.Qual("encoding/hex", "EncodeToString").Call(jen.Id("discriminator")),
			)),
		),
	)

	return cases
}

// GenerateInstructionsFile generates the instructions.go file.
func GenerateInstructionsFile(idl *codegen.IDL, packageName, outputDir string) error {
	gen := NewGenerator(idl, packageName)
	instrGen := NewInstructionsGenerator(gen)

	// Add file header comment
	gen.File.Comment("Code generated by go-carbon. DO NOT EDIT.")
	gen.File.Line()

	// Generate all instructions
	if err := instrGen.Generate(); err != nil {
		return err
	}

	// Save to file
	filename := fmt.Sprintf("%s/instructions.go", outputDir)
	return gen.WriteToFile(filename)
}
