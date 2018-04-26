package main

import (
	"fmt"
	"github.com/egnyte/ax/pkg/complete"
	"github.com/egnyte/ax/pkg/config"
	"github.com/spf13/cobra"
)

var (
	completionCommand = &cobra.Command{
		Use:    "complete",
		Hidden: true,
	}
)

func init() {
	completionCommand.AddCommand(&cobra.Command{
		Use: "env",
		Run: func(cmd *cobra.Command, args []string) {
			config := config.LoadConfig()
			for envName := range config.Environments {
				fmt.Println(envName)
			}
		},
	})
	var suffixOperatorFlag string
	attrsCommand := &cobra.Command{
		Use:  "attrs",
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			defaultEnv := ""
			if len(args) > 0 {
				defaultEnv = args[0]
			}
			rc := config.BuildConfig(defaultEnv, "")
			for completion := range complete.GetCompletions(rc) {
				fmt.Printf("%s%s\n", completion, suffixOperatorFlag)
			}
		},
	}
	attrsCommand.Flags().StringVar(&suffixOperatorFlag, "suffix", "", "Suffix to use")
	completionCommand.AddCommand(attrsCommand)
}
