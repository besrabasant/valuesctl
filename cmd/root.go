package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "valuesctl",
	Short: "Patch Helm values.yaml from template+config; generate config samples from schema",
	Long:  "Schema-first tool: generate sample configs from a YAML JSON-Schema; validate & render template with config; JSON-merge-patch onto existing values.yaml, atomic in-place by default.",
}


func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}