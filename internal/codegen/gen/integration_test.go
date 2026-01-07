package gen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lugondev/go-carbon/internal/codegen"
	"github.com/lugondev/go-carbon/internal/codegen/gen"
)

func TestGenerateAll_MinimalIDL(t *testing.T) {
	idl := &codegen.IDL{
		Address: "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
		Metadata: codegen.IDLMetadata{
			Name:    "test_program",
			Version: "0.1.0",
			Spec:    "0.1.0",
		},
		Types: []codegen.IDLTypeDef{
			{
				Name: "user_data",
				Type: codegen.IDLType{
					Struct: &codegen.IDLStructType{
						Fields: []codegen.IDLField{
							{
								Name: "amount",
								Type: codegen.IDLType{Kind: "u64"},
							},
							{
								Name: "owner",
								Type: codegen.IDLType{Kind: "pubkey"},
							},
						},
					},
				},
			},
		},
		Accounts: []codegen.IDLAccountDef{
			{
				Name:          "user_account",
				Discriminator: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				Type: codegen.IDLType{
					Struct: &codegen.IDLStructType{
						Fields: []codegen.IDLField{
							{
								Name: "balance",
								Type: codegen.IDLType{Kind: "u64"},
							},
						},
					},
				},
			},
		},
		Instructions: []codegen.IDLInstruction{
			{
				Name:          "initialize",
				Discriminator: []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18},
				Args: []codegen.IDLField{
					{
						Name: "amount",
						Type: codegen.IDLType{Kind: "u64"},
					},
				},
				Accounts: []codegen.IDLAccountMeta{
					{
						Name:     "user",
						Signer:   true,
						Writable: true,
					},
					{
						Name:     "system_program",
						Signer:   false,
						Writable: false,
					},
				},
			},
		},
		Events: []codegen.IDLEvent{
			{
				Name:          "initialized",
				Discriminator: []byte{0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28},
				Fields: []codegen.IDLField{
					{
						Name: "user",
						Type: codegen.IDLType{Kind: "pubkey"},
					},
					{
						Name: "amount",
						Type: codegen.IDLType{Kind: "u64"},
					},
				},
			},
		},
	}

	tmpDir := t.TempDir()

	if err := gen.GenerateAll(idl, "testprog", tmpDir); err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	expectedFiles := []string{
		"program.go",
		"types.go",
		"accounts.go",
		"instructions.go",
		"events.go",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not generated", file)
		} else {
			t.Logf("✓ Generated: %s", file)
		}
	}
}

func TestGenerateAll_TypesOnly(t *testing.T) {
	idl := &codegen.IDL{
		Address: "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
		Metadata: codegen.IDLMetadata{
			Name:    "types_test",
			Version: "0.1.0",
			Spec:    "0.1.0",
		},
		Types: []codegen.IDLTypeDef{
			{
				Name: "status",
				Type: codegen.IDLType{
					Enum: &codegen.IDLEnumType{
						Variants: []codegen.IDLEnumVariant{
							{Name: "pending"},
							{Name: "active"},
							{Name: "closed"},
						},
					},
				},
			},
		},
	}

	tmpDir := t.TempDir()

	if err := gen.GenerateAll(idl, "typestest", tmpDir); err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "program.go")); os.IsNotExist(err) {
		t.Error("program.go should always be generated")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "types.go")); os.IsNotExist(err) {
		t.Error("types.go should be generated when types are present")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "accounts.go")); !os.IsNotExist(err) {
		t.Error("accounts.go should not be generated when no accounts")
	}
}

func TestGenerateAll_Validation(t *testing.T) {
	tests := []struct {
		name    string
		idl     *codegen.IDL
		wantErr bool
	}{
		{
			name:    "nil IDL",
			idl:     nil,
			wantErr: true,
		},
		{
			name: "empty address",
			idl: &codegen.IDL{
				Address: "",
				Metadata: codegen.IDLMetadata{
					Name: "test",
				},
			},
			wantErr: true,
		},
		{
			name: "empty name",
			idl: &codegen.IDL{
				Address: "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
				Metadata: codegen.IDLMetadata{
					Name: "",
				},
			},
			wantErr: true,
		},
		{
			name: "valid minimal",
			idl: &codegen.IDL{
				Address: "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
				Metadata: codegen.IDLMetadata{
					Name: "valid",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			err := gen.GenerateAll(tt.idl, "test", tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateAll() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetGeneratedFiles(t *testing.T) {
	idl := &codegen.IDL{
		Types:        []codegen.IDLTypeDef{{Name: "test"}},
		Accounts:     []codegen.IDLAccountDef{{Name: "test"}},
		Instructions: []codegen.IDLInstruction{{Name: "test"}},
		Events:       []codegen.IDLEvent{{Name: "test"}},
	}

	files := gen.GetGeneratedFiles(idl)
	expected := []string{"program.go", "types.go", "accounts.go", "instructions.go", "events.go"}

	if len(files) != len(expected) {
		t.Errorf("Expected %d files, got %d", len(expected), len(files))
	}

	fileMap := make(map[string]bool)
	for _, f := range files {
		fileMap[f] = true
	}

	for _, exp := range expected {
		if !fileMap[exp] {
			t.Errorf("Expected file %s not in result", exp)
		}
	}
}

func TestDebugTypesGeneration(t *testing.T) {
	idl := &codegen.IDL{
		Address: "Test111111111111111111111111111111111111111",
		Metadata: codegen.IDLMetadata{
			Name:    "test",
			Version: "0.1.0",
		},
		Types: []codegen.IDLTypeDef{
			{
				Name: "user_data",
				Type: codegen.IDLType{
					Struct: &codegen.IDLStructType{
						Fields: []codegen.IDLField{
							{
								Name: "amount",
								Type: codegen.IDLType{Kind: "u64"},
							},
							{
								Name: "owner",
								Type: codegen.IDLType{Kind: "pubkey"},
							},
						},
					},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	if err := gen.GenerateAll(idl, "test", tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "types.go"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	t.Logf("Generated types.go:\n%s", string(content))

	if !strings.Contains(string(content), "type UserData struct") {
		t.Errorf("Expected 'type UserData struct' in generated code")
	}
}
