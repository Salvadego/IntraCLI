package cmd

import (
	"fmt"
	"os"

	"github.com/Salvadego/IntraCLI/cache"
	"github.com/spf13/cobra"
)

type CleanFile string

const (
	Timesheet CleanFile = "timesheets"
	Employee  CleanFile = "employees"
	Contracts CleanFile = "contracts"
	Tickets   CleanFile = "tickets"
	All       CleanFile = "all"
)

var cleanFileValues = []string{
	string(Timesheet),
	string(Employee),
	string(Contracts),
	string(Tickets),
	string(All),
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}

func deleteByCleanType(f CleanFile) error {
	switch f {
	case Timesheet:
		files, err := cache.ListCacheFiles("timesheets")
		if err != nil {
			return err
		}

		for _, file := range files {
			file, err := cache.GetCacheFilePath(file)
			if err != nil {
				return err
			}

			err = os.Remove(file)
			if err != nil {
				return err
			}
		}

	case Tickets:
		files, err := cache.ListCacheFiles("tickets_")
		if err != nil {
			return err
		}

		for _, file := range files {
			file, err := cache.GetCacheFilePath(file)
			if err != nil {
				return err
			}

			err = os.Remove(file)
			if err != nil {
				return err
			}
		}
	case Employee:
		if err := os.Remove(cache.EmployeeListCacheFileName); err != nil {
			return err
		}
	case Contracts:
		if err := os.Remove(cache.ContractsListCacheFileName); err != nil {
			return err
		}

	case All:
		filepath, err := cache.GetCacheDirPath()
		if err != nil {
			return err
		}
		return os.RemoveAll(filepath)

	default:
		return fmt.Errorf("Invalid Clean Type")
	}

	return nil
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean cache files from the cache directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteByCleanType(CleanFile(args[0]))
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var completions []string
		completions = append(completions, cleanFileValues...)
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
}
