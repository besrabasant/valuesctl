package cmd

import (
	"github.com/besrabasant/valuesctl/internal/fileutil"
	"github.com/besrabasant/valuesctl/internal/schema"
	"github.com/spf13/cobra"
)

var sampleOut string

func init() {
	cmd := &cobra.Command{
		Use:   "gen-sample",
		Short: "Generate a sample config from a YAML JSON-Schema (types & descriptions respected)",
		RunE: func(cmd *cobra.Command, args []string) error {
			y, err := schema.BuildSampleFromSchema(schemaPath)
			if err != nil {
				return err
			}
			return fileutil.WriteFileAtomic(sampleOut, y)
		},
	}

	cmd.Flags().StringVarP(&schemaPath, "schema", "s", "config.schema.yaml", "path to YAML JSON-Schema")
	cmd.Flags().StringVarP(&sampleOut, "out", "o", "config.sample.yaml", "output sample config path")

	rootCmd.AddCommand(cmd)
}
