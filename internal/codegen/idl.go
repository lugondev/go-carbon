// Package codegen provides code generation from Anchor IDL JSON files.
package codegen

// IDL represents an Anchor IDL (Interface Definition Language) structure.
// This is the JSON schema used by Anchor to describe Solana programs.
type IDL struct {
	Address      string           `json:"address"`
	Metadata     IDLMetadata      `json:"metadata"`
	Instructions []IDLInstruction `json:"instructions"`
	Accounts     []IDLAccountDef  `json:"accounts"`
	Events       []IDLEvent       `json:"events"`
	Errors       []IDLError       `json:"errors"`
	Types        []IDLTypeDef     `json:"types"`
	Constants    []IDLConstant    `json:"constants,omitempty"`
}

// IDLMetadata contains program metadata.
type IDLMetadata struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Spec        string `json:"spec"`
	Description string `json:"description,omitempty"`
}

// IDLInstruction represents a program instruction.
type IDLInstruction struct {
	Name          string           `json:"name"`
	Discriminator []byte           `json:"discriminator"`
	Accounts      []IDLAccountMeta `json:"accounts"`
	Args          []IDLField       `json:"args"`
	Returns       *IDLType         `json:"returns,omitempty"`
	Docs          []string         `json:"docs,omitempty"`
}

// IDLAccountMeta represents an account in an instruction.
type IDLAccountMeta struct {
	Name     string   `json:"name"`
	Writable bool     `json:"writable,omitempty"`
	Signer   bool     `json:"signer,omitempty"`
	Optional bool     `json:"optional,omitempty"`
	Address  string   `json:"address,omitempty"`
	PDA      *IDLPDA  `json:"pda,omitempty"`
	Docs     []string `json:"docs,omitempty"`
}

// IDLPDA represents a Program Derived Address configuration.
type IDLPDA struct {
	Seeds   []IDLSeed `json:"seeds"`
	Program *IDLSeed  `json:"program,omitempty"`
}

// IDLSeed represents a PDA seed.
type IDLSeed struct {
	Kind    string   `json:"kind"` // "const", "account", "arg"
	Value   []byte   `json:"value,omitempty"`
	Path    string   `json:"path,omitempty"`
	Account string   `json:"account,omitempty"`
	Type    *IDLType `json:"type,omitempty"`
}

// IDLAccountDef represents an account type definition.
type IDLAccountDef struct {
	Name          string   `json:"name"`
	Discriminator []byte   `json:"discriminator"`
	Type          IDLType  `json:"type,omitempty"`
	Docs          []string `json:"docs,omitempty"`
}

// IDLEvent represents a program event.
type IDLEvent struct {
	Name          string     `json:"name"`
	Discriminator []byte     `json:"discriminator"`
	Fields        []IDLField `json:"fields,omitempty"`
	Docs          []string   `json:"docs,omitempty"`
}

// IDLError represents a program error.
type IDLError struct {
	Code int    `json:"code"`
	Name string `json:"name"`
	Msg  string `json:"msg,omitempty"`
}

// IDLTypeDef represents a custom type definition.
type IDLTypeDef struct {
	Name     string       `json:"name"`
	Docs     []string     `json:"docs,omitempty"`
	Type     IDLType      `json:"type"`
	Generics []IDLGeneric `json:"generics,omitempty"`
}

// IDLGeneric represents a generic type parameter.
type IDLGeneric struct {
	Kind string `json:"kind"` // "type", "const"
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// IDLType represents a type in the IDL.
// It can be a primitive type, a defined type, or a complex type (struct, enum, array, etc.)
type IDLType struct {
	// For primitive types, this is the type name directly
	Kind string `json:"kind,omitempty"`

	// For complex types
	Defined *IDLDefinedType `json:"defined,omitempty"`
	Option  *IDLType        `json:"option,omitempty"`
	Coption *IDLType        `json:"coption,omitempty"`
	Vec     *IDLType        `json:"vec,omitempty"`
	Array   *IDLArrayType   `json:"array,omitempty"`
	Struct  *IDLStructType  `json:"struct,omitempty"`
	Enum    *IDLEnumType    `json:"enum,omitempty"`
	Tuple   []IDLType       `json:"tuple,omitempty"`
}

// IDLDefinedType references a defined type.
type IDLDefinedType struct {
	Name     string    `json:"name"`
	Generics []IDLType `json:"generics,omitempty"`
}

// IDLArrayType represents a fixed-size array.
type IDLArrayType struct {
	Type IDLType `json:"type"`
	Len  int     `json:"len"`
}

// IDLStructType represents a struct type.
type IDLStructType struct {
	Fields []IDLField `json:"fields"`
}

// IDLEnumType represents an enum type.
type IDLEnumType struct {
	Variants []IDLEnumVariant `json:"variants"`
}

// IDLEnumVariant represents an enum variant.
type IDLEnumVariant struct {
	Name   string     `json:"name"`
	Fields []IDLField `json:"fields,omitempty"`
}

// IDLField represents a field in a struct, event, or instruction.
type IDLField struct {
	Name string   `json:"name"`
	Type IDLType  `json:"type"`
	Docs []string `json:"docs,omitempty"`
}

// IDLConstant represents a constant definition.
type IDLConstant struct {
	Name  string  `json:"name"`
	Type  IDLType `json:"type"`
	Value string  `json:"value"`
}
