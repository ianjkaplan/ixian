package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/iankaplan/ixian/internal/binder"
	"github.com/iankaplan/ixian/internal/checker"
	"github.com/iankaplan/ixian/internal/emitter"
	"github.com/iankaplan/ixian/internal/formatter"
	"github.com/iankaplan/ixian/internal/parser"
	"github.com/iankaplan/ixian/internal/planner"
)

var (
	specPath  string
	outputDir string
)

var rootCmd = &cobra.Command{
	Use:   "ixian",
	Short: "Generate a Go CLI from an OpenAPI 3.x spec",
	Long:  "Ixian reads an OpenAPI 3.x specification and generates a fully functional Go CLI application using Cobra.",
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVarP(&specPath, "spec", "s", "", "Path to OpenAPI spec file (required)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "./output", "Output directory for generated code")
	_ = rootCmd.MarkFlagRequired("spec")
}

func run(cmd *cobra.Command, args []string) error {
	// 1. Parse
	fmt.Fprintf(os.Stderr, "Parsing %s...\n", specPath)
	spec, err := parser.Parse(specPath)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// 2. Bind
	fmt.Fprintln(os.Stderr, "Binding references...")
	bound, err := binder.Bind(spec)
	if err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	// 3. Check
	fmt.Fprintln(os.Stderr, "Checking spec...")
	result := checker.Check(bound)
	for _, d := range result.Warnings() {
		fmt.Fprintf(os.Stderr, "  %s\n", d)
	}
	if result.HasErrors() {
		for _, d := range result.Errors() {
			fmt.Fprintf(os.Stderr, "  %s\n", d)
		}
		return fmt.Errorf("spec validation failed with %d error(s)", len(result.Errors()))
	}

	// 4. Plan
	fmt.Fprintln(os.Stderr, "Planning codegen...")
	plan := planner.Plan(bound)

	// 5. Emit
	fmt.Fprintln(os.Stderr, "Emitting source files...")
	files, err := emitter.Emit(plan)
	if err != nil {
		return fmt.Errorf("emit: %w", err)
	}

	// 6. Format
	fmt.Fprintln(os.Stderr, "Formatting...")
	files, err = formatter.Format(files)
	if err != nil {
		return fmt.Errorf("format: %w", err)
	}

	// 7. Write
	fmt.Fprintf(os.Stderr, "Writing to %s...\n", outputDir)
	if err := formatter.Write(outputDir, files); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Done! Generated %d files.\n", len(files))
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
