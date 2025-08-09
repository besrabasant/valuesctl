package cmd

import (
	"fmt"

	"github.com/besrabasant/valuesctl/internal/fileutil"
	"github.com/besrabasant/valuesctl/internal/patcher"
	"github.com/besrabasant/valuesctl/internal/schema"
	"github.com/besrabasant/valuesctl/internal/tmpl"
	"github.com/spf13/cobra"
)

var (
	filePath   string
	outPath    string
	cfgPath    string
	tplPath    string
	backup     bool
	schemaPath string
	validate   bool
)
func init() {
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Patch an existing values.yaml using template + config (schema-first; opt-in defaults)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Optional: validate config against schema
			if validate && schemaPath != "" {
				if err := schema.ValidateYAMLWithSchema(schemaPath, cfgPath); err != nil {
					return fmt.Errorf("config validation failed: %w", err)
				}
			}

			// 1) read old values
			oldYAML, err := fileutil.ReadFile(filePath)
			if err != nil {
				return err
			}

			// 2) load config as map, optionally applying schema defaults
			data, err := schema.LoadConfigWithSchemaDefaults(schemaPath, cfgPath, schemaPath != "")
			if err != nil {
				return err
			}

			// 3) render desired from template + (data map)
			desiredYAML, err := tmpl.RenderWithData(tplPath, data)
			if err != nil {
				return err
			}

			// 4) compute merge patch & apply
			newYAML, err := patcher.MergePatchYAML(oldYAML, desiredYAML)
			if err != nil {
				return err
			}

			// 5) write output (in place by default) with optional backup
			target := outPath
			if target == "" {
				target = filePath
				if backup {
					if err := fileutil.WriteFileAtomic(filePath+".bak", oldYAML); err != nil {
						return fmt.Errorf("write backup: %w", err)
					}
				}
			}
			return fileutil.WriteFileAtomic(target, newYAML)
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "values.yaml", "path to existing values.yaml (also default output)")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "optional different output path (default: in-place)")
	cmd.Flags().StringVarP(&cfgPath, "config", "c", "config.yaml", "path to config.yaml")
	cmd.Flags().StringVarP(&tplPath, "template", "t", "template.tmpl", "path to Go text/template for values.yaml")
	cmd.Flags().BoolVar(&backup, "backup", true, "write a .bak beside --file before in-place update")

	cmd.Flags().StringVarP(&schemaPath, "schema", "s", "", "optional JSON Schema (YAML or JSON) for validation/defaults")
	cmd.Flags().BoolVar(&validate, "validate", false, "validate --config against --schema before patching")

	rootCmd.AddCommand(cmd)
}