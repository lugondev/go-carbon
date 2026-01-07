package codegen

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestParseDefinedType(t *testing.T) {
	idlJSON := `{
		"kind": "defined",
		"name": "swap_state"
	}`
	
	var typ IDLType
	if err := json.Unmarshal([]byte(idlJSON), &typ); err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	
	fmt.Printf("Type kind: %s\n", typ.Kind)
	if typ.Defined != nil {
		fmt.Printf("Defined type name: %s\n", typ.Defined.Name)
	} else {
		t.Error("Defined is nil!")
	}
}
