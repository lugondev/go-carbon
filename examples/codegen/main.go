package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	fmt.Println("=== Go-Carbon Code Generation Example ===")
	fmt.Println()

	exampleDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	idlPath := filepath.Join(exampleDir, "examples", "codegen", "sample_idl.json")
	if _, err := os.Stat(idlPath); os.IsNotExist(err) {
		idlPath = filepath.Join(exampleDir, "sample_idl.json")
	}

	outputDir := filepath.Join(exampleDir, "examples", "codegen", "generated", "tokenswap")
	if _, err := os.Stat(filepath.Join(exampleDir, "examples")); os.IsNotExist(err) {
		outputDir = filepath.Join(exampleDir, "generated", "tokenswap")
	}

	fmt.Printf("IDL Path: %s\n", idlPath)
	fmt.Printf("Output Dir: %s\n", outputDir)
	fmt.Println()

	fmt.Println("Running: carbon codegen --idl sample_idl.json --output ./generated/tokenswap")
	fmt.Println()

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "run", "../../cmd/carbon/main.go",
		"codegen",
		"--idl", idlPath,
		"--output", outputDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Code generation failed: %v", err)
	}

	fmt.Println()
	fmt.Println("=== Usage Example ===")
	fmt.Println()
	fmt.Println("After generation, you can use the generated code like this:")
	fmt.Println(`
package main

import (
    "context"
    "fmt"
    
    "github.com/gagliardetto/solana-go"
    "github.com/lugondev/go-carbon/examples/codegen/generated/tokenswap"
    "github.com/lugondev/go-carbon/pkg/plugin"
)

func main() {
    // Create plugin registry
    registry := plugin.NewRegistry()
    
    // Register the generated plugin
    programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
    registry.MustRegister(tokenswap.NewTokenSwapPlugin(programID))
    
    // Initialize
    ctx := context.Background()
    registry.Initialize(ctx)
    
    // Get decoder registry for manual decoding
    decoderRegistry := tokenswap.GetDecoderRegistry(programID)
    
    // Decode events from transaction logs
    // events, _ := decoderRegistry.DecodeAll(programDataList, &programID)
    
    fmt.Println("Plugin registered successfully!")
    _ = decoderRegistry
}
`)
}
