package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lugondev/go-carbon/internal/codegen"
	"github.com/spf13/cobra"
)

var (
	idlPath     string
	outputDir   string
	packageName string
)

var codegenCmd = &cobra.Command{
	Use:   "codegen",
	Short: "Generate Go code from Anchor IDL",
	Long: `Generate Go structs, decoders, and plugins from an Anchor IDL JSON file.

Example:
  carbon codegen --idl ./target/idl/my_program.json --output ./generated/myprogram
  carbon codegen -i idl.json -o ./pkg/myprogram -p myprogram`,
	RunE: runCodegen,
}

func init() {
	rootCmd.AddCommand(codegenCmd)

	codegenCmd.Flags().StringVarP(&idlPath, "idl", "i", "", "Path to Anchor IDL JSON file (required)")
	codegenCmd.Flags().StringVarP(&outputDir, "output", "o", "./generated", "Output directory for generated code")
	codegenCmd.Flags().StringVarP(&packageName, "package", "p", "", "Go package name (defaults to program name from IDL)")

	if err := codegenCmd.MarkFlagRequired("idl"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag required: %v\n", err)
	}
}

func runCodegen(cmd *cobra.Command, args []string) error {
	absIDLPath, err := filepath.Abs(idlPath)
	if err != nil {
		return fmt.Errorf("failed to resolve IDL path: %w", err)
	}

	if _, err := os.Stat(absIDLPath); os.IsNotExist(err) {
		return fmt.Errorf("IDL file not found: %s", absIDLPath)
	}

	idl, err := codegen.ParseIDLFile(absIDLPath)
	if err != nil {
		return fmt.Errorf("failed to parse IDL: %w", err)
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	generator := codegen.NewGenerator(idl, packageName, absOutputDir)

	fmt.Printf("Generating code from IDL: %s\n", absIDLPath)
	fmt.Printf("  Program: %s (v%s)\n", idl.Metadata.Name, idl.Metadata.Version)
	fmt.Printf("  Address: %s\n", idl.Address)
	fmt.Printf("  Output:  %s\n", absOutputDir)
	fmt.Printf("  Package: %s\n", generator.PackageName)
	fmt.Println()

	if err := generator.Generate(); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	fmt.Println("Generated files:")
	fmt.Printf("  - %s/program.go\n", absOutputDir)
	if len(idl.Types) > 0 {
		fmt.Printf("  - %s/types.go (%d types)\n", absOutputDir, len(idl.Types))
	}
	if len(idl.Accounts) > 0 {
		fmt.Printf("  - %s/accounts.go (%d accounts)\n", absOutputDir, len(idl.Accounts))
	}
	if len(idl.Events) > 0 {
		fmt.Printf("  - %s/events.go (%d events)\n", absOutputDir, len(idl.Events))
	}
	if len(idl.Instructions) > 0 {
		fmt.Printf("  - %s/instructions.go (%d instructions)\n", absOutputDir, len(idl.Instructions))
	}

	fmt.Println("\nCode generation complete!")
	return nil
}
