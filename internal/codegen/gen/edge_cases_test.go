package gen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lugondev/go-carbon/internal/codegen"
)

func TestEdgeCaseNestedTypes(t *testing.T) {
	idl := &codegen.IDL{
		Address: "Test111111111111111111111111111111111111111",
		Metadata: codegen.IDLMetadata{
			Name:    "nested_test",
			Version: "1.0.0",
		},
		Types: []codegen.IDLTypeDef{
			{
				Name: "inner_config",
				Type: codegen.IDLType{
					Struct: &codegen.IDLStructType{
						Fields: []codegen.IDLField{
							{Name: "value", Type: codegen.IDLType{Kind: "u64"}},
						},
					},
				},
			},
			{
				Name: "middle_config",
				Type: codegen.IDLType{
					Struct: &codegen.IDLStructType{
						Fields: []codegen.IDLField{
							{Name: "inner", Type: codegen.IDLType{Defined: &codegen.IDLDefinedType{Name: "inner_config"}}},
						},
					},
				},
			},
			{
				Name: "outer_config",
				Type: codegen.IDLType{
					Struct: &codegen.IDLStructType{
						Fields: []codegen.IDLField{
							{Name: "middle", Type: codegen.IDLType{Defined: &codegen.IDLDefinedType{Name: "middle_config"}}},
						},
					},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	err := GenerateAll(idl, "nested_test", tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate nested types: %v", err)
	}

	typesFile := filepath.Join(tmpDir, "types.go")
	content, err := os.ReadFile(typesFile)
	if err != nil {
		t.Fatalf("Failed to read types.go: %v", err)
	}

	code := string(content)
	if !contains(code, "type InnerConfig struct") {
		t.Error("Missing InnerConfig type")
	}
	if !contains(code, "type MiddleConfig struct") {
		t.Error("Missing MiddleConfig type")
	}
	if !contains(code, "type OuterConfig struct") {
		t.Error("Missing OuterConfig type")
	}
	if !contains(code, "Middle MiddleConfig") {
		t.Error("Missing nested field reference")
	}
}

func TestEdgeCaseVecOfVec(t *testing.T) {
	idl := &codegen.IDL{
		Address: "Test111111111111111111111111111111111111111",
		Metadata: codegen.IDLMetadata{
			Name:    "vec_test",
			Version: "1.0.0",
		},
		Types: []codegen.IDLTypeDef{
			{
				Name: "matrix",
				Type: codegen.IDLType{
					Struct: &codegen.IDLStructType{
						Fields: []codegen.IDLField{
							{
								Name: "data",
								Type: codegen.IDLType{
									Vec: &codegen.IDLType{
										Vec: &codegen.IDLType{Kind: "u64"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	err := GenerateAll(idl, "vec_test", tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate vec of vec: %v", err)
	}

	typesFile := filepath.Join(tmpDir, "types.go")
	content, err := os.ReadFile(typesFile)
	if err != nil {
		t.Fatalf("Failed to read types.go: %v", err)
	}

	if !contains(string(content), "[][]uint64") {
		t.Error("Missing nested vec type [][]uint64")
	}
}

func TestEdgeCaseOptionOfVec(t *testing.T) {
	idl := &codegen.IDL{
		Address: "Test111111111111111111111111111111111111111",
		Metadata: codegen.IDLMetadata{
			Name:    "option_test",
			Version: "1.0.0",
		},
		Types: []codegen.IDLTypeDef{
			{
				Name: "optional_list",
				Type: codegen.IDLType{
					Struct: &codegen.IDLStructType{
						Fields: []codegen.IDLField{
							{
								Name: "items",
								Type: codegen.IDLType{
									Option: &codegen.IDLType{
										Vec: &codegen.IDLType{Kind: "pubkey"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	err := GenerateAll(idl, "option_test", tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate option of vec: %v", err)
	}

	typesFile := filepath.Join(tmpDir, "types.go")
	content, err := os.ReadFile(typesFile)
	if err != nil {
		t.Fatalf("Failed to read types.go: %v", err)
	}

	if !contains(string(content), "*[]") {
		t.Error("Missing option of vec type pointer")
	}
}

func TestEdgeCaseManyInstructions(t *testing.T) {
	instructions := make([]codegen.IDLInstruction, 20)
	for i := 0; i < 20; i++ {
		instructions[i] = codegen.IDLInstruction{
			Name:          "instruction_" + string(rune('a'+i)),
			Discriminator: []byte{byte(i), 0, 0, 0, 0, 0, 0, 0},
			Args: []codegen.IDLField{
				{Name: "amount", Type: codegen.IDLType{Kind: "u64"}},
			},
			Accounts: []codegen.IDLAccountMeta{
				{Name: "user", Signer: true},
			},
		}
	}

	idl := &codegen.IDL{
		Address: "Test111111111111111111111111111111111111111",
		Metadata: codegen.IDLMetadata{
			Name:    "many_ix_test",
			Version: "1.0.0",
		},
		Instructions: instructions,
	}

	tmpDir := t.TempDir()
	err := GenerateAll(idl, "many_ix_test", tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate many instructions: %v", err)
	}

	ixFile := filepath.Join(tmpDir, "instructions.go")
	content, err := os.ReadFile(ixFile)
	if err != nil {
		t.Fatalf("Failed to read instructions.go: %v", err)
	}

	code := string(content)
	for i := 0; i < 20; i++ {
		instrName := "Instruction" + string(rune('A'+i))
		if !contains(code, instrName) {
			t.Errorf("Missing instruction: %s", instrName)
			break
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
