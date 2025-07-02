package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Salvadego/IntraCLI/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "intracli",
	Short: "IntraCLI is a CLI tool for managing Mantis timesheets.",
	Long:  `A simple CLI tool to interact with the Mantis API.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() != "completion" && cmd.Name() != "intracli" {
			return initCommonMantisClient(cmd)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&profileName, "profile", "P", "", "Profile to use (overrides default)")

	rootCmd.RegisterFlagCompletionFunc("profile",
		func(
			cmd *cobra.Command,
			args []string,
			toComplete string,
		) ([]string, cobra.ShellCompDirective) {
			cfg, err := config.LoadConfig()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var profiles []string
			for p := range cfg.Profiles {
				if strings.HasPrefix(p, toComplete) {
					profiles = append(profiles, p)
				}
			}
			return profiles, cobra.ShellCompDirectiveNoFileComp
		})
}
