package codegen

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/lugondev/go-carbon/pkg/utils"
)

type Generator struct {
	IDL         *IDL
	PackageName string
	OutputDir   string
}

func NewGenerator(idl *IDL, packageName, outputDir string) *Generator {
	if packageName == "" {
		packageName = utils.ToSnakeCase(idl.Metadata.Name)
	}
	return &Generator{
		IDL:         idl,
		PackageName: packageName,
		OutputDir:   outputDir,
	}
}

func (g *Generator) Generate() error {
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := g.generateTypes(); err != nil {
		return fmt.Errorf("failed to generate types: %w", err)
	}

	if err := g.generateAccounts(); err != nil {
		return fmt.Errorf("failed to generate accounts: %w", err)
	}

	if err := g.generateEvents(); err != nil {
		return fmt.Errorf("failed to generate events: %w", err)
	}

	if err := g.generateInstructions(); err != nil {
		return fmt.Errorf("failed to generate instructions: %w", err)
	}

	if err := g.generateProgram(); err != nil {
		return fmt.Errorf("failed to generate program: %w", err)
	}

	return nil
}

func (g *Generator) generateTypes() error {
	if len(g.IDL.Types) == 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", g.PackageName))
	buf.WriteString("import (\n")
	buf.WriteString("\t\"github.com/gagliardetto/solana-go\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("var _ = solana.PublicKey{}\n\n")

	for _, typeDef := range g.IDL.Types {
		code := g.generateTypeDef(typeDef)
		buf.WriteString(code)
		buf.WriteString("\n")
	}

	return g.writeFile("types.go", buf.Bytes())
}

func (g *Generator) generateTypeDef(typeDef IDLTypeDef) string {
	name := utils.ToPascalCase(typeDef.Name)

	if typeDef.Type.Struct != nil {
		return g.generateStruct(name, typeDef.Type.Struct.Fields, typeDef.Docs)
	}

	if typeDef.Type.Enum != nil {
		return g.generateEnum(name, typeDef.Type.Enum.Variants, typeDef.Docs)
	}

	return ""
}

func (g *Generator) generateStruct(name string, fields []IDLField, docs []string) string {
	var buf bytes.Buffer

	for _, doc := range docs {
		buf.WriteString(fmt.Sprintf("// %s\n", doc))
	}
	buf.WriteString(fmt.Sprintf("type %s struct {\n", name))

	for _, field := range fields {
		fieldName := utils.ToPascalCase(field.Name)
		fieldType := g.idlTypeToGo(field.Type)
		jsonTag := utils.ToSnakeCase(field.Name)
		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\" borsh:\"%s\"`\n", fieldName, fieldType, jsonTag, jsonTag))
	}

	buf.WriteString("}\n")
	return buf.String()
}

func (g *Generator) generateEnum(name string, variants []IDLEnumVariant, docs []string) string {
	var buf bytes.Buffer

	for _, doc := range docs {
		buf.WriteString(fmt.Sprintf("// %s\n", doc))
	}
	buf.WriteString(fmt.Sprintf("type %s uint8\n\n", name))

	buf.WriteString("const (\n")
	for i, variant := range variants {
		variantName := fmt.Sprintf("%s%s", name, utils.ToPascalCase(variant.Name))
		if i == 0 {
			buf.WriteString(fmt.Sprintf("\t%s %s = iota\n", variantName, name))
		} else {
			buf.WriteString(fmt.Sprintf("\t%s\n", variantName))
		}
	}
	buf.WriteString(")\n")

	return buf.String()
}

func (g *Generator) generateAccounts() error {
	if len(g.IDL.Accounts) == 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", g.PackageName))
	buf.WriteString("import (\n")
	buf.WriteString("\t\"fmt\"\n\n")
	buf.WriteString("\t\"github.com/gagliardetto/solana-go\"\n")
	buf.WriteString(")\n\n")

	for _, account := range g.IDL.Accounts {
		code := g.generateAccountDef(account)
		buf.WriteString(code)
		buf.WriteString("\n")
	}

	return g.writeFile("accounts.go", buf.Bytes())
}

