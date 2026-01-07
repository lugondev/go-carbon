package gen

import (
	"testing"

	"github.com/lugondev/go-carbon/internal/codegen"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple snake_case", "user_account", "UserAccount"},
		{"multiple underscores", "my_program_name", "MyProgramName"},
		{"single word", "program", "Program"},
		{"all caps", "USD", "USD"},
		{"mixed", "u64_value", "U64Value"},
		{"empty string", "", ""},
		{"single char", "a", "A"},
		{"with numbers", "token_2022", "Token2022"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple snake_case", "user_account", "userAccount"},
		{"multiple underscores", "my_program_name", "myProgramName"},
		{"single word", "program", "program"},
		{"empty string", "", ""},
		{"single char", "a", "a"},
		{"with numbers", "token_2022", "token2022"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolveType_Primitives(t *testing.T) {
	gen := NewGenerator(&codegen.IDL{}, "test")

	tests := []struct {
		name     string
		idlType  *codegen.IDLType
		expected string
	}{
		{"u8", &codegen.IDLType{Kind: "u8"}, "uint8"},
		{"u16", &codegen.IDLType{Kind: "u16"}, "uint16"},
		{"u32", &codegen.IDLType{Kind: "u32"}, "uint32"},
		{"u64", &codegen.IDLType{Kind: "u64"}, "uint64"},
		{"i8", &codegen.IDLType{Kind: "i8"}, "int8"},
		{"i16", &codegen.IDLType{Kind: "i16"}, "int16"},
		{"i32", &codegen.IDLType{Kind: "i32"}, "int32"},
		{"i64", &codegen.IDLType{Kind: "i64"}, "int64"},
		{"f32", &codegen.IDLType{Kind: "f32"}, "float32"},
		{"f64", &codegen.IDLType{Kind: "f64"}, "float64"},
		{"bool", &codegen.IDLType{Kind: "bool"}, "bool"},
		{"string", &codegen.IDLType{Kind: "string"}, "string"},
		{"bytes", &codegen.IDLType{Kind: "bytes"}, "[]byte"},
		{"pubkey", &codegen.IDLType{Kind: "pubkey"}, "PublicKey"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := gen.ResolveType(tt.idlType)
			result := stmt.GoString()

			// Check if the expected type is contained in the result
			if !containsType(result, tt.expected) {
				t.Errorf("ResolveType(%v) = %q; want to contain %q", tt.idlType.Kind, result, tt.expected)
			}
		})
	}
}

func TestResolveType_Option(t *testing.T) {
	gen := NewGenerator(&codegen.IDL{}, "test")

	// Option<u64>
	optionU64 := &codegen.IDLType{
		Option: &codegen.IDLType{Kind: "u64"},
	}

	stmt := gen.ResolveType(optionU64)
	result := stmt.GoString()

	// Should be *uint64
	if !containsType(result, "*") || !containsType(result, "uint64") {
		t.Errorf("ResolveType(Option<u64>) = %q; want to contain '*uint64'", result)
	}
}

func TestResolveType_Vec(t *testing.T) {
	gen := NewGenerator(&codegen.IDL{}, "test")

	// Vec<u8>
	vecU8 := &codegen.IDLType{
		Vec: &codegen.IDLType{Kind: "u8"},
	}

	stmt := gen.ResolveType(vecU8)
	result := stmt.GoString()

	// Should be []uint8
	if !containsType(result, "[]") || !containsType(result, "uint8") {
		t.Errorf("ResolveType(Vec<u8>) = %q; want to contain '[]uint8'", result)
	}
}

func TestResolveType_Array(t *testing.T) {
	gen := NewGenerator(&codegen.IDL{}, "test")

	// [u8; 32]
	arrayU8 := &codegen.IDLType{
		Array: &codegen.IDLArrayType{
			Type: codegen.IDLType{Kind: "u8"},
			Len:  32,
		},
	}

	stmt := gen.ResolveType(arrayU8)
	result := stmt.GoString()

	// Should be [32]uint8
	if !containsType(result, "[32]") || !containsType(result, "uint8") {
		t.Errorf("ResolveType([u8; 32]) = %q; want to contain '[32]uint8'", result)
	}
}

func TestResolveType_Defined(t *testing.T) {
	gen := NewGenerator(&codegen.IDL{}, "test")

	// Custom type: MyStruct
	definedType := &codegen.IDLType{
		Defined: &codegen.IDLDefinedType{
			Name: "my_struct",
		},
	}

	stmt := gen.ResolveType(definedType)
	result := stmt.GoString()

	// Should be MyStruct
	if !containsType(result, "MyStruct") {
		t.Errorf("ResolveType(my_struct) = %q; want to contain 'MyStruct'", result)
	}
}

func TestIsOptionalType(t *testing.T) {
	tests := []struct {
		name     string
		typ      *codegen.IDLType
		expected bool
	}{
		{"nil type", nil, false},
		{"Option<u64>", &codegen.IDLType{Option: &codegen.IDLType{Kind: "u64"}}, true},
		{"u64", &codegen.IDLType{Kind: "u64"}, false},
		{"Vec<u8>", &codegen.IDLType{Vec: &codegen.IDLType{Kind: "u8"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOptionalType(tt.typ)
			if result != tt.expected {
				t.Errorf("IsOptionalType(%v) = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsCOptionType(t *testing.T) {
	tests := []struct {
		name     string
		typ      *codegen.IDLType
		expected bool
	}{
		{"nil type", nil, false},
		{"COption<pubkey>", &codegen.IDLType{Coption: &codegen.IDLType{Kind: "pubkey"}}, true},
		{"pubkey", &codegen.IDLType{Kind: "pubkey"}, false},
		{"Option<pubkey>", &codegen.IDLType{Option: &codegen.IDLType{Kind: "pubkey"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCOptionType(tt.typ)
			if result != tt.expected {
				t.Errorf("IsCOptionType(%v) = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsVecType(t *testing.T) {
	tests := []struct {
		name     string
		typ      *codegen.IDLType
		expected bool
	}{
		{"nil type", nil, false},
		{"Vec<u8>", &codegen.IDLType{Vec: &codegen.IDLType{Kind: "u8"}}, true},
		{"u64", &codegen.IDLType{Kind: "u64"}, false},
		{"Array", &codegen.IDLType{Array: &codegen.IDLArrayType{}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsVecType(tt.typ)
			if result != tt.expected {
				t.Errorf("IsVecType(%v) = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsArrayType(t *testing.T) {
	tests := []struct {
		name     string
		typ      *codegen.IDLType
		expected bool
	}{
		{"nil type", nil, false},
		{"[u8; 32]", &codegen.IDLType{Array: &codegen.IDLArrayType{Type: codegen.IDLType{Kind: "u8"}, Len: 32}}, true},
		{"u64", &codegen.IDLType{Kind: "u64"}, false},
		{"Vec<u8>", &codegen.IDLType{Vec: &codegen.IDLType{Kind: "u8"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsArrayType(tt.typ)
			if result != tt.expected {
				t.Errorf("IsArrayType(%v) = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsPrimitiveType(t *testing.T) {
	tests := []struct {
		name     string
		typ      *codegen.IDLType
		expected bool
	}{
		{"nil type", nil, false},
		{"u64", &codegen.IDLType{Kind: "u64"}, true},
		{"string", &codegen.IDLType{Kind: "string"}, true},
		{"Vec<u8>", &codegen.IDLType{Vec: &codegen.IDLType{Kind: "u8"}}, false},
		{"Defined", &codegen.IDLType{Defined: &codegen.IDLDefinedType{Name: "Custom"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPrimitiveType(tt.typ)
			if result != tt.expected {
				t.Errorf("IsPrimitiveType(%v) = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsPublicKeyType(t *testing.T) {
	tests := []struct {
		name     string
		typ      *codegen.IDLType
		expected bool
	}{
		{"nil type", nil, false},
		{"pubkey", &codegen.IDLType{Kind: "pubkey"}, true},
		{"publicKey", &codegen.IDLType{Kind: "publicKey"}, true},
		{"u64", &codegen.IDLType{Kind: "u64"}, false},
		{"string", &codegen.IDLType{Kind: "string"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPublicKeyType(tt.typ)
			if result != tt.expected {
				t.Errorf("IsPublicKeyType(%v) = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestGetTypeSize(t *testing.T) {
	tests := []struct {
		name     string
		typ      *codegen.IDLType
		expected int
	}{
		{"nil", nil, -1},
		{"u8", &codegen.IDLType{Kind: "u8"}, 1},
		{"u16", &codegen.IDLType{Kind: "u16"}, 2},
		{"u32", &codegen.IDLType{Kind: "u32"}, 4},
		{"u64", &codegen.IDLType{Kind: "u64"}, 8},
		{"u128", &codegen.IDLType{Kind: "u128"}, 16},
		{"pubkey", &codegen.IDLType{Kind: "pubkey"}, 32},
		{"bool", &codegen.IDLType{Kind: "bool"}, 1},
		{"string", &codegen.IDLType{Kind: "string"}, -1},
		{"bytes", &codegen.IDLType{Kind: "bytes"}, -1},
		{"[u8; 32]", &codegen.IDLType{Array: &codegen.IDLArrayType{Type: codegen.IDLType{Kind: "u8"}, Len: 32}}, 32},
		{"[u64; 10]", &codegen.IDLType{Array: &codegen.IDLArrayType{Type: codegen.IDLType{Kind: "u64"}, Len: 10}}, 80},
		{"Vec<u8>", &codegen.IDLType{Vec: &codegen.IDLType{Kind: "u8"}}, -1},
		{"Option<u64>", &codegen.IDLType{Option: &codegen.IDLType{Kind: "u64"}}, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTypeSize(tt.typ)
			if result != tt.expected {
				t.Errorf("GetTypeSize(%v) = %d; want %d", tt.name, result, tt.expected)
			}
		})
	}
}

func TestFormatDocs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"empty", []string{}, nil},
		{"single line", []string{"This is a comment"}, []string{"// This is a comment"}},
		{"multiple lines", []string{"Line 1", "Line 2"}, []string{"// Line 1", "// Line 2"}},
		{"with empty lines", []string{"Line 1", "", "Line 2"}, []string{"// Line 1", "// Line 2"}},
		{"already formatted", []string{"// Already formatted"}, []string{"// Already formatted"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDocs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("FormatDocs(%v) length = %d; want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("FormatDocs(%v)[%d] = %q; want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestDiscriminatorToHex(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"empty", []byte{}, ""},
		{"single byte", []byte{0x01}, "01"},
		{"multiple bytes", []byte{0x01, 0x02, 0x03, 0x04}, "01020304"},
		{"8 bytes", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, "0102030405060708"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiscriminatorToHex(tt.input)
			if result != tt.expected {
				t.Errorf("DiscriminatorToHex(%v) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"constant", "MAX_SIZE", "MAX_SIZE"},
		{"field", "user_account", "UserAccount"},
		{"variable", "my_var", "myVar"},
		{"type", "custom_type", "CustomType"},
		{"function", "get_user", "GetUser"},
	}

	t.Run("FormatConstantName", func(t *testing.T) {
		for _, tt := range tests {
			result := FormatConstantName(tt.input)
			if result != tt.expected && tt.name == "constant" {
				t.Errorf("FormatConstantName(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("FormatFieldName", func(t *testing.T) {
		for _, tt := range tests {
			result := FormatFieldName(tt.input)
			if result != tt.expected && tt.name == "field" {
				t.Errorf("FormatFieldName(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("FormatTypeName", func(t *testing.T) {
		for _, tt := range tests {
			result := FormatTypeName(tt.input)
			if result != tt.expected && tt.name == "type" {
				t.Errorf("FormatTypeName(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		}
	})
}

// Helper function to check if a string contains a type representation
func containsType(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr ||
			s[0:len(substr)] == substr ||
			len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
