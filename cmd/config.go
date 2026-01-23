package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.GetConfigPath()
		if err != nil {
			return err
		}

		editor := appConfig.Editor
		if editor == "" {
			editor = os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim"
			}
		}

		editorArgs := append(strings.Fields(editor), cfgPath)

		editorCmd := exec.Command(editorArgs[0], editorArgs[1:]...)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("Error opening editor: %v", err)
		}
		return nil
	},
}