func (g *Generator) generateAccountDef(account IDLAccountDef) string {
	var buf bytes.Buffer
	name := utils.ToPascalCase(account.Name)
	discHex := formatDiscriminator(account.Discriminator)

	for _, doc := range account.Docs {
		buf.WriteString(fmt.Sprintf("// %s\n", doc))
	}

	buf.WriteString(fmt.Sprintf("var %sDiscriminator = [8]byte{%s}\n\n", name, discHex))

	if account.Type.Struct != nil {
		buf.WriteString(g.generateStruct(name, account.Type.Struct.Fields, nil))
	} else {
		buf.WriteString(fmt.Sprintf("type %s struct {\n", name))
		buf.WriteString("\tData []byte\n")
		buf.WriteString("}\n")
	}

	buf.WriteString(fmt.Sprintf(`
func (a *%s) Discriminator() [8]byte {
	return %sDiscriminator
}

func Decode%s(data []byte) (*%s, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for %s account")
	}

	var disc [8]byte
	copy(disc[:], data[:8])
	if disc != %sDiscriminator {
		return nil, fmt.Errorf("invalid discriminator for %s")
	}

	account := &%s{}
	// TODO: Implement Borsh deserialization
	_ = data[8:]
	return account, nil
}
`, name, name, name, name, name, name, name, name))

	return buf.String()
}

func (g *Generator) generateEvents() error {
	if len(g.IDL.Events) == 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", g.PackageName))
	buf.WriteString("import (\n")
	buf.WriteString("\t\"encoding/binary\"\n")
	buf.WriteString("\t\"fmt\"\n\n")
	buf.WriteString("\t\"github.com/gagliardetto/solana-go\"\n")
	buf.WriteString("\t\"github.com/lugondev/go-carbon/pkg/decoder\"\n")
	buf.WriteString("\t\"github.com/lugondev/go-carbon/internal/decoder/anchor\"\n")
	buf.WriteString(")\n\n")

	buf.WriteString("var _ = binary.LittleEndian\n\n")

	for _, event := range g.IDL.Events {
		code := g.generateEventDef(event)
		buf.WriteString(code)
		buf.WriteString("\n")
	}

	buf.WriteString(g.generateEventDecoders())

	return g.writeFile("events.go", buf.Bytes())
}

func (g *Generator) generateEventDef(event IDLEvent) string {
	var buf bytes.Buffer
	name := utils.ToPascalCase(event.Name)
	discHex := formatDiscriminator(event.Discriminator)

	for _, doc := range event.Docs {
		buf.WriteString(fmt.Sprintf("// %s\n", doc))
	}

	buf.WriteString(fmt.Sprintf("var %sEventDiscriminator = [8]byte{%s}\n\n", name, discHex))

	buf.WriteString(fmt.Sprintf("type %sEvent struct {\n", name))
	for _, field := range event.Fields {
		fieldName := utils.ToPascalCase(field.Name)
		fieldType := g.idlTypeToGo(field.Type)
		jsonTag := utils.ToSnakeCase(field.Name)
		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\" borsh:\"%s\"`\n", fieldName, fieldType, jsonTag, jsonTag))
	}
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf(`func (e *%sEvent) Discriminator() [8]byte {
	return %sEventDiscriminator
}

func Decode%sEvent(data []byte) (*%sEvent, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for %s event")
	}

	event := &%sEvent{}
	offset := 0
`, name, name, name, name, name, name))

	for _, field := range event.Fields {
		fieldName := utils.ToPascalCase(field.Name)
		buf.WriteString(g.generateFieldDecoder(fieldName, field.Type, "event"))
	}

	buf.WriteString("\n\t_ = offset\n")
	buf.WriteString("\treturn event, nil\n}\n")

	return buf.String()
}

