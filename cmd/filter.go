package cmd

import (
	"fmt"
	"log"

	"github.com/Salvadego/IntraCLI/config"
	"github.com/spf13/cobra"
)

var (
	filterSaveName   string
	filterDeleteName string
	filterList       bool
)

func init() {
	filterTimesheetsCmd.Flags().StringVar(&filterSaveName, "save", "", "Save the query with the given name")
	filterTimesheetsCmd.Flags().BoolVar(&filterList, "list", false, "List saved filters")
	filterTimesheetsCmd.Flags().StringVar(&filterDeleteName, "delete", "", "Delete the named filter")

	filterTimesheetsCmd.RegisterFlagCompletionFunc("delete", filterNameCompletionFunc)

	rootCmd.AddCommand(filterTimesheetsCmd)
}

// filterTimesheetsCmd manages saved qlvm query strings for timesheet filtering.
//
// Saving a filter:
//
//	intracli filter-timesheets --save myproj "project = myproj AND date >= 2025-07-01"
//
// Using a saved filter elsewhere:
//
//	intracli list-timesheets  --filter myproj
//	intracli date-summary     --filter-timesheet myproj
//	intracli delete-timesheet --filter myproj
//	intracli edit-timesheet   --filter myproj
//
// Query language quick-reference (qlvm):
//
//	ticket = "INC-123"       exact ticket match
//	.ticket = INC            ticket contains "INC"
//	#ticket = INC-123        alias for exact ticket match
//	has_ticket?              only entries with a ticket
//	date >= 2025-07-01       date comparison (YYYY-MM-DD)
//	hours > 4
//	project = myalias        matches the project alias defined in your profile
//	type = Normal
//	description = "bug fix"
var filterTimesheetsCmd = &cobra.Command{
	Use:   "filter-timesheets [query]",
	Short: "Manage saved timesheet filters (qlvm query strings)",
	Long: `Save, list, or delete named qlvm query strings used to filter timesheets.

The optional positional argument is the raw qlvm query string to save.

Examples:
  intracli filter-timesheets --save myproj "project = myproj"
  intracli filter-timesheets --save tickets "has_ticket?"
  intracli filter-timesheets --save recent "date >= 2025-07-01 AND hours > 0"
  intracli filter-timesheets --list
  intracli filter-timesheets --delete myproj`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.InitializeConfig()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		if filterList {
			if len(cfg.SavedFilters) == 0 {
				fmt.Println("No saved filters.")
				return
			}
			fmt.Println("Saved timesheet filters:")
			for name, q := range cfg.SavedFilters {
				fmt.Printf("  %-20s  %s\n", name, q)
			}
			return
		}

		if filterDeleteName != "" {
			if _, ok := cfg.SavedFilters[filterDeleteName]; !ok {
				log.Fatalf("Filter '%s' not found.", filterDeleteName)
			}
			delete(cfg.SavedFilters, filterDeleteName)
			if err := config.SaveConfig(cfg); err != nil {
				log.Fatalf("Failed to save config: %v", err)
			}
			fmt.Printf("Filter '%s' deleted.\n", filterDeleteName)
			return
		}

		if filterSaveName == "" {
			fmt.Println("No action taken. Use --save <name> [query], --list, or --delete <name>.")
			return
		}

		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		if cfg.SavedFilters == nil {
			cfg.SavedFilters = make(map[string]string)
		}
		cfg.SavedFilters[filterSaveName] = query
		if err := config.SaveConfig(cfg); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		fmt.Printf("Filter '%s' saved: %q\n", filterSaveName, query)
	},
}
