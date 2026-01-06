package codegen

import (
	"encoding/json"
	"fmt"
	"os"
)

func ParseIDLFile(filePath string) (*IDL, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read IDL file: %w", err)
	}

	return ParseIDL(data)
}

func ParseIDL(data []byte) (*IDL, error) {
	var idl IDL
	if err := json.Unmarshal(data, &idl); err != nil {
		return nil, fmt.Errorf("failed to parse IDL JSON: %w", err)
	}

	return &idl, nil
}