func (g *Generator) generateFieldDecoder(fieldName string, fieldType IDLType, varName string) string {
	goType := g.idlTypeToGo(fieldType)

	switch goType {
	case "uint8":
		return fmt.Sprintf("\t%s.%s = data[offset]\n\toffset += 1\n", varName, fieldName)
	case "uint16":
		return fmt.Sprintf("\t%s.%s = binary.LittleEndian.Uint16(data[offset:])\n\toffset += 2\n", varName, fieldName)
	case "uint32":
		return fmt.Sprintf("\t%s.%s = binary.LittleEndian.Uint32(data[offset:])\n\toffset += 4\n", varName, fieldName)
	case "uint64":
		return fmt.Sprintf("\t%s.%s = binary.LittleEndian.Uint64(data[offset:])\n\toffset += 8\n", varName, fieldName)
	case "int8":
		return fmt.Sprintf("\t%s.%s = int8(data[offset])\n\toffset += 1\n", varName, fieldName)
	case "int16":
		return fmt.Sprintf("\t%s.%s = int16(binary.LittleEndian.Uint16(data[offset:]))\n\toffset += 2\n", varName, fieldName)
	case "int32":
		return fmt.Sprintf("\t%s.%s = int32(binary.LittleEndian.Uint32(data[offset:]))\n\toffset += 4\n", varName, fieldName)
	case "int64":
		return fmt.Sprintf("\t%s.%s = int64(binary.LittleEndian.Uint64(data[offset:]))\n\toffset += 8\n", varName, fieldName)
	case "bool":
		return fmt.Sprintf("\t%s.%s = data[offset] != 0\n\toffset += 1\n", varName, fieldName)
	case "solana.PublicKey":
		return fmt.Sprintf("\tcopy(%s.%s[:], data[offset:offset+32])\n\toffset += 32\n", varName, fieldName)
	case "[16]byte":
		return fmt.Sprintf("\tcopy(%s.%s[:], data[offset:offset+16])\n\toffset += 16\n", varName, fieldName)
	default:
		return fmt.Sprintf("\t// TODO: decode %s (%s)\n", fieldName, goType)
	}
}

func (g *Generator) generateEventDecoders() string {
	if len(g.IDL.Events) == 0 {
		return ""
	}

	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf(`
func NewEventDecoders(programID solana.PublicKey) []decoder.Decoder {
	return []decoder.Decoder{
`))

	for _, event := range g.IDL.Events {
		name := utils.ToPascalCase(event.Name)
		buf.WriteString(fmt.Sprintf("\t\tNew%sDecoder(programID),\n", name))
	}

	buf.WriteString("\t}\n}\n\n")

	for _, event := range g.IDL.Events {
		name := utils.ToPascalCase(event.Name)
		buf.WriteString(fmt.Sprintf(`func New%sDecoder(programID solana.PublicKey) decoder.Decoder {
	return anchor.NewAnchorEventDecoder(
		"%s",
		programID,
		decoder.NewAnchorDiscriminator(%sEventDiscriminator[:]),
		func(data []byte) (interface{}, error) {
			return Decode%sEvent(data)
		},
	)
}

`, name, event.Name, name, name))
	}

	return buf.String()
}

func (g *Generator) generateInstructions() error {
	if len(g.IDL.Instructions) == 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", g.PackageName))
	buf.WriteString("import (\n")
	buf.WriteString("\t\"github.com/gagliardetto/solana-go\"\n")
	buf.WriteString(")\n\n")

	for _, ix := range g.IDL.Instructions {
		code := g.generateInstructionDef(ix)
		buf.WriteString(code)
		buf.WriteString("\n")
	}

	return g.writeFile("instructions.go", buf.Bytes())
}

func (g *Generator) generateInstructionDef(ix IDLInstruction) string {
	var buf bytes.Buffer
	name := utils.ToPascalCase(ix.Name)
	discHex := formatDiscriminator(ix.Discriminator)

	for _, doc := range ix.Docs {
		buf.WriteString(fmt.Sprintf("// %s\n", doc))
	}

	buf.WriteString(fmt.Sprintf("var %sDiscriminator = [8]byte{%s}\n\n", name, discHex))

	buf.WriteString(fmt.Sprintf("type %sInstruction struct {\n", name))
	for _, arg := range ix.Args {
		argName := utils.ToPascalCase(arg.Name)
		argType := g.idlTypeToGo(arg.Type)
		jsonTag := utils.ToSnakeCase(arg.Name)
		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\" borsh:\"%s\"`\n", argName, argType, jsonTag, jsonTag))
	}
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("type %sAccounts struct {\n", name))
	for _, acc := range ix.Accounts {
		accName := utils.ToPascalCase(acc.Name)
		buf.WriteString(fmt.Sprintf("\t%s solana.PublicKey\n", accName))
	}
	buf.WriteString("}\n")

	return buf.String()
}

func (g *Generator) generateProgram() error {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("package %s\n\n", g.PackageName))
	buf.WriteString("import (\n")
	buf.WriteString("\t\"github.com/gagliardetto/solana-go\"\n")
	buf.WriteString("\t\"github.com/lugondev/go-carbon/pkg/decoder\"\n")
	buf.WriteString("\t\"github.com/lugondev/go-carbon/pkg/plugin\"\n")
	buf.WriteString("\t\"github.com/lugondev/go-carbon/internal/decoder/anchor\"\n")
	buf.WriteString(")\n\n")

	programName := utils.ToPascalCase(g.IDL.Metadata.Name)
	programID := g.IDL.Address

	buf.WriteString(fmt.Sprintf("const ProgramName = \"%s\"\n", g.IDL.Metadata.Name))
	buf.WriteString(fmt.Sprintf("const ProgramVersion = \"%s\"\n", g.IDL.Metadata.Version))

	if programID != "" {
		buf.WriteString(fmt.Sprintf("var ProgramID = solana.MustPublicKeyFromBase58(\"%s\")\n\n", programID))
	} else {
		buf.WriteString("var ProgramID solana.PublicKey\n\n")
	}

	buf.WriteString(fmt.Sprintf(`func New%sPlugin(programID solana.PublicKey) plugin.Plugin {
	decoders := NewEventDecoders(programID)
	return anchor.NewAnchorEventPlugin(
		ProgramName,
		programID,
		decoders,
	)
}

func GetDecoderRegistry(programID solana.PublicKey) *decoder.Registry {
	registry := decoder.NewRegistry()
	for _, d := range NewEventDecoders(programID) {
		registry.Register(d.GetName(), d)
	}
	return registry
}
`, programName))

	return g.writeFile("program.go", buf.Bytes())
}

func (g *Generator) idlTypeToGo(t IDLType) string {
	if t.Kind != "" {
		return primitiveToGo(t.Kind)
	}

	if t.Defined != nil {
		return utils.ToPascalCase(t.Defined.Name)
	}

	if t.Option != nil {
		innerType := g.idlTypeToGo(*t.Option)
		return "*" + innerType
	}

	if t.Vec != nil {
		innerType := g.idlTypeToGo(*t.Vec)
		return "[]" + innerType
	}

	if t.Array != nil {
		innerType := g.idlTypeToGo(t.Array.Type)
		return fmt.Sprintf("[%d]%s", t.Array.Len, innerType)
	}

	return "interface{}"
}

func primitiveToGo(kind string) string {
	switch kind {
	case "bool":
		return "bool"
	case "u8":
		return "uint8"
	case "u16":
		return "uint16"
	case "u32":
		return "uint32"
	case "u64":
		return "uint64"
	case "u128":
		return "[16]byte"
	case "i8":
		return "int8"
	case "i16":
		return "int16"
	case "i32":
		return "int32"
	case "i64":
		return "int64"
	case "i128":
		return "[16]byte"
	case "f32":
		return "float32"
	case "f64":
		return "float64"
	case "string":
		return "string"
	case "bytes":
		return "[]byte"
	case "pubkey", "publicKey":
		return "solana.PublicKey"
	default:
		return "interface{}"
	}
}

func (g *Generator) writeFile(filename string, content []byte) error {
	formatted, err := format.Source(content)
	if err != nil {
		formatted = content
	}

	path := filepath.Join(g.OutputDir, filename)
	return os.WriteFile(path, formatted, 0644)
}

func formatDiscriminator(disc []byte) string {
	if len(disc) == 0 {
		return "0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00"
	}

	parts := make([]string, len(disc))
	for i, b := range disc {
		parts[i] = fmt.Sprintf("0x%02x", b)
	}
	return strings.Join(parts, ", ")
}

func computeEventDiscriminator(eventName string) [8]byte {
	data := []byte(fmt.Sprintf("event:%s", eventName))
	hash := sha256.Sum256(data)
	var disc [8]byte
	copy(disc[:], hash[:8])
	return disc
}

func computeInstructionDiscriminator(ixName string) [8]byte {
	data := []byte(fmt.Sprintf("global:%s", utils.ToSnakeCase(ixName)))
	hash := sha256.Sum256(data)
	var disc [8]byte
	copy(disc[:], hash[:8])
	return disc
}

var _ = hex.EncodeToString
var _ = template.New
